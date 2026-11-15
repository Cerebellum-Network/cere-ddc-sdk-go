package auth

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/mr-tron/base58"
	"google.golang.org/protobuf/proto"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb"
)

// DefaultMaxChainDepth bounds how many delegation levels a chain may carry.
// Real chains are one level (a bearer token signed by the bucket owner) or two
// (an owner grant to a service, and a leaf the service signs for a derived
// key). The bound stops an attacker-supplied deeply nested token from driving
// unbounded recursion.
const DefaultMaxChainDepth = 8

// DefaultMaxEncodedTokenLength bounds the encoded credential in characters,
// before any decoding is attempted.
//
// This is a DoS control, not a correctness one. base58 decoding is quadratic in
// the input length and runs BEFORE any signature is checked, so an unbounded
// credential is a pre-authentication CPU amplifier: net/http accepts a 1 MiB
// header by default, and decoding one costs minutes of a core. Do not raise this.
//
// 4096 is ~10x the longest real credential (a two-level agent chain is ~420
// characters, a one-level developer bearer ~161).
const DefaultMaxEncodedTokenLength = 4096

// Errors returned by Authorize. Every one of them is a rejection: there is no
// path through this package that allows a request on an error.
var (
	ErrTokenRequired         = errors.New("token is required")
	ErrInvalidToken          = errors.New("token is invalid")
	ErrTokenTooLong          = errors.New("encoded token exceeds maximum length")
	ErrInvalidSignature      = errors.New("signature is invalid")
	ErrUnsupportedAlgorithm  = errors.New("unsupported signature algorithm")
	ErrInvalidExpirationTime = errors.New("invalid expiration time")
	ErrExpired               = errors.New("token expired")
	ErrDelegatedToken        = errors.New("token delegation is not allowed")
	ErrChainTooDeep          = errors.New("token chain exceeds maximum depth")
	ErrInvalidPermissions    = errors.New("token permissions are invalid")
	ErrInvalidIssuer         = errors.New("issuer doesn't match subject in parent token")
	ErrInvalidOwnership      = errors.New("bucket belongs to other owner")
	ErrInsufficientBalance   = errors.New("bucket owner has insufficient balance")
	ErrBucketRemoved         = errors.New("bucket has been removed")
	ErrBucketMismatch        = errors.New("token bucket does not match requested bucket")
	ErrPieceScopedToken      = errors.New("piece-scoped token cannot authorize a bucket-level operation")
	ErrUnknownOperation      = errors.New("requested operation is unknown")
	ErrOwnerLookup           = errors.New("bucket owner lookup failed")
)

// Kind classifies a rejection so a transport can shape a response without
// pattern-matching on individual sentinels.
type Kind uint8

const (
	// KindUnauthenticated means the credential itself is not credible: it is
	// absent, undecodable, expired, badly signed, or the delegation chain does
	// not hold together. 401-shaped.
	KindUnauthenticated Kind = iota + 1

	// KindForbidden means the credential is credible but does not carry the
	// authority claimed: it does not root in the bucket owner or one of its
	// honored delegates, the owner cannot pay, the bucket is gone, or the
	// operation is not granted. 403-shaped.
	KindForbidden

	// KindUnavailable means authority could not be established because the
	// bucket owner index could not answer. The request is still rejected — this
	// kind exists so operators can tell an outage apart from a forgery.
	// 503-shaped.
	KindUnavailable
)

func (k Kind) String() string {
	switch k {
	case KindUnauthenticated:
		return "unauthenticated"
	case KindForbidden:
		return "forbidden"
	case KindUnavailable:
		return "unavailable"
	default:
		return "unknown"
	}
}

// HTTPStatus maps a Kind onto the status code callers should return. An
// unrecognised Kind maps to 403 so an unhandled case still denies.
func (k Kind) HTTPStatus() int {
	switch k {
	case KindUnauthenticated:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusForbidden
	}
}

// Error is the typed rejection every Authorize failure is wrapped in. The
// underlying sentinel stays reachable through errors.Is.
type Error struct {
	Kind Kind
	Err  error
}

func (e *Error) Error() string {
	return e.Kind.String() + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

// KindOf reports how err should be shaped by a transport. Anything that is not
// an authz rejection — including a nil error, which no caller should be asking
// about — reports KindForbidden, so a mis-wired caller denies rather than
// allows.
func KindOf(err error) Kind {
	var typed *Error
	if errors.As(err, &typed) {
		return typed.Kind
	}
	return KindForbidden
}

// forbiddenErrors are the sentinels that mean "credible credential, insufficient
// authority". Everything else this package produces is a credential problem.
var forbiddenErrors = []error{
	ErrInvalidPermissions,
	ErrInvalidOwnership,
	ErrInsufficientBalance,
	ErrBucketRemoved,
	ErrBucketMismatch,
	ErrPieceScopedToken,
	ErrUnknownOperation,
}

// reject wraps err in a typed rejection, classifying it by sentinel so the Kind
// does not depend on which phase of the check produced it. An unrecognised
// error — including one bubbled out of the crypto layer — classifies as
// KindUnauthenticated, which still denies.
func reject(err error) *Error {
	if errors.Is(err, ErrOwnerLookup) {
		return &Error{Kind: KindUnavailable, Err: err}
	}
	for _, sentinel := range forbiddenErrors {
		if errors.Is(err, sentinel) {
			return &Error{Kind: KindForbidden, Err: err}
		}
	}
	return &Error{Kind: KindUnauthenticated, Err: err}
}

// Verified is the result of a successful authorization. Every field is derived
// from a signature-checked chain that roots in the bucket owner or one of the
// owner's honored delegates.
type Verified struct {
	// RootIssuer signed the root of the chain, and has been confirmed to be the
	// bucket owner, or an honored proxy of the owner acting as root signer (see
	// LookupResult.Delegates). Billing never moves: the balance asserted is
	// still the owner's.
	RootIssuer ed25519.PublicKey

	BucketID uint64

	// Subject is the leaf's subject: on a two-level agent chain the derived
	// execution key, which callers use as the job-binding handle. Nil for a
	// one-level bearer token, which names no subject.
	Subject ed25519.PublicKey

	// ExpiresAt is the leaf's expiry (unix seconds), the tightest bound on the
	// chain: the walk rejects any level that outlives its parent.
	ExpiresAt int64

	// Operations is the root level's operation set. Given the (inverted)
	// attenuation check the chain enforces, the root set is contained in every
	// descendant's, so it is the narrowest grant and the correct one to authorize
	// against.
	Operations []pb.Operation
}

// BucketOwnerIndex answers who owns a bucket and whether they can pay.
//
// Implementations must fail closed: return an error rather than a zero-value
// LookupResult when the underlying source is unavailable. A zero LookupResult
// with a nil error is read as "no owner, no balance" and denies, but an
// implementation must not rely on that — an error is the contract.
type BucketOwnerIndex interface {
	Lookup(ctx context.Context, bucketID uint64) (LookupResult, error)
}

// LookupResult is what a BucketOwnerIndex knows about one bucket.
type LookupResult struct {
	// Owner is the account that owns the bucket, as a raw 32-byte ed25519
	// public key. Empty when the bucket has no known owner.
	Owner ed25519.PublicKey

	// HasBalance reports whether the owner can pay for the work being
	// authorized. False denies.
	HasBalance bool

	// Removed reports whether the bucket has been removed on chain. True
	// denies.
	Removed bool

	// Delegates lists the owner's proxy delegates whose on-chain grants are
	// honored for DDC writes, as raw 32-byte ed25519 public keys: the accounts
	// that may stand in for Owner. Nil or empty means none are known, and
	// delegation is denied rather than assumed — which is also what an
	// implementation that predates this field reports, so a mixed fleet fails
	// closed.
	Delegates [][]byte
}

// Authorizer verifies an encoded AuthToken chain against a requested operation
// and payer bucket.
type Authorizer interface {
	// Authorize walks the chain, resolves the root issuer, and asserts the
	// root is bucketID's owner — or one of the owner's honored delegates —
	// that the owner has positive balance, and that the chain grants op.
	Authorize(ctx context.Context, encodedToken string, op pb.Operation, bucketID uint64) (*Verified, error)
}

// Options tunes an Authorizer. The zero value selects the defaults.
type Options struct {
	// MaxChainDepth overrides DefaultMaxChainDepth when > 0.
	MaxChainDepth int

	// MaxEncodedTokenLength overrides DefaultMaxEncodedTokenLength when > 0. Read
	// that constant's doc before raising it: decoding cost grows quadratically.
	MaxEncodedTokenLength int

	// Now overrides the clock used for expiry checks. Test seam.
	Now func() time.Time
}

type authorizer struct {
	index     BucketOwnerIndex
	maxDepth  int
	maxLength int
	now       func() time.Time
}

// New returns an Authorizer backed by index, using the default options.
func New(index BucketOwnerIndex) Authorizer {
	return NewWithOptions(index, Options{})
}

// NewWithOptions returns an Authorizer backed by index with opts applied.
func NewWithOptions(index BucketOwnerIndex, opts Options) Authorizer {
	maxDepth := opts.MaxChainDepth
	if maxDepth <= 0 {
		maxDepth = DefaultMaxChainDepth
	}
	maxLength := opts.MaxEncodedTokenLength
	if maxLength <= 0 {
		maxLength = DefaultMaxEncodedTokenLength
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &authorizer{index: index, maxDepth: maxDepth, maxLength: maxLength, now: now}
}

// Authorize implements Authorizer.
func (a *authorizer) Authorize(
	ctx context.Context,
	encodedToken string,
	op pb.Operation,
	bucketID uint64,
) (*Verified, error) {
	if op == pb.Operation_UNKNOWN {
		return nil, reject(ErrUnknownOperation)
	}
	// A bucket-less request has nothing to bill and nothing to bind an owner
	// to, so there is no owner assertion to make. Reject rather than fall
	// through to a chain check that would authorize on signature alone.
	if bucketID == 0 {
		return nil, reject(ErrBucketMismatch)
	}
	if encodedToken == "" {
		return nil, reject(ErrTokenRequired)
	}
	// Length is checked before anything looks at the bytes: decoding is
	// quadratic, so this is the only cheap place to stop an oversized
	// credential. See DefaultMaxEncodedTokenLength.
	if len(encodedToken) > a.maxLength {
		return nil, reject(fmt.Errorf("%w: %d characters, limit %d",
			ErrTokenTooLong, len(encodedToken), a.maxLength))
	}
	// No index means no owner assertion is possible. Fail closed.
	if a.index == nil {
		return nil, reject(fmt.Errorf("%w: no bucket owner index configured", ErrOwnerLookup))
	}

	token, _, err := decodeToken(encodedToken, a.maxLength)
	if err != nil {
		return nil, reject(err)
	}

	expanded := &expandedToken{}
	if err := a.expandAndVerifyToken(token, expanded, 0); err != nil {
		return nil, reject(err)
	}

	if err := verifyRequestedAction(expanded, op, bucketID); err != nil {
		return nil, reject(err)
	}

	result, err := a.index.Lookup(ctx, bucketID)
	if err != nil {
		// A lookup failure is a rejection, never an allow.
		return nil, reject(fmt.Errorf("%w: %w", ErrOwnerLookup, err))
	}
	if result.Removed {
		return nil, reject(ErrBucketRemoved)
	}
	// The root signer must be the bucket owner or one of the owner's honored
	// proxy delegates, as vouched for by the index. This is the check that
	// kills a self-signed token naming somebody else's bucket: the forger's
	// key is the root issuer, and it is neither.
	root := expanded.rootIssuer
	isOwner := len(result.Owner) > 0 && bytes.Equal(result.Owner, root)
	isDelegate := !isOwner && slices.ContainsFunc(result.Delegates, func(d []byte) bool {
		return bytes.Equal(d, root)
	})
	if len(result.Owner) == 0 || (!isOwner && !isDelegate) {
		return nil, reject(ErrInvalidOwnership)
	}
	if !result.HasBalance {
		return nil, reject(ErrInsufficientBalance)
	}

	verified := &Verified{
		RootIssuer: ed25519.PublicKey(bytes.Clone(expanded.rootIssuer)),
		BucketID:   bucketID,
		ExpiresAt:  expanded.leafExpiresAt,
		Operations: slices.Clone(expanded.operations),
	}
	if subject := token.GetPayload().GetSubject(); len(subject) > 0 {
		verified.Subject = ed25519.PublicKey(bytes.Clone(subject))
	}
	return verified, nil
}

// expandedToken accumulates state while walking the chain leaf -> root.
type expandedToken struct {
	// rootIssuer holds the signer of the most recently visited level. When the
	// walk finishes it holds the root level's signer, which is the identity the
	// bucket-owner assertion runs against. Mid-walk it doubles as "the child's
	// signer", which the next level's Subject must equal.
	rootIssuer []byte

	// expiresAt holds the most recently visited level's expiry, used to reject
	// a child that outlives its parent.
	expiresAt int64

	// leafExpiresAt is the leaf's expiry, the effective lifetime of the grant.
	leafExpiresAt int64

	bucketID   uint64
	pieceCid   []byte
	operations []pb.Operation
}

// expandAndVerifyToken walks one level of the chain and recurses into its
// parent. depth 0 is the leaf (the token actually presented); depth grows
// towards the root.
//
// The walk is behaviour-identical to the DDC storage node's, with two additions
// that are strict rejections and cannot widen access: a chain-depth bound and a
// signature sanity check (see signatureUsable).
func (a *authorizer) expandAndVerifyToken(token *pb.AuthToken, expanded *expandedToken, depth int) error {
	if depth >= a.maxDepth {
		return ErrChainTooDeep
	}

	if !fieldsValid(token) {
		return ErrInvalidToken
	}

	// A nil/zero ExpiresAt reads as expired: expiry is mandatory at every
	// level, and must be in the future.
	if a.isExpired(token) {
		return ErrExpired
	}

	// Both sides normalised — the fleet does not agree on the unit, so comparing
	// raw values could pit a millisecond child against a second-based parent.
	levelExpiresAt := expiresAtSeconds(token)
	if expanded.expiresAt != 0 && expanded.expiresAt > levelExpiresAt {
		// the more recent (child) token expires after its parent
		return ErrInvalidExpirationTime
	}

	expanded.expiresAt = levelExpiresAt
	if depth == 0 {
		expanded.leafExpiresAt = expanded.expiresAt
	}

	// Reject signatures the verifier cannot use before any crypto runs.
	if err := signatureUsable(token.Signature); err != nil {
		return err
	}

	verified, err := VerifySignature(token)
	if err != nil {
		return err
	}
	if !verified {
		return ErrInvalidSignature
	}

	if len(token.Payload.Subject) == 0 {
		if depth != 0 {
			// got a token which wasn't delegated
			return ErrDelegatedToken
		}
	}

	// PRESERVED QUIRK 4: depth > 1, not depth > 0. The level directly above the
	// leaf may carry canDelegate=false and still have that leaf signed beneath
	// it; only a second level of delegation trips this. Kept as the storage
	// node has it. Harmless in a bearer model (see the package doc), but it does mean
	// canDelegate=false is not a defence against a two-level chain.
	if !token.Payload.GetCanDelegate() {
		if depth > 1 {
			// got a token which couldn't be delegated
			return ErrDelegatedToken
		}
	}

	if len(expanded.rootIssuer) > 0 && !bytes.Equal(expanded.rootIssuer, token.Payload.Subject) {
		// the previous (child) level was issued by someone other than the
		// party this level delegated to
		return ErrInvalidIssuer
	}

	expanded.rootIssuer = token.Signature.GetSigner()

	if token.Payload.GetBucketId() != 0 {
		// if a bucket is mentioned
		if expanded.bucketID == 0 {
			// remember the bucket, if not remembered before
			expanded.bucketID = token.Payload.GetBucketId()
		} else if expanded.bucketID != token.Payload.GetBucketId() {
			// every level must scope to the same bucket
			return ErrInvalidPermissions
		}
	}

	// PRESERVED QUIRK 3: the guard skips the check while the accumulated set is
	// empty, so a leaf minted with no operations at all does not fail here — it
	// simply adopts whatever its parent grants. Kept as the storage node has it.
	if len(expanded.operations) > 0 && !permissionsValid(expanded.operations, token.Payload.Operations) {
		return ErrInvalidPermissions
	}

	expanded.operations = token.Payload.Operations

	if len(token.Payload.PieceCid) > 0 {
		// if a piece is mentioned
		if len(expanded.pieceCid) == 0 {
			// remember the piece, if not remembered before
			expanded.pieceCid = token.Payload.PieceCid
		} else if !bytes.Equal(expanded.pieceCid, token.Payload.PieceCid) {
			// every level must scope to the same piece
			return ErrInvalidPermissions
		}
	}

	if token.Payload.Prev == nil {
		return nil
	}

	return a.expandAndVerifyToken(token.Payload.Prev, expanded, depth+1)
}

// verifyRequestedAction checks the expanded chain against the requested
// operation and bucket. The bucket-owner assertion lives in Authorize because it
// needs the index.
func verifyRequestedAction(expanded *expandedToken, op pb.Operation, bucketID uint64) error {
	// The storage node tolerates a chain that names no bucket at all and leans
	// on the owner check alone. This package does not: a token that does not
	// name the bucket it bills is rejected.
	if expanded.bucketID == 0 || expanded.bucketID != bucketID {
		return ErrBucketMismatch
	}

	// A piece-scoped token is a narrower grant than the bucket-level operation
	// being requested, and this entry point carries no piece to match it
	// against. Honouring it is impossible, so reject rather than widen it.
	if len(expanded.pieceCid) != 0 {
		return ErrPieceScopedToken
	}

	if !slices.Contains(expanded.operations, op) {
		return ErrInvalidPermissions
	}

	return nil
}

// expiresAtMillisThreshold separates a seconds-since-epoch ExpiresAt from a
// milliseconds-since-epoch one.
//
// Seconds since epoch is ~1.8e9 today and does not reach 1e11 until the year
// 5138. Milliseconds since epoch passed 1e11 in 1973. So for the next ~3000
// years any value at or above this threshold is unambiguously milliseconds and
// anything below it is unambiguously seconds.
const expiresAtMillisThreshold = 100_000_000_000

// expiresAtSeconds normalises a token's ExpiresAt to seconds.
//
// THE UNIT IS NOT AGREED ACROSS MINTERS, and reading it wrongly is the
// difference between an expiring credential and an immortal one. The JS SDK
// (AuthToken.create) writes MILLISECONDS; Go minters write SECONDS. Comparing a
// millisecond value against a seconds clock makes the token valid for roughly
// 50,000 years.
//
// Normalising rather than picking one unit verifies both kinds of minter
// correctly. It creates no bypass: a value below the threshold is treated as
// seconds, so a far-future expiry is a large number in either reading, and the
// chain walk bounds every level by its parent's expiry regardless.
func expiresAtSeconds(token *pb.AuthToken) int64 {
	raw := token.GetPayload().GetExpiresAt()
	if raw >= expiresAtMillisThreshold {
		return raw / 1000
	}
	return raw
}

// ExpiresAtSeconds reads a token's expiry in whatever unit its minter wrote and
// returns unix SECONDS.
//
// Minters need it. Signing a level beneath an existing one means clamping to the
// parent's expiry, read through exactly the normalisation the walk uses: clamping
// in the wrong unit yields either a token that outlives its parent (rejected) or
// one that is already expired.
func ExpiresAtSeconds(token *pb.AuthToken) int64 {
	return expiresAtSeconds(token)
}

// isExpired reports whether the token's expiry has passed, in seconds, against
// the injectable clock.
//
// A missing or zero ExpiresAt yields 0 and therefore reads as EXPIRED, as on the
// storage node. An unbounded credential is exactly what an attacker wants, so
// expiry is mandatory at every level of the chain. Minters must set an explicit
// future ExpiresAt on every level.
func (a *authorizer) isExpired(token *pb.AuthToken) bool {
	return a.now().UTC().Unix() > expiresAtSeconds(token)
}

// permissionsValid is the storage node's check, unchanged.
//
// KNOWN QUIRK — the check is inverted. It is called as
// permissionsValid(childOps, parentOps), so the parameter named
// parentTokenPerm actually receives the CHILD's operations and curTokenPerm
// receives the PARENT's. The effect is that it enforces
//
//	parent.ops ⊆ child.ops
//
// which is the opposite of attenuation: a parent granting a SUPERSET of what
// the child claims is rejected. A `fullAccess` grant sitting above a
// COMPUTE-only leaf therefore fails.
//
// This is deliberately NOT fixed here: fixing it changes live storage
// behaviour. Minters sidestep it by keeping operation sets identical across
// levels, which never trips the inversion.
func permissionsValid(parentTokenPerm, curTokenPerm []pb.Operation) bool {
	if len(parentTokenPerm) == 0 {
		return false
	}

	parentPerm := make(map[pb.Operation]bool, len(parentTokenPerm))
	for _, p := range parentTokenPerm {
		parentPerm[p] = true
	}

	for _, p := range curTokenPerm {
		if !parentPerm[p] {
			// the current token carries permissions its parent does not
			return false
		}
	}
	return true
}

// fieldsValid checks the structural minimum: every level needs a payload and a
// signature.
func fieldsValid(token *pb.AuthToken) bool {
	return token != nil && token.Signature != nil && token.Payload != nil
}

// signatureUsable rejects a signature the chain walk cannot use: an unknown
// algorithm, a signer that is not a 32-byte public key, or an empty value. All
// three are attacker-controlled fields.
func signatureUsable(sig *pb.Signature) error {
	switch sig.GetAlgorithm() {
	case pb.Signature_ED_25519, pb.Signature_SR_25519:
	default:
		return ErrUnsupportedAlgorithm
	}
	// Both supported schemes use 32-byte public keys.
	if len(sig.GetSigner()) != ed25519.PublicKeySize {
		return ErrInvalidSignature
	}
	if len(sig.GetValue()) == 0 {
		return ErrInvalidSignature
	}
	return nil
}

// decoders are tried in order. base58 is DDC-native: the JS SDK mints it and the
// storage nodes speak it. base64 (standard) is also accepted, because some
// compute services emit it.
//
// Only these two exist in the fleet, and adding speculative alphabets buys nothing:
// a wrong alphabet essentially never unmarshals into a token carrying both a payload
// and a signature, and one that did would still have to bear a valid signature at
// every level and chain to the bucket owner.
var decoders = []func(string) ([]byte, error){
	base58.Decode,
	base64.StdEncoding.DecodeString,
}

// DecodeToken decodes an encoded pb.AuthToken, trying each supported encoding.
// Input longer than DefaultMaxEncodedTokenLength is rejected undecoded.
func DecodeToken(encoded string) (*pb.AuthToken, error) {
	token, _, err := DecodeTokenBytes(encoded)
	return token, err
}

// DecodeTokenBytes is DecodeToken, also returning the protobuf bytes the
// successful encoding yielded.
//
// Callers that forward a credential to another verifier need those bytes
// verbatim — the signature was computed over them, and re-marshalling the decoded
// message is not guaranteed to reproduce them.
func DecodeTokenBytes(encoded string) (*pb.AuthToken, []byte, error) {
	return decodeToken(encoded, DefaultMaxEncodedTokenLength)
}

func decodeToken(encoded string, maxLength int) (*pb.AuthToken, []byte, error) {
	if encoded == "" {
		return nil, nil, ErrTokenRequired
	}
	if maxLength > 0 && len(encoded) > maxLength {
		return nil, nil, fmt.Errorf("%w: %d characters, limit %d",
			ErrTokenTooLong, len(encoded), maxLength)
	}
	for _, decode := range decoders {
		raw, err := decode(encoded)
		if err != nil || len(raw) == 0 {
			continue
		}
		token := &pb.AuthToken{}
		if err := proto.Unmarshal(raw, token); err != nil {
			continue
		}
		if token.Signature == nil || token.Payload == nil {
			continue
		}
		return token, raw, nil
	}
	return nil, nil, ErrInvalidToken
}

// EncodeToken serialises a token in the DDC-native encoding (base58).
func EncodeToken(token *pb.AuthToken) (string, error) {
	marshaled, err := proto.Marshal(token)
	if err != nil {
		return "", err
	}
	return base58.Encode(marshaled), nil
}

// BucketID returns the payer bucket a decoded chain names: the first non-zero
// bucket_id found walking leaf -> root.
//
// Why this is safe to feed straight back into Authorize as the bucket to bill:
// the bucket read here is attacker-supplied, and it is used ONLY as the claim
// "this is the bucket I want billed". Authorize then re-derives it independently
// and enforces two things this function does not — every level of the chain must
// name the same bucket (ErrInvalidPermissions otherwise), and the root issuer
// must be that bucket's on-chain owner, or one of the owner's honored delegates,
// with the owner holding balance (ErrInvalidOwnership / ErrInsufficientBalance
// otherwise). A forger who names a victim's bucket here is rejected by the
// ownership assertion; one who names their own is billed correctly. So no trust
// is placed in the value returned.
//
// It exists because Authorize deliberately takes the bucket as a parameter — on
// most surfaces the bucket comes from the request (a path segment, a query
// param, a job record) and the token merely has to agree with it. On the compute
// invocation surfaces there is no such request-side bucket: the caller declares
// the payer bucket inside the credential itself. This is the one-line bridge
// between the two, in one place rather than copied into every component.
//
// Returns 0 for a nil token or a chain that names no bucket at all; Authorize
// rejects 0.
func BucketID(token *pb.AuthToken) uint64 {
	// Bounded by the same depth limit the walk enforces, so a cyclic or
	// absurdly deep attacker-supplied chain cannot spin here. (proto.Unmarshal
	// cannot build a cycle, but this function is also reachable with a
	// hand-built token.)
	level := token
	for depth := 0; level != nil && depth < DefaultMaxChainDepth; depth++ {
		if bucketID := level.GetPayload().GetBucketId(); bucketID != 0 {
			return bucketID
		}
		level = level.GetPayload().GetPrev()
	}
	return 0
}

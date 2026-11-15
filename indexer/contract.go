// Package indexer speaks the blockchain index contract: a small, versioned,
// read-only HTTP API over materialised on-chain customer state. It answers three
// questions:
//
//   - about one bucket — who owns it, whether that owner has balance, whether
//     the bucket is still live, and which of the owner's proxy delegates may
//     stand in for them on writes;
//   - about one account — which buckets it owns;
//   - about one owner — which delegates it has granted, asked by owner rather
//     than by bucket because the question can precede the bucket (a claim
//     waits for a grant to land while the bucket it will bill is still being
//     created).
//
// The three are shaped by opposite pressures and it is worth naming why up front.
// The bucket lookup feeds authorization, so it must be cheap, cacheable and
// fail closed. The owner listing feeds enumeration — a login, a dashboard — so
// its worst failure is not a stale grant but an invisible omission: a bucket
// created a second ago that the caller cannot see and cannot prove is missing.
// The delegates lookup feeds both, and its worst failure is the inverse of the
// listing's: an outage read as "no delegates" would look exactly like a
// revocation. The rules therefore differ per operation, deliberately, and each
// one says so where it is implemented.
//
// The contract is deliberately host-agnostic. A consumer configures exactly one
// thing, the index's base URL, and knows nothing about which process answers
// it. Any process that materialises the on-chain customer state can implement
// the contract; moving from one implementation to another is a URL change and
// nothing else.
//
// Today DDC storage nodes serve it, because they already materialise the
// customer index. A dedicated indexer service may serve it later; for a client
// that is a URL change.
//
// The data served is public on-chain state. The endpoint provides availability,
// not confidentiality — but availability is load-bearing, because callers use
// the answer to authorize billable work. Clients therefore fail closed: when
// the index cannot be reached and no sufficiently recent answer is cached, the
// lookup returns an error rather than a permissive default.
package indexer

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ContractVersion is the major version of the wire contract implemented here.
// It is encoded in the request path so a v2 endpoint can be served alongside v1
// during a rollout, and echoed in every response body so a client pointed at a
// mismatched implementation fails loudly instead of misreading fields.
const ContractVersion = 1

const (
	// PathPrefix is the versioned root of the contract.
	PathPrefix = "/v1"

	// BucketsPath is the collection root for bucket lookups. Implementations
	// register a single-segment path parameter beneath it; clients build
	// concrete paths with BucketPath.
	BucketsPath = PathPrefix + "/buckets"

	// DelegatesPath is the owner-scoped delegates lookup. It is a required
	// ?owner= filter on its own collection, not a subresource of BucketsPath,
	// because the question "who may act for this owner" is asked before any
	// bucket exists to hang it off — claim-time grant-wait polls this while
	// the bucket it will bill is still being created.
	DelegatesPath = PathPrefix + "/delegates"
)

// Query parameters of the bucket listing.
//
// They are constants because both sides must spell them identically. Note that
// a test asserting the literal string "limit" is testing the contract, while a
// test that builds its URL from these constants is testing nothing at all.
const (
	// ParamLimit caps the page. Absent means the implementation's maximum.
	ParamLimit = "limit"
	// ParamAfter is an exclusive lower bound on bucket id: the cursor.
	ParamAfter = "after"
	// ParamIncludeRemoved asks for removed buckets to be listed too.
	ParamIncludeRemoved = "includeRemoved"
	// ParamOwner scopes the listing to one account. REQUIRED — see
	// BucketsRequestPath.
	ParamOwner = "owner"
)

// BucketPath returns the contract path for a single bucket lookup.
func BucketPath(bucketID uint64) string {
	return BucketsPath + "/" + strconv.FormatUint(bucketID, 10)
}

// BucketsRequestPath renders a complete request target for the bucket
// listing, query string included.
//
// The listing is BucketsPath with a REQUIRED ?owner= filter, not a subresource of
// an /owners collection. There is no owner entity in this contract: nothing else
// hangs off an account and no implementation serves /v1/owners/{owner}, so a
// collection root existing solely to carry one child is a resource the contract
// does not have. And it does not collide with the item lookup — `/v1/buckets/{id}`
// and `/v1/buckets` are different paths, so filtering the collection cannot change
// what the item path returns.
//
// Owner is REQUIRED, and that is the load-bearing part of the shape: an
// unfiltered GET /v1/buckets would mean "every bucket on the network", which is an
// accidental full dump of a 500k-row index. An implementation answers 400
// CodeInvalidOwner when it is absent rather than defaulting to everything.
//
// Rendering is deterministic — url.Values.Encode sorts by key, and parameters left
// at their defaults are omitted — because the client uses this string as its
// request-coalescing key. Two callers asking the same question must produce the
// same key; two asking different questions must not.
func BucketsRequestPath(owner []byte, query BucketsQuery) string {
	values := query.values()
	values.Set(ParamOwner, EncodeOwner(owner))
	return BucketsPath + "?" + values.Encode()
}

// DelegatesRequestPath renders the request target for the owner-scoped
// delegates lookup. Owner is required, for the same reason the listing's is.
//
// Rendering is deterministic for the same reason BucketsRequestPath's is: the
// client uses this string as its request-coalescing key.
func DelegatesRequestPath(owner []byte) string {
	values := url.Values{}
	values.Set(ParamOwner, EncodeOwner(owner))
	return DelegatesPath + "?" + values.Encode()
}

// BucketsQuery is the request side of the owner listing.
type BucketsQuery struct {
	// After is an exclusive lower bound on bucket id — the cursor. Zero starts
	// from the beginning, which is safe because zero is never a bucket.
	After uint64
	// Limit caps the page. Zero asks for the implementation's maximum rather
	// than for nothing, so a caller that wants "as much as you'll give me" says
	// it by saying nothing.
	Limit uint32
	// IncludeRemoved asks for removed buckets as well as live ones. Off by
	// default; see the implementation's handler for why the default is off and
	// why the switch exists at all.
	IncludeRemoved bool
}

func (q BucketsQuery) values() url.Values {
	values := url.Values{}
	if q.After != 0 {
		values.Set(ParamAfter, strconv.FormatUint(q.After, 10))
	}
	if q.Limit != 0 {
		values.Set(ParamLimit, strconv.FormatUint(uint64(q.Limit), 10))
	}
	if q.IncludeRemoved {
		values.Set(ParamIncludeRemoved, "true")
	}

	return values
}

// Error codes carried by ErrorResponse.Code. Codes, not HTTP statuses, are the
// contract: a 404 alone cannot distinguish "no such bucket" from "the owning
// account has no ledger entry", and the two mean different things to a caller.
const (
	// CodeBucketNotFound means the index holds no such bucket.
	CodeBucketNotFound = "bucket_not_found"
	// CodeAccountNotFound means the bucket exists but its owner has no
	// account record, so balance cannot be established.
	CodeAccountNotFound = "account_not_found"
	// CodeInvalidBucketID means the bucket id in the path was unparseable or
	// zero. Zero is never a real bucket.
	CodeInvalidBucketID = "invalid_bucket_id"
	// CodeInvalidOwner means the ?owner= filter was absent, or was not a 32-byte
	// hex account id. Absent is an error rather than "list everything" — see
	// BucketsRequestPath. It says nothing about any account: one this index
	// has never seen owns nothing, and that is an empty list and a 200.
	CodeInvalidOwner = "invalid_owner"
	// CodeInvalidPagination means the limit or the cursor was unusable — zero,
	// unparseable, or above the implementation's maximum page.
	CodeInvalidPagination = "invalid_pagination"
	// CodeUnavailable means the implementation could not read its index.
	CodeUnavailable = "unavailable"
)

// BucketResponse is the JSON body of GET {BucketsPath}/{bucketId}.
type BucketResponse struct {
	ContractVersion int    `json:"contractVersion"`
	BucketID        uint64 `json:"bucketId"`
	// Owner is the bucket owner's 32-byte account public key, hex encoded
	// with a "0x" prefix.
	Owner string `json:"owner"`
	// HasBalance reports a strictly positive owner balance.
	HasBalance bool `json:"hasBalance"`
	// IsRemoved reports that the bucket has been removed on chain. A removed
	// bucket authorizes nothing.
	IsRemoved bool `json:"isRemoved"`
	// Delegates lists the bucket owner's proxy delegates whose grant types are
	// honored for DDC writes, hex encoded with a "0x" prefix, in the order the
	// index holds them. ADDITIVE AT CONTRACT VERSION 1, and optional on both
	// sides: a server that predates the field omits it, and an absent or empty
	// list means the same thing — no account may stand in for this owner, so a
	// consumer denies. That is the fail-closed rollout behaviour, not an
	// oversight: requiring the key would take every pre-upgrade server offline
	// for consumers that had never asked about delegation.
	//
	// The owner listing does not populate it. Enumeration renders; it does not
	// authorize — and every element of a page shares one owner, so populating
	// it would repeat one delegate list per bucket on a wire the contract
	// already flags as the first thing to trim.
	Delegates []string `json:"delegates,omitempty"`
}

// BucketsResponse is the JSON body of GET {BucketsPath}?owner=…
//
// Each element is a full BucketResponse — the very shape the single-bucket
// operation returns — and that is a decision, not laziness:
//
//   - One decoder, one validator. An element lifted out of a page IS a valid
//     single-bucket response, so BucketResponse.Decode validates it: contract
//     version, owner encoding, bucket identity. A trimmed element type would
//     need a second decode path, and a second path is where the two drift.
//   - No follow-up per bucket. The page carries every fact the single-bucket
//     operation would have returned about each bucket it names, so a caller
//     rendering "my buckets, and which ones are usable" is done after one
//     request.
//
// The cost is duplication on the wire. Per element, contractVersion (~21 bytes)
// and owner (~77 bytes) are redundant against the envelope, and hasBalance is an
// account property that is necessarily identical across the whole page — roughly
// 60% of a ~168-byte element. At a page of 1000 that is ~100 KB of redundancy,
// which buys one decoder. At a page of 100k it would not, and this is the field
// the contract would trim first if pages ever grew that far.
type BucketsResponse struct {
	ContractVersion int `json:"contractVersion"`
	// Owner echoes the requested account, hex with a "0x" prefix. A client
	// checks it rather than trusting the body, so an implementation answering
	// about a different account is rejected instead of having its buckets
	// attributed to this one.
	Owner string `json:"owner"`
	// Buckets is the page, ascending by bucket id. It is an empty array and
	// never null for an account that owns nothing: null and [] decode the same
	// in Go but not in JavaScript, and the JS SDK is the reason this operation
	// exists.
	Buckets []BucketResponse `json:"buckets"`
	// NextAfter is the value to pass back as ParamAfter for the following page,
	// and 0 when this page is the last. Zero is unambiguous because zero is
	// never a bucket, which is why there is no separate "hasMore" flag that
	// could contradict it.
	NextAfter uint64 `json:"nextAfter"`
}

// ErrorResponse is the JSON body of any non-2xx contract response.
type ErrorResponse struct {
	ContractVersion int    `json:"contractVersion"`
	Code            string `json:"code"`
	Message         string `json:"message,omitempty"`
}

// DelegatesResponse is the JSON body of GET {DelegatesPath}?owner=…
//
// It exists because the delegates question can precede the bucket: a claim
// waits for the owner's grant to land while the bucket it will bill is still
// being created, and a bucket-scoped response cannot serve that moment. It
// reuses the contract's existing error codes — there is nothing in "who may act
// for this owner" that needs a code the contract does not already have.
type DelegatesResponse struct {
	ContractVersion int `json:"contractVersion"`
	// Owner echoes the requested account, hex with a "0x" prefix. A client
	// checks it rather than trusting the body, so an implementation answering
	// about a different account is rejected instead of having its delegates
	// attributed to this one.
	Owner string `json:"owner"`
	// Delegates is the owner's honored delegates, hex with a "0x" prefix. It
	// is an empty array and never null for an owner with none — the same Go
	// and JavaScript disagreement, and the same resolution, as
	// BucketsResponse.Buckets.
	Delegates []string `json:"delegates"`
}

// Decode validates the body and converts it to raw 32-byte keys, in order.
// owner is the account that was asked about.
//
// An owner with no honored delegates — including one this index has never
// heard of, and one whose proxies are all of unhonored types — is an empty
// answer and NOT an error. "Not granted yet" is precisely what a claim-time
// caller polls this operation for, and erroring on it would make a wait that
// is still running look like an index that cannot answer.
func (r DelegatesResponse) Decode(owner ed25519.PublicKey) ([][]byte, error) {
	if r.ContractVersion != ContractVersion {
		return nil, fmt.Errorf(
			"%w: contract version %d, want %d", ErrInvalidResponse, r.ContractVersion, ContractVersion)
	}

	responseOwner, err := DecodeOwner(r.Owner)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(responseOwner, owner) {
		return nil, fmt.Errorf("%w: response is for owner %s, requested %s",
			ErrInvalidResponse, EncodeOwner(responseOwner), EncodeOwner(owner))
	}

	return decodeDelegates(r.Delegates)
}

// decodeDelegates converts the wire hex list to raw keys, preserving order.
// An absent or empty list is valid and decodes empty.
func decodeDelegates(encoded []string) ([][]byte, error) {
	delegates := make([][]byte, 0, len(encoded))
	for _, delegate := range encoded {
		key, err := DecodeOwner(delegate)
		if err != nil {
			return nil, err
		}
		delegates = append(delegates, key)
	}

	return delegates, nil
}

// BucketInfo is the decoded form of BucketResponse.
type BucketInfo struct {
	BucketID   uint64
	Owner      ed25519.PublicKey
	HasBalance bool
	IsRemoved  bool
	// Delegates is the decoded form of BucketResponse.Delegates: the owner's
	// honored delegates as raw 32-byte keys. Nil or empty means none — nobody
	// may stand in for the owner on writes — which is also what a server that
	// predates the field reports.
	Delegates [][]byte
}

// OwnerBuckets is the decoded form of BucketsResponse.
type OwnerBuckets struct {
	Owner     ed25519.PublicKey
	Buckets   []BucketInfo
	NextAfter uint64
}

// HasMore reports whether a further page follows this one.
func (b OwnerBuckets) HasMore() bool {
	return b.NextAfter != 0
}

// clone deep-copies a page.
//
// Every caller that shared one coalesced fetch is handed the same OwnerBuckets,
// and its owner keys are byte slices. Without this, one caller sorting or
// truncating its own result in place would be sorting or truncating everyone
// else's — the hazard entry.result exists to close on the caching path, minus
// the cache.
func (b OwnerBuckets) clone() OwnerBuckets {
	cloned := OwnerBuckets{Owner: bytes.Clone(b.Owner), NextAfter: b.NextAfter}
	if b.Buckets == nil {
		return cloned
	}

	cloned.Buckets = make([]BucketInfo, len(b.Buckets))
	for i, info := range b.Buckets {
		info.Owner = bytes.Clone(info.Owner)
		info.Delegates = cloneDelegates(info.Delegates)
		cloned.Buckets[i] = info
	}

	return cloned
}

// cloneDelegates deep-copies a delegate list: the same aliasing hazard as the
// owner key in OwnerBuckets.clone and entry.result, one level down.
func cloneDelegates(delegates [][]byte) [][]byte {
	if delegates == nil {
		return nil
	}

	cloned := make([][]byte, len(delegates))
	for i, delegate := range delegates {
		cloned[i] = bytes.Clone(delegate)
	}

	return cloned
}

// Contract errors. Callers distinguish two classes:
//
//   - Definitive denials (ErrBucketNotFound, ErrAccountNotFound) — the index
//     answered, and the answer is "no". These are cached.
//   - Availability failures (ErrUnavailable, ErrInvalidResponse) — the index
//     did not answer usefully. These are never cached, and they are the
//     fail-closed path.
//
// A removed bucket is deliberately NOT an error here: it is a fact carried on
// BucketInfo.IsRemoved and on the authz.LookupResult the client returns, and
// the authorizer is what turns it into a rejection.
//
// The owner listing has no denial at all. "This account owns nothing" is an
// empty page and a nil error, so every non-nil error from that operation is an
// availability failure — which is what makes an empty page trustworthy.
var (
	ErrBucketNotFound  = errors.New("blockchain index: bucket not found")
	ErrAccountNotFound = errors.New("blockchain index: bucket owner account not found")
	ErrUnavailable     = errors.New("blockchain index: unavailable")
	ErrInvalidResponse = errors.New("blockchain index: invalid response")
	ErrNotConfigured   = errors.New("blockchain index: url not configured")
	// ErrInvalidOwner reports a malformed owner key passed BY THE CALLER, caught
	// before a request is sent. It is not an answer about an account: returning
	// an empty page for a mangled key would be indistinguishable from an account
	// that genuinely owns nothing.
	ErrInvalidOwner = errors.New("blockchain index: invalid owner")
)

// EncodeOwner renders a 32-byte account public key in the contract's wire form.
func EncodeOwner(owner []byte) string {
	return "0x" + hex.EncodeToString(owner)
}

// DecodeOwner parses the contract's wire form of an account public key.
func DecodeOwner(encoded string) (ed25519.PublicKey, error) {
	raw, err := hex.DecodeString(strings.TrimPrefix(encoded, "0x"))
	if err != nil {
		return nil, fmt.Errorf("%w: owner is not hex: %w", ErrInvalidResponse, err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: owner is %d bytes, want %d", ErrInvalidResponse, len(raw), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}

// Decode validates a response body and converts it to a BucketInfo. bucketID is
// the id that was requested; a response describing a different bucket is
// rejected rather than trusted.
//
// The delegates list is optional and additive at version 1: an absent key
// decodes as no delegates, which denies delegation — the fail-closed rollout
// behaviour — while a present entry that is not a 32-byte hex key fails the
// whole decode. A malformed key means the body is not from this contract, and a
// partial answer from a foreign shape is worse than none.
func (r BucketResponse) Decode(bucketID uint64) (BucketInfo, error) {
	if r.ContractVersion != ContractVersion {
		return BucketInfo{}, fmt.Errorf(
			"%w: contract version %d, want %d", ErrInvalidResponse, r.ContractVersion, ContractVersion)
	}
	if r.BucketID != bucketID {
		return BucketInfo{}, fmt.Errorf(
			"%w: response is for bucket %d, requested %d", ErrInvalidResponse, r.BucketID, bucketID)
	}

	owner, err := DecodeOwner(r.Owner)
	if err != nil {
		return BucketInfo{}, err
	}

	delegates, err := decodeDelegates(r.Delegates)
	if err != nil {
		return BucketInfo{}, err
	}

	return BucketInfo{
		BucketID:   r.BucketID,
		Owner:      owner,
		HasBalance: r.HasBalance,
		IsRemoved:  r.IsRemoved,
		Delegates:  delegates,
	}, nil
}

// Decode validates a page and converts it to an OwnerBuckets. owner is the
// account that was asked about.
//
// This is stricter than the single-bucket decoder, because a page has two
// failure modes a single answer does not and both are silent:
//
//   - A wrong filter. An implementation that filtered on the wrong field returns
//     somebody else's buckets, and the caller renders them as the user's own.
//     Every element's owner must therefore equal the requested owner — checking
//     the envelope alone would not catch it.
//   - A skipped bucket. ParamAfter is a bucket id, so a caller that pages with
//     the last id it saw loses every smaller id if the page arrives unordered.
//     Ids must be strictly ascending (which also rejects duplicates and the
//     impossible id 0), and NextAfter, when set, must be the last id on the
//     page — the only value that cannot skip.
//
// An empty page is valid and is not an error. That is the point of the whole
// operation: "this account owns nothing" has to be distinguishable from "the
// index is broken", and the second one is the only one that gets an error.
func (r BucketsResponse) Decode(owner ed25519.PublicKey) (OwnerBuckets, error) {
	if r.ContractVersion != ContractVersion {
		return OwnerBuckets{}, fmt.Errorf(
			"%w: contract version %d, want %d", ErrInvalidResponse, r.ContractVersion, ContractVersion)
	}

	responseOwner, err := DecodeOwner(r.Owner)
	if err != nil {
		return OwnerBuckets{}, err
	}
	if !bytes.Equal(responseOwner, owner) {
		return OwnerBuckets{}, fmt.Errorf("%w: response is for owner %s, requested %s",
			ErrInvalidResponse, EncodeOwner(responseOwner), EncodeOwner(owner))
	}

	buckets := make([]BucketInfo, 0, len(r.Buckets))
	var previousID uint64
	for i, element := range r.Buckets {
		if element.BucketID <= previousID {
			return OwnerBuckets{}, fmt.Errorf("%w: bucket ids must ascend strictly, element %d is %d after %d",
				ErrInvalidResponse, i, element.BucketID, previousID)
		}

		// Decoding an element against its own id makes the identity half of
		// BucketResponse.Decode vacuous — a page names its own ids, no caller
		// asked for a specific one — but the version and owner-encoding halves
		// are the ones that matter here, and reusing the single validator is why
		// the element shape is BucketResponse to begin with.
		info, err := element.Decode(element.BucketID)
		if err != nil {
			return OwnerBuckets{}, err
		}
		if !bytes.Equal(info.Owner, owner) {
			return OwnerBuckets{}, fmt.Errorf("%w: bucket %d is owned by %s, requested %s",
				ErrInvalidResponse, info.BucketID, EncodeOwner(info.Owner), EncodeOwner(owner))
		}

		previousID = element.BucketID
		buckets = append(buckets, info)
	}

	if r.NextAfter != 0 {
		if len(buckets) == 0 {
			return OwnerBuckets{}, fmt.Errorf(
				"%w: nextAfter %d on an empty page", ErrInvalidResponse, r.NextAfter)
		}
		if last := buckets[len(buckets)-1].BucketID; r.NextAfter != last {
			return OwnerBuckets{}, fmt.Errorf("%w: nextAfter is %d, want the last id on the page (%d)",
				ErrInvalidResponse, r.NextAfter, last)
		}
	}

	return OwnerBuckets{Owner: bytes.Clone(owner), Buckets: buckets, NextAfter: r.NextAfter}, nil
}

// ErrorForCode maps a contract error code onto the package's error values.
// Unknown codes are treated as availability failures, which keeps an
// implementation that grows a new code from being read as a definitive denial.
//
// CodeInvalidOwner and CodeInvalidPagination are deliberately absent from the
// switch and so land on that default. They are request-side faults, and the
// owner listing has no denial to map them onto; leaving them out means no future
// edit can make a rejected request look like "this account owns nothing".
func ErrorForCode(code, message string) error {
	var base error
	switch code {
	case CodeBucketNotFound:
		base = ErrBucketNotFound
	case CodeAccountNotFound:
		base = ErrAccountNotFound
	case CodeInvalidBucketID:
		base = ErrBucketNotFound
	default:
		base = ErrUnavailable
	}
	if message == "" {
		return base
	}
	return fmt.Errorf("%w: %s", base, message)
}

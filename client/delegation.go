package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb"
)

// Defaults of a Delegation.
const (
	// DefaultCallTTL is how long the tokens a Delegation mints for ordinary
	// calls live. Each is reused until half of it has passed.
	DefaultCallTTL = 10 * time.Minute
	// lifetimeSlack is added to what a create asks for, so the token covers
	// the time the create itself takes and clock skew.
	lifetimeSlack = 2 * time.Minute
)

// Delegation mints short-lived tokens beneath a grant. The bucket owner signs
// the grant once, to the service's own key (see NewGrant); the service holds
// the grant and that key, never the owner's:
//
//	level 1, the grant: signed by the bucket owner; subject = the service's key,
//	                    the bucket, the operations, can_delegate, an expiry
//	level 2, minted:    signed by the service's key per call or per sandbox; the
//	                    same bucket and operations, a short expiry, never later
//	                    than the grant's
//
// A leaked minted token dies with its short expiry. The sandbox API refuses
// the grant presented alone, but other DDC gateways accept a one-level grant,
// so keep the grant as secret as the key.
//
// A Delegation is safe for concurrent use.
type Delegation struct {
	grant   *pb.AuthToken
	signer  auth.Signer
	callTTL time.Duration
	subject []byte
	now     func() time.Time

	mu     sync.Mutex
	cached string
	until  time.Time
}

var _ LifetimeTokenSource = (*Delegation)(nil)

// DelegationOption configures a Delegation.
type DelegationOption func(*Delegation)

// WithCallTTL sets how long the tokens minted for ordinary calls live
// (default DefaultCallTTL).
func WithCallTTL(ttl time.Duration) DelegationOption {
	return func(d *Delegation) { d.callTTL = ttl }
}

// WithSubject names subject in every token the Delegation mints, for example
// a key that identifies the run, for attribution. Without it the minted levels
// name no subject, so nothing can be chained beneath them.
func WithSubject(subject []byte) DelegationOption {
	return func(d *Delegation) { d.subject = bytes.Clone(subject) }
}

// NewDelegation checks that grant names signer's key as its subject, names a
// bucket and has not expired, and returns a Delegation that signs with signer.
// For an ed25519 key, signer is auth.NewEd25519Signer(key).
func NewDelegation(grant *pb.AuthToken, signer auth.Signer, opts ...DelegationOption) (*Delegation, error) {
	if grant == nil || grant.GetPayload() == nil {
		return nil, errors.New("cere: delegation: no grant")
	}
	if signer == nil {
		return nil, errors.New("cere: delegation: no signer")
	}
	p := grant.GetPayload()
	if !bytes.Equal(p.GetSubject(), signer.PublicKey()) {
		return nil, errors.New("cere: delegation: the grant names another key than the signer's; " +
			"the bucket owner must sign a grant to this key")
	}
	if p.GetBucketId() == 0 {
		return nil, errors.New("cere: delegation: the grant names no bucket")
	}
	if len(p.GetOperations()) == 0 {
		return nil, errors.New("cere: delegation: the grant grants no operation")
	}
	d := &Delegation{grant: grant, signer: signer, callTTL: DefaultCallTTL, now: time.Now}
	for _, opt := range opts {
		opt(d)
	}
	if d.callTTL <= 0 {
		return nil, errors.New("cere: delegation: the call TTL must be positive")
	}
	if !d.ExpiresAt().After(d.now()) {
		return nil, d.expired()
	}
	return d, nil
}

// ParseDelegation is NewDelegation for a grant encoded as auth.EncodeToken
// encodes it.
func ParseDelegation(encodedGrant string, signer auth.Signer, opts ...DelegationOption) (*Delegation, error) {
	grant, err := auth.DecodeToken(encodedGrant)
	if err != nil {
		return nil, fmt.Errorf("cere: delegation: the grant: %w", err)
	}
	return NewDelegation(grant, signer, opts...)
}

// Bucket is the bucket the grant is for.
func (d *Delegation) Bucket() uint64 { return d.grant.GetPayload().GetBucketId() }

// ExpiresAt is when the grant, and everything minted beneath it, stops
// working.
func (d *Delegation) ExpiresAt() time.Time { return time.Unix(auth.ExpiresAtSeconds(d.grant), 0) }

func (d *Delegation) expired() error {
	return fmt.Errorf("cere: delegation: the grant expired at %s: the bucket owner must sign a new one",
		d.ExpiresAt().UTC().Format(time.RFC3339))
}

// Mint signs one level beneath the grant, valid for ttl but never past the
// grant, naming subject (which may be nil). It returns the encoded token and
// its expiry.
func (d *Delegation) Mint(ttl time.Duration, subject []byte) (string, time.Time, error) {
	now := d.now()
	parent := auth.ExpiresAtSeconds(d.grant)
	if parent <= now.Unix() {
		return "", time.Time{}, d.expired()
	}
	exp := now.Add(ttl).Unix()
	if exp > parent {
		exp = parent // a level may not outlive its parent
	}
	bucket := d.Bucket()
	no := false
	leaf := &pb.AuthToken{Payload: &pb.Payload{
		Prev:        d.grant,
		Subject:     subject,
		CanDelegate: &no,
		// The same operations at every level: the attenuation check reads them so.
		Operations: append([]pb.Operation(nil), d.grant.GetPayload().GetOperations()...),
		BucketId:   &bucket,
		ExpiresAt:  &exp,
	}}
	if _, err := auth.SignMessage(d.signer, leaf); err != nil {
		return "", time.Time{}, fmt.Errorf("cere: delegation: sign: %w", err)
	}
	enc, err := auth.EncodeToken(leaf)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("cere: delegation: encode: %w", err)
	}
	return enc, time.Unix(exp, 0), nil
}

// Token returns a token for an ordinary call: minted with the call TTL and
// reused until half of it has passed.
func (d *Delegation) Token(context.Context) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cached != "" && d.until.Sub(d.now()) > d.callTTL/2 {
		return d.cached, nil
	}
	tok, exp, err := d.Mint(d.callTTL, d.subject)
	if err != nil {
		return "", err
	}
	d.cached, d.until = tok, exp
	return tok, nil
}

// TokenFor mints a token valid for life and two minutes more, but never past
// the grant.
func (d *Delegation) TokenFor(_ context.Context, life time.Duration) (string, error) {
	tok, _, err := d.Mint(life+lifetimeSlack, d.subject)
	return tok, err
}

// NewGrant is what the bucket owner signs once: a one-level token for bucket
// that grants COMPUTE to subject (the service's public key) until expires, and
// lets subject sign levels beneath it. In production the owner's wallet signs
// it; this is for owners who hold their key in a program, and for tests.
func NewGrant(owner auth.Signer, subject []byte, bucket uint64, expires time.Time) (string, error) {
	if len(subject) == 0 {
		return "", errors.New("cere: grant: no subject key")
	}
	if bucket == 0 {
		return "", errors.New("cere: grant: no bucket")
	}
	exp := expires.Unix()
	yes := true
	grant := &pb.AuthToken{Payload: &pb.Payload{
		Subject:     bytes.Clone(subject),
		CanDelegate: &yes,
		Operations:  []pb.Operation{pb.Operation_COMPUTE},
		BucketId:    &bucket,
		ExpiresAt:   &exp,
	}}
	if _, err := auth.SignMessage(owner, grant); err != nil {
		return "", fmt.Errorf("cere: grant: sign: %w", err)
	}
	return auth.EncodeToken(grant)
}

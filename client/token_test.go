package client_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/authtest"
	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
)

const bucket = uint64(4116)

func key(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, k, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func pub(k ed25519.PrivateKey) ed25519.PublicKey { return k.Public().(ed25519.PublicKey) }

// ownerToken is a one-level COMPUTE token for bucket, expiring at exp.
func ownerToken(t *testing.T, owner ed25519.PrivateKey, exp time.Time) string {
	t.Helper()
	e, b := exp.Unix(), bucket
	tok := &pb.AuthToken{Payload: &pb.Payload{BucketId: &b, Operations: []pb.Operation{pb.Operation_COMPUTE}, ExpiresAt: &e}}
	if err := auth.SignToken(owner, tok); err != nil {
		t.Fatal(err)
	}
	enc, err := auth.EncodeToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	return enc
}

func TestStaticToken(t *testing.T) {
	owner := key(t)
	tok := client.StaticToken(ownerToken(t, owner, time.Now().Add(30*time.Minute)))
	exp, err := tok.ExpiresAt()
	if err != nil || time.Until(exp) < 29*time.Minute || time.Until(exp) > 31*time.Minute {
		t.Fatalf("ExpiresAt %s %v", exp, err)
	}
	if got, err := tok.Token(context.Background()); err != nil || got != string(tok) {
		t.Fatalf("Token: %v", err)
	}
	if _, err := client.StaticToken("not-a-ddc-token").ExpiresAt(); err == nil {
		t.Error("ExpiresAt of an opaque token: no error")
	}
	for _, s := range []string{fmt.Sprint(tok), fmt.Sprintf("%v %+v %#v %s", tok, tok, tok, tok)} {
		if strings.Contains(s, string(tok)) {
			t.Fatal("fmt printed the token")
		}
	}
}

func TestStaticTokenExpiryIsTheEarliestLevel(t *testing.T) {
	owner, svc := key(t), key(t)
	grant, err := client.NewGrant(auth.NewEd25519Signer(owner), pub(svc), bucket, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	d, err := client.ParseDelegation(grant, auth.NewEd25519Signer(svc))
	if err != nil {
		t.Fatal(err)
	}
	leaf, exp, err := d.Mint(10*time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.StaticToken(leaf).ExpiresAt()
	if err != nil || !got.Equal(exp) {
		t.Fatalf("%s, want %s (%v)", got, exp, err)
	}
}

// authorize checks a token the way the API does: the chain roots in the
// bucket's owner, COMPUTE on the bucket, not expired.
func authorize(t *testing.T, owner ed25519.PrivateKey, encoded string) *auth.Verified {
	t.Helper()
	ownerKP := authtest.Keypair{Pub: pub(owner), Priv: owner}
	v, err := auth.New(authtest.OwnedBy(ownerKP, bucket)).Authorize(context.Background(), encoded, pb.Operation_COMPUTE, bucket)
	if err != nil {
		t.Fatalf("the API would refuse the token: %v", err)
	}
	return v
}

func TestDelegation(t *testing.T) {
	owner, svc := key(t), key(t)
	grantEnd := time.Now().Add(2 * time.Hour)
	grant, err := client.NewGrant(auth.NewEd25519Signer(owner), pub(svc), bucket, grantEnd)
	if err != nil {
		t.Fatal(err)
	}
	authorize(t, owner, grant) // the grant alone verifies; the sandbox API refuses it for naming a subject
	run := pub(key(t))
	d, err := client.ParseDelegation(grant, auth.NewEd25519Signer(svc), client.WithCallTTL(4*time.Minute), client.WithSubject(run))
	if err != nil {
		t.Fatal(err)
	}
	if d.Bucket() != bucket || d.ExpiresAt().Unix() != grantEnd.Unix() {
		t.Fatalf("bucket %d, expires %s", d.Bucket(), d.ExpiresAt())
	}

	// Call tokens: minted with the call TTL, reused for half of it.
	first, err := d.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := d.Token(context.Background()); again != first {
		t.Error("a call token was not reused")
	}
	v := authorize(t, owner, first)
	if string(v.RootIssuer) != string(pub(owner)) || string(v.Subject) != string(run) {
		t.Errorf("root issuer %x, subject %x", v.RootIssuer, v.Subject)
	}
	exp, _ := client.StaticToken(first).ExpiresAt()
	if d := time.Until(exp); d < 3*time.Minute || d > 4*time.Minute {
		t.Errorf("a call token lives %s, want 4 min", d)
	}
	decoded, _ := auth.DecodeToken(first)
	if p := decoded.GetPayload(); p.GetCanDelegate() || p.GetBucketId() != bucket ||
		fmt.Sprint(p.GetOperations()) != fmt.Sprint([]pb.Operation{pb.Operation_COMPUTE}) {
		t.Errorf("minted level: %+v", p)
	}

	// A token for a sandbox: its life and two minutes, never past the grant.
	long, err := d.TokenFor(context.Background(), 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	authorize(t, owner, long)
	exp, _ = client.StaticToken(long).ExpiresAt()
	if d := time.Until(exp); d < 21*time.Minute || d > 22*time.Minute {
		t.Errorf("a 20 min token lives %s, want 22 min", d)
	}
	// Asked for more than the grant has left, it mints up to the grant.
	upto, err := d.TokenFor(context.Background(), 3*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if exp, _ = client.StaticToken(upto).ExpiresAt(); exp.Unix() != grantEnd.Unix() {
		t.Errorf("a token past the grant expires %s, want the grant's %s", exp, grantEnd)
	}
	clamped, cexp, err := d.Mint(5*time.Hour, nil)
	if err != nil || cexp.Unix() != grantEnd.Unix() {
		t.Fatalf("Mint past the grant: %s %v", cexp, err)
	}
	authorize(t, owner, clamped)
}

func TestNewDelegationChecks(t *testing.T) {
	owner, svc, other := key(t), key(t), key(t)
	signer := auth.NewEd25519Signer(svc)
	grant, _ := client.NewGrant(auth.NewEd25519Signer(owner), pub(svc), bucket, time.Now().Add(time.Hour))
	if _, err := client.ParseDelegation(grant, auth.NewEd25519Signer(other)); err == nil {
		t.Error("a grant to another key: no error")
	}
	if _, err := client.ParseDelegation("not a token", signer); err == nil {
		t.Error("a malformed grant: no error")
	}
	expired, _ := client.NewGrant(auth.NewEd25519Signer(owner), pub(svc), bucket, time.Now().Add(-time.Minute))
	if _, err := client.ParseDelegation(expired, signer); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Errorf("an expired grant: %v", err)
	}
	if _, err := client.NewGrant(auth.NewEd25519Signer(owner), pub(svc), 0, time.Now().Add(time.Hour)); err == nil {
		t.Error("a grant with no bucket: no error")
	}
	if _, err := client.NewGrant(auth.NewEd25519Signer(owner), nil, bucket, time.Now().Add(time.Hour)); err == nil {
		t.Error("a grant with no subject: no error")
	}
}

func TestStaticTokenNeverPrints(t *testing.T) {
	const secret = "secret-token-value"
	tok := client.StaticToken(secret)
	cfg := client.Config{BaseURL: "https://api.example", Token: tok, TokenFrom: client.EnvToken}
	var logged bytes.Buffer
	slog.New(slog.NewJSONHandler(&logged, nil)).Info("config", "token", tok, "cfg", cfg)
	slog.New(slog.NewTextHandler(&logged, nil)).Info("config", "token", tok)
	asJSON, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{
		fmt.Sprintf("%v %+v %#v %s %q %x %X %d %10s %-10v", tok, tok, tok, tok, tok, tok, tok, tok, tok, tok),
		fmt.Sprintf("%v %+v %#v %s", cfg, cfg, cfg, cfg),
		fmt.Sprint(&cfg), string(asJSON), logged.String(),
	} {
		if strings.Contains(s, secret) || strings.Contains(s, fmt.Sprintf("%x", secret)) {
			t.Fatalf("the token printed: %s", s)
		}
	}
	if !strings.Contains(string(asJSON), client.StaticTokenRedacted) {
		t.Errorf("JSON %s: want the placeholder", asJSON)
	}
	if got, _ := tok.Token(context.Background()); got != secret || string(cfg.Token) != secret {
		t.Error("the token itself is lost")
	}
}

package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
)

// TokenSource gives the access token for the next call.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// LifetimeTokenSource is a TokenSource that can give a token that stays valid
// for at least life. A sandbox create asks for one: a sandbox ends 60 s before
// the token it was created with expires.
type LifetimeTokenSource interface {
	TokenSource
	TokenFor(ctx context.Context, life time.Duration) (string, error)
}

// ErrTokenExpiresTooSoon is returned (wrapped) when a token would expire
// before what it is asked for would end: a sandbox created with it would stop
// early with exit reason token_expiry.
var ErrTokenExpiresTooSoon = errors.New("the access token expires too soon")

// TokenFunc adapts a function to TokenSource.
type TokenFunc func(ctx context.Context) (string, error)

// Token calls f.
func (f TokenFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

// StaticToken is one encoded access token used for every call: a COMPUTE token
// for the bucket from the developer console (1, 7 or 30 days), or what `cere
// login` saved. It cannot be revoked; it only expires. Printing it with fmt
// shows a placeholder, not the token.
type StaticToken string

var _ LifetimeTokenSource = StaticToken("")

// Token returns the token.
func (t StaticToken) Token(context.Context) (string, error) { return string(t), nil }

// TokenFor returns the token if it stays valid for life, and an error wrapping
// ErrTokenExpiresTooSoon if it does not. A token whose expiry cannot be read
// is returned as it is; the API will judge it.
func (t StaticToken) TokenFor(_ context.Context, life time.Duration) (string, error) {
	exp, err := t.ExpiresAt()
	if err != nil || exp.IsZero() {
		return string(t), nil
	}
	if time.Until(exp) < life {
		return "", fmt.Errorf("cere: %w: it expires at %s and this needs it until %s: use a newer token",
			ErrTokenExpiresTooSoon, exp.UTC().Format(time.RFC3339), time.Now().Add(life).UTC().Format(time.RFC3339))
	}
	return string(t), nil
}

// ExpiresAt decodes the token and returns its expiry: the earliest of its
// levels'. Zero when the token names none.
func (t StaticToken) ExpiresAt() (time.Time, error) {
	tok, err := auth.DecodeToken(string(t))
	if err != nil {
		return time.Time{}, err
	}
	exp := int64(0)
	for level := tok; level != nil; level = level.GetPayload().GetPrev() {
		if s := auth.ExpiresAtSeconds(level); s > 0 && (exp == 0 || s < exp) {
			exp = s
		}
	}
	if exp == 0 {
		return time.Time{}, nil
	}
	return time.Unix(exp, 0), nil
}

// String hides the token.
func (t StaticToken) String() string { return "StaticToken(…)" }

// GoString hides the token.
func (t StaticToken) GoString() string { return t.String() }

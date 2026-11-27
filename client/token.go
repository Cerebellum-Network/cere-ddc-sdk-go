package client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
)

// TokenSource gives the access token for the next call.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// LifetimeTokenSource is a TokenSource that can also give a token that stays
// valid for life, as far as it can: a token to hand to a sandbox for its run
// (for example the model key of an agent run). A Delegation mints one; a
// StaticToken is itself.
type LifetimeTokenSource interface {
	TokenSource
	TokenFor(ctx context.Context, life time.Duration) (string, error)
}

// TokenFunc adapts a function to TokenSource.
type TokenFunc func(ctx context.Context) (string, error)

// Token calls f.
func (f TokenFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

// StaticToken is one encoded access token used for every call: a COMPUTE token
// for the bucket from the developer console (1, 7 or 30 days), or what `cere
// login` saved. It cannot be revoked; it only expires.
//
// It never prints: fmt (every verb), slog and encoding/json show a
// placeholder, StaticTokenRedacted, not the token. Token returns it.
type StaticToken string

// StaticTokenRedacted is what a StaticToken prints as.
const StaticTokenRedacted = "StaticToken(…)"

var _ LifetimeTokenSource = StaticToken("")

// Token returns the token.
func (t StaticToken) Token(context.Context) (string, error) { return string(t), nil }

// TokenFor returns the token as it is: a static token cannot mint another.
func (t StaticToken) TokenFor(context.Context, time.Duration) (string, error) { return string(t), nil }

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
func (t StaticToken) String() string { return StaticTokenRedacted }

// GoString hides the token.
func (t StaticToken) GoString() string { return StaticTokenRedacted }

// Format hides the token, whatever the verb (%s, %v, %q, %x, %d, ...).
func (t StaticToken) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, StaticTokenRedacted) }

// LogValue hides the token from log/slog.
func (t StaticToken) LogValue() slog.Value { return slog.StringValue(StaticTokenRedacted) }

// MarshalJSON hides the token: it encodes as the placeholder. A Config or a
// struct that holds a StaticToken can be logged as JSON; it cannot be saved
// that way (keep the token itself, from Token, where it belongs).
func (t StaticToken) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(StaticTokenRedacted)), nil
}

var (
	_ fmt.Formatter  = StaticToken("")
	_ slog.LogValuer = StaticToken("")
)

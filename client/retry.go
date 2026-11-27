package client

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"slices"
	"time"
)

// RetryPolicy says how often, and how far apart, a call is tried.
//
// A call is tried again only when doing so cannot make it take effect twice:
//
//   - GET, HEAD, PUT and DELETE (reads and whole replacements), and any call
//     with an Idempotency-Key (a sandbox create, a detached exec): on 429,
//     502, 503 and 504, and on transport errors;
//   - any call, POST included: on 429 and 503 AUTH_UNAVAILABLE (the API refused
//     before doing anything), on codes the service lists for that call (a
//     sandbox create on 503 no_capacity), and when the request reached no
//     server (a failed name lookup, dial or proxy connection).
//
// Waits grow exponentially from MinBackoff to MaxBackoff with full jitter; a
// Retry-After from the server is honoured up to MaxBackoff. A cancelled
// context stops the retries at once.
type RetryPolicy struct {
	// MaxAttempts counts the first try; 1 turns retries off.
	MaxAttempts int
	// MinBackoff and MaxBackoff bound the wait between two tries.
	MinBackoff, MaxBackoff time.Duration
}

// DefaultRetryPolicy is what a Client uses unless WithRetry says otherwise.
var DefaultRetryPolicy = RetryPolicy{MaxAttempts: 3, MinBackoff: 250 * time.Millisecond, MaxBackoff: 5 * time.Second}

// NoRetry tries every call once.
var NoRetry = RetryPolicy{MaxAttempts: 1}

// idempotent reports whether sending r twice has the effect of sending it
// once: a read, a whole replacement, or a call with an Idempotency-Key.
func (r *Request) idempotent() bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	}
	return r.IdempotencyKey != ""
}

// shouldRetry reports whether err, from one try of r, may be retried.
// r.RetryOn lists codes the call's service says were refused before anything
// was done.
func shouldRetry(r *Request, err error) bool {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.Status == http.StatusTooManyRequests:
			return true
		case apiErr.Status == http.StatusServiceUnavailable && apiErr.Code == CodeAuthUnavailable:
			return true
		case apiErr.Code != "" && slices.Contains(r.RetryOn, apiErr.Code):
			return true
		case r.idempotent():
			switch apiErr.Status {
			case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
				return true
			}
		}
		return false
	}
	var te *transportError
	if errors.As(err, &te) {
		return te.unsent || r.idempotent()
	}
	return false
}

// unsent reports whether a transport error happened before req reached any
// server: a failed name lookup, dial or proxy connection for req itself. A
// failure on a redirect's next hop is not: the first server had the request.
func unsent(req *http.Request, err error) bool {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.URL != req.URL.String() {
		return false
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr) && (opErr.Op == "dial" || opErr.Op == "proxyconnect")
}

// backoff is the wait before try number attempt+1 (attempt counts from 1).
func (p RetryPolicy) backoff(attempt int, err error) time.Duration {
	maxWait := p.MaxBackoff
	if maxWait <= 0 {
		maxWait = DefaultRetryPolicy.MaxBackoff
	}
	var apiErr *Error
	if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
		return min(apiErr.RetryAfter, maxWait)
	}
	base := p.MinBackoff
	if base <= 0 {
		base = DefaultRetryPolicy.MinBackoff
	}
	ceiling := base << min(attempt-1, 16)
	if ceiling > maxWait || ceiling <= 0 {
		ceiling = maxWait
	}
	return time.Duration(rand.Int64N(int64(ceiling)) + 1)
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

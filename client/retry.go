package client

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"net/http"
	"slices"
	"time"
)

// RetryPolicy says how often, and how far apart, a call is tried.
//
// A call is tried again only when doing so cannot make it take effect twice:
//
//   - GET, HEAD, PUT and DELETE (reads and whole replacements): on 429, 502, 503
//     and 504, and on transport errors;
//   - any call, POST included: on 429 and 503 AUTH_UNAVAILABLE (the API refused
//     before doing anything), on codes the service lists for that call (a
//     sandbox create on 503 no_capacity), and when no connection was made.
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

func idempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	}
	return false
}

// shouldRetry reports whether err, from one try of a call with this method,
// may be retried. retryOn lists codes the call's service says were refused
// before anything was done.
func shouldRetry(method string, retryOn []string, err error) bool {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.Status == http.StatusTooManyRequests:
			return true
		case apiErr.Status == http.StatusServiceUnavailable && apiErr.Code == CodeAuthUnavailable:
			return true
		case apiErr.Code != "" && slices.Contains(retryOn, apiErr.Code):
			return true
		case idempotent(method):
			switch apiErr.Status {
			case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
				return true
			}
		}
		return false
	}
	var te *transportError
	if errors.As(err, &te) {
		return te.unsent || idempotent(method)
	}
	return false
}

// unsent reports whether a transport error happened before the request
// reached the server: a failed dial or name lookup.
func unsent(err error) bool {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr) && opErr.Op == "dial"
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

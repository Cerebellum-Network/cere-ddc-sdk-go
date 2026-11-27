package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

// fnError carries the error of Follow's caller through Stream. It has no
// Unwrap, so that Stream does not take an io.EOF in it for the end of the
// stream.
type fnError struct{ err error }

func (e *fnError) Error() string { return e.err.Error() }

// Follow reads a stream that can be resumed, such as an event feed or a
// detached command's output, for as long as it lasts. open gives the request
// of each connection, the first one included, from what fn has seen so far
// (a Last-Event-ID, offsets). When a connection ends, drops, sends nothing
// for idle (0: no limit, the wait for the answer's headers included), or
// cannot be opened for a reason that may pass (429, 502, 503, 504, an error
// event of the stream, a transport error), Follow opens the stream again.
//
// It waits between two connections: at least the retry policy's MinBackoff,
// growing as the policy's backoff does (up to MaxBackoff) while connections
// fail or end without an event, and back to MinBackoff after one that
// delivered an event. It never gives up on a failure that may pass: an
// outage of the API (a deploy) only delays it, and the caller's place is
// kept. It returns when fn returns an error (io.EOF ends it without one),
// the API refuses the stream for another reason, or ctx ends; bound it with
// ctx.
func (c *Client) Follow(ctx context.Context, open func() *Request, idle time.Duration, fn func(Event) error) error {
	floor := c.retry.MinBackoff
	if floor <= 0 {
		floor = DefaultRetryPolicy.MinBackoff
	}
	// fruitless counts the connections since the last one that delivered an
	// event, this one included.
	fruitless := 0
	for {
		r := open()
		got := false
		err := c.Stream(ctx, r, idle, func(ev Event) error {
			got = true
			if err := fn(ev); err != nil {
				return &fnError{err: err}
			}
			return nil
		})
		var stop *fnError
		if errors.As(err, &stop) {
			if errors.Is(stop.err, io.EOF) {
				return nil
			}
			return stop.err
		}
		if ctx.Err() != nil {
			if err == nil {
				err = wrapf("%s %s: %w", r.Method, r.Path, ctx.Err())
			}
			return err
		}
		if err != nil && !resumable(err) {
			return err
		}
		if got {
			fruitless = 0
		}
		fruitless++
		// A floor under every wait: a server that sends an event and then
		// ends, or answers an error event, is not reopened in a tight loop.
		wait := max(c.retry.backoff(fruitless, err), floor)
		if err := sleep(ctx, wait); err != nil {
			return wrapf("%s %s: %w", r.Method, r.Path, err)
		}
	}
}

// resumable reports whether a stream that failed with err may be opened
// again: it went silent, its connection failed, or the API answered a status
// that may pass.
func resumable(err error) bool {
	if errors.Is(err, ErrStreamIdle) {
		return true
	}
	var te *transportError
	if errors.As(err, &te) {
		return true
	}
	var e *Error
	if errors.As(err, &e) && !e.notStream {
		switch e.Status {
		case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
	}
	return false
}

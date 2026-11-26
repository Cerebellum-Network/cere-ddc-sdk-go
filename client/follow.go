package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

// followFailures is how many connections in a row Follow lets fail without
// an event before it gives up. Each is itself tried as the retry policy says.
const followFailures = 5

// fnError carries the error of Follow's caller through Stream. It has no
// Unwrap, so that Stream does not take an io.EOF in it for the end of the
// stream.
type fnError struct{ err error }

func (e *fnError) Error() string { return e.err.Error() }

// Follow reads a stream that can be resumed, such as an event feed or a
// detached command's output, for as long as it lasts. open gives the request
// of each connection, the first one included, from what fn has seen so far
// (a Last-Event-ID, offsets). When a connection ends, drops, sends nothing
// for idle (0: no limit), or cannot be opened for a reason that may pass
// (429, 502, 503, 504, a transport error), Follow opens the stream again,
// at once after a connection that sent events and with backoff otherwise.
//
// It returns when fn returns an error (io.EOF ends it without one), the API
// refuses the stream for another reason, connections fail 5 times in a row
// without an event, or ctx ends.
func (c *Client) Follow(ctx context.Context, open func() *Request, idle time.Duration, fn func(Event) error) error {
	failures := 0
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
		var wait time.Duration
		switch {
		case got:
			failures = 0
		case err == nil:
			// The server ended a stream that had nothing to send: not a
			// failure, but no reason to reconnect in a tight loop either.
			wait = c.retry.backoff(1, nil)
		default:
			if failures++; failures >= followFailures {
				return err
			}
			wait = c.retry.backoff(failures, err)
		}
		if wait > 0 {
			if err := sleep(ctx, wait); err != nil {
				return wrapf("%s %s: %w", r.Method, r.Path, err)
			}
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
	if errors.As(err, &e) {
		switch e.Status {
		case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
	}
	return false
}

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Codes the API's services share. A service package adds its own (for
// example sandbox.CodeNotRunning).
const (
	// CodeInvalidRequest: a field, a query parameter or the JSON body is invalid (400).
	CodeInvalidRequest = "invalid_request"
	// CodeBadRequest: the request does not say which bucket it is for (400).
	CodeBadRequest = "BAD_REQUEST"
	// CodeUnauthorized: no token, a token that does not verify, or a grant presented alone (401).
	CodeUnauthorized = "UNAUTHORIZED"
	// CodeForbidden: the token is not for this bucket, or its owner has no balance (403).
	CodeForbidden = "FORBIDDEN"
	// CodeNotFound: no such resource or route (404).
	CodeNotFound = "not_found"
	// CodeMethodNotAllowed: the route does not take the method (405).
	CodeMethodNotAllowed = "method_not_allowed"
	// CodeTooLarge: a body over its limit (413).
	CodeTooLarge = "too_large"
	// CodeTooManyRequests: too many calls in flight; nothing was done (429).
	CodeTooManyRequests = "too_many_requests"
	// CodeInternal: a fault of the API (500).
	CodeInternal = "internal"
	// CodeUpstreamError: a call behind the API failed in transport (502).
	CodeUpstreamError = "upstream_error"
	// CodeAuthUnavailable: the token's bucket could not be checked; nothing was done (503).
	CodeAuthUnavailable = "AUTH_UNAVAILABLE"
)

// ErrNoToken is returned, without a call, when neither the client nor the call
// has an access token.
var ErrNoToken = errors.New("cere: no access token: set one with client.WithToken or client.WithTokenSource " +
	"(an access token from the developer console, or CERE_TOKEN with client.NewFromEnv)")

// Error is an answer outside 2xx, or an error event in a stream.
type Error struct {
	// Status is the HTTP status; 502 for an error event in a stream.
	Status int
	// Code is the API's stable code, such as "not_running" or "no_capacity";
	// empty when the answer carried none.
	Code string
	// Message is for people and may change.
	Message string
	// Method and Path name the call.
	Method, Path string
	// RetryAfter is the server's Retry-After, when it sent one.
	RetryAfter time.Duration
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("cere: ")
	if e.Method != "" {
		b.WriteString(e.Method + " " + e.Path + ": ")
	}
	b.WriteString(strconv.Itoa(e.Status))
	if e.Code != "" {
		b.WriteString(" " + e.Code)
	}
	if e.Message != "" {
		b.WriteString(": " + e.Message)
	}
	return b.String()
}

// IsCode reports whether err is (or wraps) an *Error with this code.
func IsCode(err error, code string) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == code
}

// StatusOf is the HTTP status of the *Error err is or wraps, else 0.
func StatusOf(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.Status
	}
	return 0
}

// maxErrorBody caps how much of an error answer is read.
const maxErrorBody = 64 << 10

// errorFrom reads the API's envelope, {"error": "...", "code": "..."}, or
// whatever text the answer has.
func errorFrom(resp *http.Response, body []byte) *Error {
	e := &Error{Status: resp.StatusCode, RetryAfter: retryAfter(resp.Header.Get("Retry-After"))}
	if resp.Request != nil {
		e.Method, e.Path = resp.Request.Method, resp.Request.URL.Path
	}
	var env struct {
		Error   json.RawMessage `json:"error"`
		Code    string          `json:"code"`
		Message string          `json:"message"`
	}
	if json.Unmarshal(body, &env) == nil {
		e.Code = env.Code
		var s string
		var obj struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		}
		switch {
		case json.Unmarshal(env.Error, &s) == nil && s != "":
			e.Message = s
		case json.Unmarshal(env.Error, &obj) == nil && obj.Message != "":
			e.Message = obj.Message
			if e.Code == "" {
				e.Code = obj.Code
			}
		default:
			e.Message = env.Message
		}
		if e.Message != "" || e.Code != "" {
			return e
		}
	}
	text := strings.TrimSpace(string(body))
	if len(text) > 300 {
		text = text[:300] + "…"
	}
	if text == "" {
		text = http.StatusText(resp.StatusCode)
	}
	e.Message = text
	return e
}

// retryAfter reads a Retry-After header in seconds or as an HTTP date.
func retryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if s, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && s >= 0 {
		return time.Duration(s) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// transportError is a failure to send a request or to read its answer: the
// kind of failure a retry can get past.
type transportError struct {
	err error
	// unsent is true when the request never reached the server (no
	// connection was made), so even a call that is not idempotent can be
	// sent again.
	unsent bool
}

func (e *transportError) Error() string { return e.err.Error() }
func (e *transportError) Unwrap() error { return e.err }

func wrapf(format string, args ...any) error { return fmt.Errorf("cere: "+format, args...) }

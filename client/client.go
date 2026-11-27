package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"
)

// DefaultBaseURL is the API of the devnet cluster (Dragon 1 devnet), the
// cluster that serves sandboxes today. The `cere` CLI and Python SDK use the
// same default.
const DefaultBaseURL = "https://agent-gateway.compute.dev.ddcdragon.com"

// DefaultTimeout bounds one try of an ordinary call. Calls that take longer
// by nature (a sandbox create, an exec, a stream) set their own.
const DefaultTimeout = 60 * time.Second

// Response size caps: a larger answer is an error, not unbounded memory use.
const (
	// DefaultMaxResponse caps a JSON answer unless the request sets its own.
	DefaultMaxResponse = 32 << 20
	// maxEventBytes caps one event of a stream.
	maxEventBytes = 32 << 20
)

// Client calls the developer platform's API. It is safe for concurrent use;
// WithToken and WithTokenSource return copies that share the HTTP client.
type Client struct {
	baseURL   string
	tokens    TokenSource
	http      *http.Client
	timeout   time.Duration
	retry     RetryPolicy
	userAgent string
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL sets the API's base URL, without /v1 (default DefaultBaseURL).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithToken authorizes every call with one encoded access token.
func WithToken(encoded string) Option {
	return func(c *Client) {
		if encoded = strings.TrimSpace(encoded); encoded == "" {
			c.tokens = nil
		} else {
			c.tokens = StaticToken(encoded)
		}
	}
}

// WithTokenSource authorizes each call with the token ts gives at call time,
// for example a *Delegation.
func WithTokenSource(ts TokenSource) Option { return func(c *Client) { c.tokens = ts } }

// WithHTTPClient sends the calls through h (default: a new http.Client with no
// overall timeout; the Client bounds each call itself).
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// WithTimeout bounds one try of an ordinary call (default DefaultTimeout).
func WithTimeout(d time.Duration) Option { return func(c *Client) { c.timeout = d } }

// WithRetry sets the retry policy (default DefaultRetryPolicy; NoRetry turns
// retries off).
func WithRetry(p RetryPolicy) Option { return func(c *Client) { c.retry = p } }

// WithUserAgent puts product (for example "my-service/1.2") before the SDK's
// own User-Agent.
func WithUserAgent(product string) Option {
	return func(c *Client) { c.userAgent = strings.TrimSpace(product + " " + sdkUserAgent()) }
}

// New returns a Client. It needs no token to be made: a service that acts for
// many buckets makes one Client and calls WithToken for each.
func New(opts ...Option) (*Client, error) {
	c := &Client{
		baseURL:   DefaultBaseURL,
		timeout:   DefaultTimeout,
		retry:     DefaultRetryPolicy,
		userAgent: sdkUserAgent(),
	}
	for _, opt := range opts {
		opt(c)
	}
	c.baseURL = strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	u, err := url.Parse(c.baseURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return nil, fmt.Errorf("cere: base URL %q: want https://host[:port][/path]", c.baseURL)
	}
	if strings.HasSuffix(u.Path, "/v1") {
		return nil, fmt.Errorf("cere: base URL %q: give it without /v1", c.baseURL)
	}
	if c.http == nil {
		c.http = &http.Client{}
	}
	if c.timeout <= 0 {
		c.timeout = DefaultTimeout
	}
	if c.retry.MaxAttempts < 1 {
		c.retry.MaxAttempts = 1
	}
	return c, nil
}

// BaseURL is the API's base URL.
func (c *Client) BaseURL() string { return c.baseURL }

// TokenSource is the client's token source; nil when it has none.
func (c *Client) TokenSource() TokenSource { return c.tokens }

// WithToken returns a copy of c that authorizes its calls with encoded, for
// example the token of one run.
func (c *Client) WithToken(encoded string) *Client {
	cp := *c
	WithToken(encoded)(&cp)
	return &cp
}

// WithTokenSource returns a copy of c that authorizes its calls with ts.
func (c *Client) WithTokenSource(ts TokenSource) *Client {
	cp := *c
	cp.tokens = ts
	return &cp
}

// Request is one call, as a service package describes it.
type Request struct {
	// Method and Path ("/v1/sandboxes") name the route.
	Method, Path string
	// Query is the query string.
	Query url.Values
	// JSON, when not nil, is sent as the JSON body.
	JSON any
	// Body, when not nil and JSON is nil, is sent as it is, with ContentType
	// (default application/octet-stream).
	Body        []byte
	ContentType string
	// Accept is the Accept header (default application/json).
	Accept string
	// Header holds headers to send beside the client's own (Authorization,
	// Accept, Content-Type, User-Agent, which it sets itself), for example
	// Last-Event-ID.
	Header http.Header
	// IdempotencyKey, when set, is sent as the Idempotency-Key header. The
	// API answers a repeat of such a call with the first one's result instead
	// of acting again, so a POST with a key is retried as a PUT is.
	IdempotencyKey string
	// Timeout bounds one try, reading the answer included: 0 is the client's
	// timeout, a negative value none (the context still applies).
	Timeout time.Duration
	// RetryOn lists codes that mean the API did nothing, so even a POST may be
	// sent again (for a sandbox create, "no_capacity").
	RetryOn []string
	// MaxResponse caps the answer's size in bytes (default DefaultMaxResponse).
	MaxResponse int64

	// stream, set by Stream, marks a call whose answer is an event stream:
	// once it has been read from, it is never sent again.
	stream bool
	// headerIdle, set by Stream, bounds the wait for the answer's headers.
	headerIdle time.Duration
}

// Headers of the API's idempotent calls and resumable streams.
const (
	// HeaderIdempotencyKey carries Request.IdempotencyKey.
	HeaderIdempotencyKey = "Idempotency-Key"
	// HeaderIdempotentReplayed is "true" on the answer to a repeated call
	// with an Idempotency-Key (Response.Replayed).
	HeaderIdempotentReplayed = "Idempotent-Replayed"
	// HeaderLastEventID is the id of the last event a client read of a
	// stream it reconnects to.
	HeaderLastEventID = "Last-Event-ID"
)

// Response is the status and headers of a 2xx answer.
type Response struct {
	Status int
	Header http.Header
}

// Replayed reports whether the API answered a call that carried an
// Idempotency-Key with the result of an earlier call with the same key
// (Idempotent-Replayed: true), instead of acting on it. The earlier call may
// be this call's own first try, retried after its answer was lost.
func (r *Response) Replayed() bool {
	return r != nil && strings.EqualFold(r.Header.Get(HeaderIdempotentReplayed), "true")
}

// Do sends r and decodes a JSON answer into out (unless out is nil).
func (c *Client) Do(ctx context.Context, r *Request, out any) error {
	_, err := c.DoResponse(ctx, r, out)
	return err
}

// DoResponse is Do that also returns the answer's status and headers.
func (c *Client) DoResponse(ctx context.Context, r *Request, out any) (*Response, error) {
	var res *Response
	err := c.roundTrip(ctx, r, func(resp *http.Response) error {
		raw, err := readCapped(resp.Body, r.maxResponse())
		if err != nil {
			return err
		}
		res = &Response{Status: resp.StatusCode, Header: resp.Header.Clone()}
		if out == nil || len(bytes.TrimSpace(raw)) == 0 {
			return nil
		}
		if err := json.Unmarshal(raw, out); err != nil {
			return wrapf("%s %s: decode the answer: %w", r.Method, r.Path, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Bytes sends r and returns the answer's body as it is.
func (c *Client) Bytes(ctx context.Context, r *Request) ([]byte, error) {
	var body []byte
	err := c.roundTrip(ctx, r, func(resp *http.Response) error {
		var err error
		body, err = readCapped(resp.Body, r.maxResponse())
		return err
	})
	return body, err
}

// Event is one event of a text/event-stream answer.
type Event struct {
	// Name is the event's name ("message" when the stream names none).
	Name string
	// ID is the stream's last event id when the event was sent: the id the
	// event carries, else the last one an earlier event carried (empty when
	// none has). A client that reconnects sends it as Last-Event-ID.
	ID string
	// Data is the event's data: JSON on every stream of the API.
	Data json.RawMessage
}

// ErrStreamIdle is returned (wrapped) when a stream sends nothing, not even
// its keep-alive, for longer than the caller allows.
var ErrStreamIdle = errors.New("the stream went silent")

// Stream sends r, whose answer is an event stream, and calls fn with each
// event until the stream ends, fn returns an error (io.EOF ends the stream
// without one), or nothing arrives for idle (0: no limit), the wait for the
// answer's headers included. An "error" event becomes an *Error with status
// 502. Only the opening of the stream is retried, never a stream that has
// started.
func (c *Client) Stream(ctx context.Context, r *Request, idle time.Duration, fn func(Event) error) error {
	rr := *r
	if rr.Accept == "" {
		rr.Accept = "text/event-stream"
	}
	if rr.Timeout == 0 {
		rr.Timeout = -1 // a stream lasts as long as it lasts; idle bounds the silence
	}
	rr.stream, rr.headerIdle = true, idle
	return c.roundTrip(ctx, &rr, func(resp *http.Response) error {
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
			return &Error{Status: http.StatusBadGateway, Method: rr.Method, Path: rr.Path,
				Message: fmt.Sprintf("expected text/event-stream, got %q", ct), notStream: true}
		}
		body := io.Reader(resp.Body)
		var silent atomic.Bool
		var timer *time.Timer
		if idle > 0 {
			timer = time.AfterFunc(idle, func() {
				silent.Store(true)
				_ = resp.Body.Close()
			})
			body = &idleReader{r: resp.Body, t: timer, d: idle}
		}
		err := readEvents(body, func(ev Event) error {
			if ev.Name == "error" {
				return streamError(&rr, ev.Data)
			}
			return fn(ev)
		})
		if timer != nil {
			timer.Stop()
		}
		if err != nil && silent.Load() {
			return wrapf("%s %s: %w for %s", rr.Method, rr.Path, ErrStreamIdle, idle)
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	})
}

func streamError(r *Request, data json.RawMessage) *Error {
	e := &Error{Status: http.StatusBadGateway, Method: r.Method, Path: r.Path}
	var body struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if json.Unmarshal(data, &body) == nil && body.Error != "" {
		e.Message, e.Code = body.Error, body.Code
	} else {
		e.Message = strings.TrimSpace(string(data))
	}
	return e
}

// idleReader resets its timer whenever bytes arrive.
type idleReader struct {
	r io.Reader
	t *time.Timer
	d time.Duration
}

func (i *idleReader) Read(p []byte) (int, error) {
	n, err := i.r.Read(p)
	if n > 0 {
		i.t.Reset(i.d)
	}
	return n, err
}

func (r *Request) maxResponse() int64 {
	if r.MaxResponse > 0 {
		return r.MaxResponse
	}
	return DefaultMaxResponse
}

// roundTrip sends r, with retries, and hands a 2xx answer to read. A read
// that has started is never repeated for a stream; for other calls the retry
// rules decide (see RetryPolicy).
func (c *Client) roundTrip(ctx context.Context, r *Request, read func(*http.Response) error) error {
	var body []byte
	contentType := ""
	switch {
	case r.JSON != nil:
		b, err := json.Marshal(r.JSON)
		if err != nil {
			return wrapf("%s %s: encode the request: %w", r.Method, r.Path, err)
		}
		body, contentType = b, "application/json"
	case r.Body != nil:
		body, contentType = r.Body, r.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}
	for attempt := 1; ; attempt++ {
		started, err := c.try(ctx, r, body, contentType, read)
		if err == nil {
			return nil
		}
		if started || attempt >= c.retry.MaxAttempts || ctx.Err() != nil || !shouldRetry(r, err) {
			return err
		}
		if serr := sleep(ctx, c.retry.backoff(attempt, err)); serr != nil {
			// Both: the context's end, and what the last try answered.
			return fmt.Errorf("cere: %s %s: %w; the last try: %w", r.Method, r.Path, serr, err)
		}
	}
}

// try sends r once. started is true once a stream's events have begun, which
// makes the call impossible to repeat.
func (c *Client) try(ctx context.Context, r *Request, body []byte, contentType string,
	read func(*http.Response) error) (started bool, err error) {
	token, err := c.token(ctx)
	if err != nil {
		return false, err
	}
	timeout := r.Timeout
	if timeout == 0 {
		timeout = c.timeout
	}
	tctx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		tctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	u := c.baseURL + r.Path
	if len(r.Query) > 0 {
		u += "?" + r.Query.Encode()
	}
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(tctx, r.Method, u, rd)
	if err != nil {
		return false, wrapf("%s %s: %w", r.Method, r.Path, err)
	}
	for k, vs := range r.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if r.IdempotencyKey != "" {
		req.Header.Set(HeaderIdempotencyKey, r.IdempotencyKey)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	accept := r.Accept
	if accept == "" {
		accept = "application/json"
	}
	req.Header.Set("Accept", accept)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.send(req, r.headerIdle)
	if err != nil {
		if ctx.Err() != nil {
			return false, wrapf("%s %s: %w", r.Method, r.Path, ctx.Err())
		}
		if errors.Is(err, ErrStreamIdle) {
			return false, &transportError{err: wrapf("%s %s: %w", r.Method, r.Path, err)}
		}
		return false, &transportError{err: wrapf("%s %s: %w", r.Method, r.Path, err), unsent: unsent(req, err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return false, errorFrom(resp, raw)
	}
	if err := read(resp); err != nil {
		// A stream the API accepted is never sent again: an exec's command
		// has started.
		started := r.stream
		if ctx.Err() != nil {
			return started, wrapf("%s %s: %w", r.Method, r.Path, ctx.Err())
		}
		var te *transportError
		if errors.As(err, &te) {
			te.err = wrapf("%s %s: %w", r.Method, r.Path, te.err)
		}
		return started, err
	}
	return false, nil
}

// send sends req. A headerIdle above zero bounds the wait for the answer's
// headers: the request is cancelled after it, and the error wraps
// ErrStreamIdle.
func (c *Client) send(req *http.Request, headerIdle time.Duration) (*http.Response, error) {
	if headerIdle <= 0 {
		return c.http.Do(req)
	}
	ctx, cancel := context.WithCancelCause(req.Context())
	timer := time.AfterFunc(headerIdle, func() {
		cancel(fmt.Errorf("%w for %s before the answer's headers", ErrStreamIdle, headerIdle))
	})
	resp, err := c.http.Do(req.WithContext(ctx))
	if !timer.Stop() {
		// The limit passed: the request is cancelled, even if the headers
		// had just arrived.
		if err == nil {
			_ = resp.Body.Close()
		}
		return nil, context.Cause(ctx)
	}
	if err != nil {
		cancel(nil)
		return nil, err
	}
	// The headers arrived: the stream's own idle limit takes over. The body
	// is read under ctx, which is cancelled when the body is closed.
	resp.Body = &cancelOnClose{ReadCloser: resp.Body, cancel: func() { cancel(nil) }}
	return resp, nil
}

// cancelOnClose cancels a request's context when its body is closed.
type cancelOnClose struct {
	io.ReadCloser
	cancel func()
}

func (b *cancelOnClose) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}

// token picks the token for one try.
func (c *Client) token(ctx context.Context) (string, error) {
	if c.tokens == nil {
		return "", ErrNoToken
	}
	tok, err := c.tokens.Token(ctx)
	if err != nil {
		if strings.HasPrefix(err.Error(), "cere: ") {
			return "", err
		}
		return "", wrapf("access token: %w", err)
	}
	if tok = strings.TrimSpace(tok); tok == "" {
		return "", ErrNoToken
	}
	return tok, nil
}

// readCapped reads at most limit bytes; a failure to read is a transport
// error.
func readCapped(r io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, &transportError{err: fmt.Errorf("read the answer: %w", err)}
	}
	if int64(len(b)) > limit {
		return nil, wrapf("the answer is larger than %d bytes", limit)
	}
	return b, nil
}

// modulePath is this module's path, to find its version in the build.
const modulePath = "github.com/cerebellum-network/cere-ddc-sdk-go/client"

// sdkUserAgent is "cere-sdk-go/<module version>".
func sdkUserAgent() string {
	version := "devel"
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, m := range bi.Deps {
			if m.Path == modulePath {
				version = strings.TrimPrefix(m.Version, "v")
				if m.Replace != nil {
					version = "devel"
				}
			}
		}
	}
	return "cere-sdk-go/" + version
}

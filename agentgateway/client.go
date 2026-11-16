package agentgateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// authTokenHeader is the header DDC services read an access token from. It is
// the same value as auth.AuthTokenHeader; this module does not import auth.
const authTokenHeader = "X-Auth-Token"

// DefaultInvokeTimeout is the ceiling on one invoke when WithInvokeTimeout is
// not set: longer than any run a runtime allows, and short enough that a
// gateway that accepts the call and never answers does not hold the caller
// forever.
const DefaultInvokeTimeout = 30 * time.Minute

// Client is the agent gateway API. HTTPClient implements it.
type Client interface {
	Invoke(ctx context.Context, req InvokeRequest, authToken []byte) (*InvokeResponse, error)
	QueryLogs(ctx context.Context, q LogQuery) (*LogQueryResponse, error)
	QueryMetrics(ctx context.Context, q MetricsQuery) (*MetricsQueryResponse, error)
	Health(ctx context.Context) error
}

var _ Client = (*HTTPClient)(nil)

// StatusError is returned when the gateway answers with a status other than
// 200. Body is the response body, for diagnostics.
type StatusError struct {
	Path       string
	StatusCode int
	Body       string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("agent-gateway %s returned %d: %s", e.Path, e.StatusCode, e.Body)
}

// HTTPClient calls one agent gateway over HTTP.
type HTTPClient struct {
	baseURL string
	// client serves the log, metrics and health reads, under the fixed
	// per-request timeout.
	client *http.Client
	// invoke is bounded by the caller's context instead: a run can last far
	// longer than a read. Its Timeout is the ceiling, which a caller's deadline
	// can only shorten.
	invoke    *http.Client
	invokeMax time.Duration
}

// Option customizes an HTTPClient.
type Option func(*HTTPClient)

// NewClient returns a client for the gateway at baseURL. timeoutSeconds bounds
// each log, metrics and health request; invokes are bounded by their context
// and by WithInvokeTimeout.
func NewClient(baseURL string, timeoutSeconds int, opts ...Option) *HTTPClient {
	c := &HTTPClient{
		baseURL:   baseURL,
		client:    &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
		invoke:    &http.Client{Timeout: DefaultInvokeTimeout},
		invokeMax: DefaultInvokeTimeout,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithInvokeTimeout sets the ceiling on one invoke, and the deadline of an
// invoke whose context has none. Set it at or above the longest run you allow
// plus a response margin. Values <= 0 are ignored.
func WithInvokeTimeout(d time.Duration) Option {
	return func(c *HTTPClient) {
		if d > 0 {
			c.invoke.Timeout = d
			c.invokeMax = d
		}
	}
}

// WithTransport replaces the HTTP transport.
//
// Set it on any caller with real concurrency: http.DefaultTransport keeps two
// idle connections per host, so a worker pool sharing this client queues
// behind TCP and TLS setup, which looks like gateway latency.
func WithTransport(rt http.RoundTripper) Option {
	return func(c *HTTPClient) {
		c.client.Transport = rt
		c.invoke.Transport = rt
	}
}

// BaseURL returns the gateway URL the client was built with.
func (c *HTTPClient) BaseURL() string { return c.baseURL }

// Invoke runs one handler of an agent bundle and returns its outcome.
//
// authToken is the DDC access token chain for the bucket that pays for the
// run, as protobuf bytes (pb.AuthToken marshaled); the runtime rejects a call
// without one with 401. Invoke is bounded by ctx, not by the per-request
// timeout, so give ctx a deadline that outlasts req.Limits.TimeoutMs. A ctx
// with no deadline gets the invoke ceiling.
//
// A 200 whose Status is "error" is returned as a response, not an error: the
// handler ran and failed. Any other status is a *StatusError.
func (c *HTTPClient) Invoke(ctx context.Context, req InvokeRequest, authToken []byte) (*InvokeResponse, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.invokeMax)
		defer cancel()
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal invoke request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/invoke", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if len(authToken) > 0 {
		httpReq.Header.Set(authTokenHeader, base64.StdEncoding.EncodeToString(authToken))
	}

	var out InvokeResponse
	if err := c.do(c.invoke, httpReq, "/v1/invoke", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// QueryLogs returns one page of an agent bundle's log records.
func (c *HTTPClient) QueryLogs(ctx context.Context, q LogQuery) (*LogQueryResponse, error) {
	v := url.Values{}
	v.Set("bucket", q.BucketID)
	v.Set("cid", q.CID)
	v.Set("since", q.Since.UTC().Format(time.RFC3339Nano))
	v.Set("until", q.Until.UTC().Format(time.RFC3339Nano))
	if q.Level != "" {
		v.Set("level", q.Level)
	}
	if q.Q != "" {
		v.Set("q", q.Q)
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Cursor != "" {
		v.Set("cursor", q.Cursor)
	}
	for label, val := range q.Labels {
		if _, reserved := logQueryReserved[label]; reserved {
			continue
		}
		v.Set(label, val)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/logs?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if q.AuthToken != "" {
		httpReq.Header.Set(authTokenHeader, q.AuthToken)
	}
	var out LogQueryResponse
	if err := c.do(c.client, httpReq, "/v1/logs", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// logQueryReserved are LogQuery's own parameter names. A label with one of
// these names is dropped so it cannot shadow the parameter.
var logQueryReserved = map[string]struct{}{
	"bucket": {}, "cid": {}, "since": {}, "until": {},
	"level": {}, "q": {}, "limit": {}, "cursor": {},
}

// metricsQueryReserved are MetricsQuery's own parameter names. A label with one
// of these names is dropped so it cannot shadow, or be shadowed by, the
// parameter.
var metricsQueryReserved = map[string]struct{}{
	"bucket": {}, "cid": {}, "metric": {}, "groupBy": {},
	"from": {}, "to": {}, "step": {},
}

// QueryMetrics returns an agent bundle's approximate resource use. The gateway
// rejects a metric or label it does not allow with a 400 *StatusError.
func (c *HTTPClient) QueryMetrics(ctx context.Context, q MetricsQuery) (*MetricsQueryResponse, error) {
	v := url.Values{}
	v.Set("bucket", q.Bucket)
	v.Set("cid", q.CID)
	v.Set("metric", q.Metric)
	for label, val := range q.Labels {
		if _, reserved := metricsQueryReserved[label]; reserved || val == "" {
			continue
		}
		v.Set(label, val)
	}
	if q.GroupBy != "" {
		v.Set("groupBy", q.GroupBy)
	}
	if !q.From.IsZero() {
		v.Set("from", q.From.UTC().Format(time.RFC3339Nano))
	}
	if !q.To.IsZero() {
		v.Set("to", q.To.UTC().Format(time.RFC3339Nano))
	}
	if q.Step != "" {
		v.Set("step", q.Step)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/metrics?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if q.AuthToken != "" {
		httpReq.Header.Set(authTokenHeader, q.AuthToken)
	}
	var out MetricsQueryResponse
	if err := c.do(c.client, httpReq, "/v1/metrics", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Health reports whether the gateway answers GET /health with 200.
func (c *HTTPClient) Health(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	return c.do(c.client, httpReq, "/health", nil)
}

// do sends req and decodes a 200 body into out (when out is not nil).
func (c *HTTPClient) do(client *http.Client, req *http.Request, path string, out any) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("agent-gateway %s transport: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("agent-gateway %s read: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return &StatusError{Path: path, StatusCode: resp.StatusCode, Body: string(body)}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("agent-gateway %s decode: %w", path, err)
	}
	return nil
}

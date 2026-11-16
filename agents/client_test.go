package agents

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpClient_Invoke_ForwardsBody(t *testing.T) {
	var seenURL, seenBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenURL = r.URL.Path
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		seenBody = string(buf)
		_, _ = w.Write([]byte(`{"status":"success","result":42}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5, WithToken(testToken))
	resp, err := c.Invoke(context.Background(), InvokeRequest{
		Bundle:  BundleRef{BucketID: "b", CID: "c"},
		Handler: "onMessage",
	})
	require.NoError(t, err)
	assert.Equal(t, "/v1/invoke", seenURL)
	assert.Contains(t, seenBody, `"bucketId":"b"`)
	assert.Equal(t, "success", resp.Status)
}

func TestHttpClient_QueryMetrics_EncodesParams(t *testing.T) {
	var seen url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Query()
		assert.Equal(t, "/v1/metrics", r.URL.Path)
		_, _ = w.Write([]byte(`{"metric":"agent_invocations_total","query":"sum(...)","data":{"resultType":"vector","result":[]}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5, WithToken(testToken))
	resp, err := c.QueryMetrics(context.Background(), MetricsQuery{
		Bucket: "b", CID: "c",
		Metric: "agent_invocations_total",
		Labels: map[string]string{"outcome": "success"}, GroupBy: "handler",
	})
	require.NoError(t, err)
	assert.Equal(t, "b", seen.Get("bucket"))
	assert.Equal(t, "c", seen.Get("cid"))
	assert.Equal(t, "agent_invocations_total", seen.Get("metric"))
	assert.Equal(t, "success", seen.Get("outcome"))
	assert.Equal(t, "handler", seen.Get("groupBy"))
	assert.Equal(t, "agent_invocations_total", resp.Metric)
	assert.Contains(t, string(resp.Data), "resultType")
}

func TestHttpClient_QueryMetrics_400MapsToError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"vault scope is exact-only"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5, WithToken(testToken))
	_, err := c.QueryMetrics(context.Background(), MetricsQuery{Bucket: "b", CID: "c", Metric: "agent_invocations_total"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestHttpClient_Invoke_503MapsToError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"error","error":{"code":"NO_RUNTIMES","message":"...","retryable":false}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5, WithToken(testToken))
	_, err := c.Invoke(context.Background(), InvokeRequest{
		Bundle:  BundleRef{BucketID: "b", CID: "c"},
		Handler: "h",
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "503"))
}

// An invocation is bounded by its context, not by the client's per-request
// timeout: a run longer than that timeout but inside its own deadline must
// complete. (The chart default of 30s cut every longer run short while the
// runtime kept executing it.)
func TestHttpClient_Invoke_OutlivesTheRequestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(1200 * time.Millisecond)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 1, WithToken(testToken), WithTransport(http.DefaultTransport))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := c.Invoke(ctx, InvokeRequest{
		Bundle: BundleRef{BucketID: "b", CID: "c"}, Handler: "h",
	})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
}

// deadlineRecorder answers every request and records whether its context
// carried a deadline when it reached the transport.
type deadlineRecorder struct{ deadline time.Time }

func (d *deadlineRecorder) RoundTrip(r *http.Request) (*http.Response, error) {
	d.deadline, _ = r.Context().Deadline()
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"status":"success"}`)),
		Header:     http.Header{},
		Request:    r,
	}, nil
}

// An invoke whose context has no deadline still gets one: the invoke ceiling.
// A caller that forgot one would otherwise wait for ever on a gateway that
// accepted the call and never answered.
func TestHttpClient_Invoke_NoDeadlineGetsTheCeiling(t *testing.T) {
	rec := &deadlineRecorder{}
	c := NewClient("http://ag.test", 1, WithToken(testToken), WithTransport(rec), WithInvokeTimeout(7*time.Minute))
	// The client Timeout alone also puts a deadline on the request; take it
	// away so this sees only Invoke's own fallback.
	c.invoke.Timeout = 0
	start := time.Now()
	_, err := c.Invoke(context.Background(), InvokeRequest{
		Bundle: BundleRef{BucketID: "b", CID: "c"}, Handler: "h",
	})
	require.NoError(t, err)
	require.False(t, rec.deadline.IsZero(), "the request carried a deadline")
	assert.WithinDuration(t, start.Add(7*time.Minute), rec.deadline, time.Minute)

	// A caller's own deadline is kept, not replaced.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	want, _ := ctx.Deadline()
	_, err = c.Invoke(ctx, InvokeRequest{
		Bundle: BundleRef{BucketID: "b", CID: "c"}, Handler: "h",
	})
	require.NoError(t, err)
	assert.Equal(t, want, rec.deadline)
}

// The invoke client's own Timeout is the ceiling: a gateway that never answers
// releases the call even when the caller's deadline is longer.
func TestHttpClient_Invoke_ClientTimeoutIsTheCeiling(t *testing.T) {
	stop := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-stop:
		}
	}))
	defer srv.Close()
	defer close(stop)

	c := NewClient(srv.URL, 1, WithToken(testToken), WithTransport(http.DefaultTransport), WithInvokeTimeout(100*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	start := time.Now()
	_, err := c.Invoke(ctx, InvokeRequest{
		Bundle: BundleRef{BucketID: "b", CID: "c"}, Handler: "h",
	})
	require.Error(t, err)
	assert.Less(t, time.Since(start), 10*time.Second)

	assert.Equal(t, DefaultInvokeTimeout, NewClient(srv.URL, 1).invoke.Timeout, "the default ceiling is set")
}

func TestHttpClient_StatusErrorCarriesTheStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"token required"}`))
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL, 5, WithToken(testToken)).QueryLogs(context.Background(), LogQuery{BucketID: "b", CID: "c"})
	var statusErr *StatusError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusUnauthorized, statusErr.StatusCode)
	assert.Equal(t, "/v1/logs", statusErr.Path)
	assert.Contains(t, statusErr.Body, "token required")
}

// The call's token wins over the client's; with neither, nothing is sent. On
// the wire every token is base64 of its protobuf bytes, the only form the agent
// runtime behind the gateway reads.
func TestHttpClient_TokenComesFromTheCallOrTheClient(t *testing.T) {
	var headers []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = append(headers, r.Header.Get("X-Auth-Token"))
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer srv.Close()
	ctx := context.Background()
	req := InvokeRequest{Bundle: BundleRef{BucketID: "b", CID: "c"}, Handler: "h"}
	encode := func(s string) string { return base58.Encode([]byte(s)) }
	wire := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

	_, err := NewClient(srv.URL, 5).Invoke(ctx, req)
	require.ErrorIs(t, err, ErrNoToken)
	_, err = NewClient(srv.URL, 5).QueryLogs(ctx, LogQuery{BucketID: "b", CID: "c"})
	require.ErrorIs(t, err, ErrNoToken)
	_, err = NewClient(srv.URL, 5).QueryMetrics(ctx, MetricsQuery{Bucket: "b", CID: "c", Metric: "m"})
	require.ErrorIs(t, err, ErrNoToken)
	_, err = NewClient(srv.URL, 5, WithToken("CgH/")).Invoke(ctx, req)
	require.ErrorIs(t, err, ErrInvalidToken, "base64 is not what auth.EncodeToken makes")
	assert.Empty(t, headers, "a call without a usable token never reaches the gateway")

	client := NewClient(srv.URL, 5, WithToken(encode("client")))
	_, err = client.Invoke(ctx, req)
	require.NoError(t, err)
	_, err = client.Invoke(ctx, req, CallToken(encode("run")))
	require.NoError(t, err)
	_, err = client.QueryLogs(ctx, LogQuery{BucketID: "b", CID: "c"})
	require.NoError(t, err)

	calls := 0
	refreshed := NewClient(srv.URL, 5, WithTokenSource(func(context.Context) (string, error) {
		calls++
		return encode(fmt.Sprintf("fresh-%d", calls)), nil
	}))
	for range 2 {
		_, err = refreshed.Invoke(ctx, req)
		require.NoError(t, err)
	}
	assert.Equal(t, []string{wire("client"), wire("run"), wire("client"), wire("fresh-1"), wire("fresh-2")}, headers)
}

// testToken is a valid auth.EncodeToken-shaped (base58) token.
var testToken = base58.Encode([]byte("token"))

func TestHttpClient_LogQueryEncodesParamsAndDropsReservedLabels(t *testing.T) {
	var query url.Values
	var token string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query, token = r.URL.Query(), r.Header.Get("X-Auth-Token")
		_, _ = w.Write([]byte(`{"records":[{"msg":"hi"}],"cursor":"next"}`))
	}))
	defer srv.Close()

	since := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	resp, err := NewClient(srv.URL, 5, WithToken(testToken)).QueryLogs(context.Background(), LogQuery{
		BucketID: "1337", CID: "bafy", Since: since, Until: since.Add(time.Hour),
		Limit: 50, Labels: map[string]string{"taskId": "t1", "bucket": "shadow"},
	}, CallToken(base58.Encode([]byte("encoded"))))
	require.NoError(t, err)
	assert.Equal(t, "1337", query.Get("bucket"), "a label cannot shadow a reserved param")
	assert.Equal(t, "t1", query.Get("taskId"))
	assert.Equal(t, "50", query.Get("limit"))
	assert.Equal(t, "2026-11-01T00:00:00Z", query.Get("since"))
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("encoded")), token)
	assert.Equal(t, "next", resp.Cursor)
	assert.Len(t, resp.Records, 1)
}

func TestHttpClient_Health(t *testing.T) {
	healthy := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" || !healthy {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5, WithToken(testToken))
	require.NoError(t, c.Health(context.Background()))
	healthy = false
	var statusErr *StatusError
	require.ErrorAs(t, c.Health(context.Background()), &statusErr)
	assert.Equal(t, http.StatusServiceUnavailable, statusErr.StatusCode)
}

func TestHttpClient_MetricsQueryDropsReservedLabels(t *testing.T) {
	var query url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL, 5, WithToken(testToken)).QueryMetrics(context.Background(), MetricsQuery{
		Bucket: "1", CID: "bafy", Metric: "cpu", GroupBy: "taskId",
		Labels: map[string]string{"bucket": "2", "metric": "mem", "groupBy": "x", "taskId": "t1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "1", query.Get("bucket"), "a label cannot shadow a reserved param")
	assert.Equal(t, "cpu", query.Get("metric"))
	assert.Equal(t, []string{"taskId"}, query["groupBy"])
	assert.Equal(t, "t1", query.Get("taskId"))
}

// Golden wire value. goldenToken is a real signed token (ed25519, seed
// 0x01..0x20, bucket 1337, COMPUTE) as auth.EncodeToken prints it. The agent
// runtime reads the header with base64.StdEncoding.DecodeString and unmarshals
// the bytes (ddc-node internal/compute/agentruntime/auth/middleware.go), so the
// header must be exactly goldenWire, on every call that carries a token.
const (
	goldenToken = "5HhgSrTeTGxf4p7snJpPFouPaJfuXog7tejvP3h43EacQJfcnXojQHmXqeQ16sCQEEaekCPgC75kXHByoudh2rDZAWikE6jzm5zL4BVhE9z5dQd74aKHdJULfyTKoeYjmuwzH2mB5uXC4jmJ4R5iKnQvN6vX7WYUe"
	goldenWire  = "CmQSIHm1Vi6P5lT5QHixEuipi6eQH4U65pW+1+DjkQutBJZkGkCWJCN+Crf+/IKDV9d8i8g2RwDQYIisMy8vSxgO7R2qgr9iXHRMVDKPSWhrpUX41+uG7gpnAVvNNDpK2i2uryEAEg4YACC5CioBBDCAse+GBw=="
)

func TestHttpClient_SendsTheTokenAsTheRuntimeReadsIt(t *testing.T) {
	var headers []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = append(headers, r.Header.Get("X-Auth-Token"))
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer srv.Close()
	ctx := context.Background()

	client := NewClient(srv.URL, 5, WithToken(goldenToken))
	_, err := client.Invoke(ctx, InvokeRequest{Bundle: BundleRef{BucketID: "1337", CID: "c"}, Handler: "h"})
	require.NoError(t, err)
	_, err = client.QueryLogs(ctx, LogQuery{BucketID: "1337", CID: "c"})
	require.NoError(t, err)
	_, err = client.QueryMetrics(ctx, MetricsQuery{Bucket: "1337", CID: "c", Metric: "m"})
	require.NoError(t, err)
	_, err = NewClient(srv.URL, 5).Invoke(ctx, InvokeRequest{Handler: "h"}, CallToken(goldenToken))
	require.NoError(t, err)

	assert.Equal(t, []string{goldenWire, goldenWire, goldenWire, goldenWire}, headers)
	for _, h := range headers {
		_, err := base64.StdEncoding.DecodeString(h)
		require.NoError(t, err, "the runtime's decoder accepts it")
	}
}

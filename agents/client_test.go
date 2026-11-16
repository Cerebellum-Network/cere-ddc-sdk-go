package agents

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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

	c := NewClient(srv.URL, 5)
	resp, err := c.Invoke(context.Background(), InvokeRequest{
		Bundle:  BundleRef{BucketID: "b", CID: "c"},
		Handler: "onMessage",
	}, nil)
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

	c := NewClient(srv.URL, 5)
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

	c := NewClient(srv.URL, 5)
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

	c := NewClient(srv.URL, 5)
	_, err := c.Invoke(context.Background(), InvokeRequest{
		Bundle:  BundleRef{BucketID: "b", CID: "c"},
		Handler: "h",
	}, nil)
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

	c := NewClient(srv.URL, 1, WithTransport(http.DefaultTransport))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := c.Invoke(ctx, InvokeRequest{
		Bundle: BundleRef{BucketID: "b", CID: "c"}, Handler: "h",
	}, nil)
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
	c := NewClient("http://ag.test", 1, WithTransport(rec), WithInvokeTimeout(7*time.Minute))
	// The client Timeout alone also puts a deadline on the request; take it
	// away so this sees only Invoke's own fallback.
	c.invoke.Timeout = 0
	start := time.Now()
	_, err := c.Invoke(context.Background(), InvokeRequest{
		Bundle: BundleRef{BucketID: "b", CID: "c"}, Handler: "h",
	}, nil)
	require.NoError(t, err)
	require.False(t, rec.deadline.IsZero(), "the request carried a deadline")
	assert.WithinDuration(t, start.Add(7*time.Minute), rec.deadline, time.Minute)

	// A caller's own deadline is kept, not replaced.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	want, _ := ctx.Deadline()
	_, err = c.Invoke(ctx, InvokeRequest{
		Bundle: BundleRef{BucketID: "b", CID: "c"}, Handler: "h",
	}, nil)
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

	c := NewClient(srv.URL, 1, WithTransport(http.DefaultTransport), WithInvokeTimeout(100*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	start := time.Now()
	_, err := c.Invoke(ctx, InvokeRequest{
		Bundle: BundleRef{BucketID: "b", CID: "c"}, Handler: "h",
	}, nil)
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

	_, err := NewClient(srv.URL, 5).QueryLogs(context.Background(), LogQuery{BucketID: "b", CID: "c"})
	var statusErr *StatusError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusUnauthorized, statusErr.StatusCode)
	assert.Equal(t, "/v1/logs", statusErr.Path)
	assert.Contains(t, statusErr.Body, "token required")
}

func TestHttpClient_InvokeSendsTheTokenAsBase64(t *testing.T) {
	var header string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Get("X-Auth-Token")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL, 5).Invoke(context.Background(), InvokeRequest{
		Bundle: BundleRef{BucketID: "b", CID: "c"}, Handler: "h",
	}, []byte{0x0a, 0x01, 0xff})
	require.NoError(t, err)
	assert.Equal(t, "CgH/", header)
}

func TestHttpClient_LogQueryEncodesParamsAndDropsReservedLabels(t *testing.T) {
	var query url.Values
	var token string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query, token = r.URL.Query(), r.Header.Get("X-Auth-Token")
		_, _ = w.Write([]byte(`{"records":[{"msg":"hi"}],"cursor":"next"}`))
	}))
	defer srv.Close()

	since := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	resp, err := NewClient(srv.URL, 5).QueryLogs(context.Background(), LogQuery{
		BucketID: "1337", CID: "bafy", Since: since, Until: since.Add(time.Hour),
		Limit: 50, Labels: map[string]string{"taskId": "t1", "bucket": "shadow"},
		AuthToken: "encoded",
	})
	require.NoError(t, err)
	assert.Equal(t, "1337", query.Get("bucket"), "a label cannot shadow a reserved param")
	assert.Equal(t, "t1", query.Get("taskId"))
	assert.Equal(t, "50", query.Get("limit"))
	assert.Equal(t, "2026-11-01T00:00:00Z", query.Get("since"))
	assert.Equal(t, "encoded", token)
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

	c := NewClient(srv.URL, 5)
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

	_, err := NewClient(srv.URL, 5).QueryMetrics(context.Background(), MetricsQuery{
		Bucket: "1", CID: "bafy", Metric: "cpu", GroupBy: "taskId",
		Labels: map[string]string{"bucket": "2", "metric": "mem", "groupBy": "x", "taskId": "t1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "1", query.Get("bucket"), "a label cannot shadow a reserved param")
	assert.Equal(t, "cpu", query.Get("metric"))
	assert.Equal(t, []string{"taskId"}, query["groupBy"])
	assert.Equal(t, "t1", query.Get("taskId"))
}

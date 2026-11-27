package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
)

var fastRetry = client.RetryPolicy{MaxAttempts: 3, MinBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond}

func newClient(t *testing.T, h http.Handler, opts ...client.Option) *client.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := client.New(append([]client.Option{client.WithBaseURL(srv.URL), client.WithToken("tok"),
		client.WithRetry(fastRetry)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNewChecksTheBaseURL(t *testing.T) {
	for _, u := range []string{"agent-gateway.example.com", "ftp://x", "https://", "https://x.example/v1"} {
		if _, err := client.New(client.WithBaseURL(u)); err == nil {
			t.Errorf("%q: no error", u)
		}
	}
	c, err := client.New(client.WithBaseURL("https://x.example/api/"))
	if err != nil || c.BaseURL() != "https://x.example/api" {
		t.Fatalf("%v %v", c, err)
	}
	c, err = client.New()
	if err != nil || c.BaseURL() != client.DefaultBaseURL {
		t.Fatalf("default: %v %v", c, err)
	}
}

func TestDoSendsTheTokenAndDecodes(t *testing.T) {
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" || r.Header.Get("Accept") != "application/json" ||
			r.Header.Get("Content-Type") != "application/json" || r.URL.Query().Get("q") != "1" {
			http.Error(w, "bad request headers", http.StatusTeapot)
			return
		}
		if ua := r.Header.Get("User-Agent"); !strings.HasPrefix(ua, "svc/1 cere-sdk-go/") {
			http.Error(w, "user agent "+ua, http.StatusTeapot)
			return
		}
		body, _ := io.ReadAll(r.Body)
		fmt.Fprintf(w, `{"echo": %s}`, body)
	}), client.WithUserAgent("svc/1"))
	var out struct {
		Echo struct{ A int } `json:"echo"`
	}
	err := c.Do(context.Background(), &client.Request{Method: http.MethodPost, Path: "/v1/x",
		Query: map[string][]string{"q": {"1"}}, JSON: map[string]int{"a": 7}}, &out)
	if err != nil || out.Echo.A != 7 {
		t.Fatalf("%+v %v", out, err)
	}
}

func TestNoToken(t *testing.T) {
	var calls atomic.Int32
	c := newClient(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	err := c.WithToken("").Do(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/x"}, nil)
	if !errors.Is(err, client.ErrNoToken) || calls.Load() != 0 {
		t.Fatalf("%v, %d calls", err, calls.Load())
	}
	// WithToken returns a copy: the original keeps its token.
	if err := c.Do(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/x"}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestErrors(t *testing.T) {
	cases := []struct {
		status            int
		body, code, msg   string
		header            string
		wantRetryAfterSec int
	}{
		{409, `{"error":"sandbox is terminated","code":"not_running"}`, "not_running", "sandbox is terminated", "", 0},
		{404, `{"message":"not found"}`, "", "not found", "", 0},
		{400, `{"error":{"message":"bad field","code":"invalid_request"}}`, "invalid_request", "bad field", "", 0},
		{403, `{"error":"bucket owner has insufficient balance","code":"insufficient_balance"}`,
			client.CodeInsufficientBalance, "bucket owner has insufficient balance", "", 0},
		{502, "upstream exploded", "", "upstream exploded", "", 0},
		{503, "", "", "Service Unavailable", "7", 7},
	}
	for _, tc := range cases {
		c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tc.header != "" {
				w.Header().Set("Retry-After", tc.header)
			}
			w.WriteHeader(tc.status)
			_, _ = io.WriteString(w, tc.body)
		}), client.WithRetry(client.NoRetry))
		err := c.Do(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/sandboxes/sb-1"}, nil)
		var e *client.Error
		if !errors.As(err, &e) {
			t.Fatalf("%d: %v", tc.status, err)
		}
		if e.Status != tc.status || e.Code != tc.code || e.Message != tc.msg || e.Path != "/v1/sandboxes/sb-1" ||
			e.RetryAfter != time.Duration(tc.wantRetryAfterSec)*time.Second {
			t.Errorf("%d: %+v", tc.status, e)
		}
		if client.StatusOf(err) != tc.status || (tc.code != "" && !client.IsCode(err, tc.code)) {
			t.Errorf("%d: StatusOf/IsCode", tc.status)
		}
	}
	e := &client.Error{Status: 409, Code: "not_running", Message: "m", Method: "POST", Path: "/v1/x"}
	if e.Error() != "cere: POST /v1/x: 409 not_running: m" {
		t.Errorf("%q", e.Error())
	}
}

// statusSeq answers with each status in turn (then 200), counting calls.
func statusSeq(calls *atomic.Int32, codes []string, statuses ...int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := int(calls.Add(1)) - 1
		if i < len(statuses) {
			code := ""
			if i < len(codes) {
				code = codes[i]
			}
			w.WriteHeader(statuses[i])
			fmt.Fprintf(w, `{"error":"try again","code":%q}`, code)
			return
		}
		_, _ = io.WriteString(w, `{}`)
	})
}

func TestRetries(t *testing.T) {
	cases := []struct {
		name      string
		method    string
		retryOn   []string
		codes     []string
		statuses  []int
		wantCalls int32
		wantErr   bool
	}{
		{"get on 502 and 503", http.MethodGet, nil, nil, []int{502, 503}, 3, false},
		{"get gives up after three", http.MethodGet, nil, nil, []int{502, 502, 502}, 3, true},
		{"put on 504", http.MethodPut, nil, nil, []int{504}, 2, false},
		{"get not on 500", http.MethodGet, nil, nil, []int{500}, 1, true},
		{"get not on 409", http.MethodGet, nil, nil, []int{409}, 1, true},
		{"post not on 502", http.MethodPost, nil, nil, []int{502}, 1, true},
		{"post not on a plain 503", http.MethodPost, nil, nil, []int{503}, 1, true},
		{"post on 429", http.MethodPost, nil, nil, []int{429}, 2, false},
		{"post on AUTH_UNAVAILABLE", http.MethodPost, nil, []string{"AUTH_UNAVAILABLE"}, []int{503}, 2, false},
		{"post on a listed code", http.MethodPost, []string{"no_capacity"}, []string{"no_capacity"}, []int{503}, 2, false},
		{"post not on an unlisted code", http.MethodPost, nil, []string{"no_capacity"}, []int{503}, 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			c := newClient(t, statusSeq(&calls, tc.codes, tc.statuses...))
			err := c.Do(context.Background(), &client.Request{Method: tc.method, Path: "/v1/x", RetryOn: tc.retryOn}, nil)
			if (err != nil) != tc.wantErr || calls.Load() != tc.wantCalls {
				t.Fatalf("err %v, %d calls; want error %v, %d calls", err, calls.Load(), tc.wantErr, tc.wantCalls)
			}
		})
	}
}

func TestRetryOnConnectionRefusedEvenForPost(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	u := srv.URL
	srv.Close() // nothing listens now
	// The token source is asked once per try, so it counts the tries.
	var tries atomic.Int32
	c, err := client.New(client.WithBaseURL(u), client.WithRetry(fastRetry),
		client.WithTokenSource(client.TokenFunc(func(context.Context) (string, error) { tries.Add(1); return "tok", nil })))
	if err != nil {
		t.Fatal(err)
	}
	err = c.Do(context.Background(), &client.Request{Method: http.MethodPost, Path: "/v1/x"}, nil)
	if err == nil || tries.Load() != 3 {
		t.Fatalf("%v after %d tries; want 3 tries", err, tries.Load())
	}
}

func TestRetryAfterIsHonouredUpToTheMaximum(t *testing.T) {
	var calls atomic.Int32
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = io.WriteString(w, `{}`)
	}), client.WithRetry(client.RetryPolicy{MaxAttempts: 2, MinBackoff: time.Millisecond, MaxBackoff: 50 * time.Millisecond}))
	start := time.Now()
	if err := c.Do(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/x"}, nil); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d < 40*time.Millisecond || d > 2*time.Second {
		t.Errorf("waited %s; want about the 50 ms maximum", d)
	}
}

func TestCancelStopsRetries(t *testing.T) {
	var calls atomic.Int32
	c := newClient(t, statusSeq(&calls, nil, 503, 503, 503),
		client.WithRetry(client.RetryPolicy{MaxAttempts: 3, MinBackoff: time.Hour, MaxBackoff: time.Hour}))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := c.Do(ctx, &client.Request{Method: http.MethodGet, Path: "/v1/x"}, nil)
	if !errors.Is(err, context.DeadlineExceeded) || client.StatusOf(err) != 503 || calls.Load() != 1 {
		t.Fatalf("%v, %d calls", err, calls.Load())
	}
}

func TestTimeoutPerTry(t *testing.T) {
	var calls atomic.Int32
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			<-r.Context().Done() // the first try hangs
			return
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	err := c.Do(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/x", Timeout: 50 * time.Millisecond}, nil)
	if err != nil || calls.Load() != 2 {
		t.Fatalf("%v, %d calls", err, calls.Load())
	}
}

func TestBytesAndTheResponseCap(t *testing.T) {
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", 100))
	}))
	b, err := c.Bytes(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/f", MaxResponse: 100})
	if err != nil || len(b) != 100 {
		t.Fatalf("%d %v", len(b), err)
	}
	if _, err := c.Bytes(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/f", MaxResponse: 99}); err == nil {
		t.Fatal("an answer over the cap: no error")
	}
}

func sse(w http.ResponseWriter, lines ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, l := range lines {
		_, _ = io.WriteString(w, l+"\n")
		w.(http.Flusher).Flush()
	}
}

func TestStream(t *testing.T) {
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sse(w, ": ping", "event: stdout", `data: {"data":"a"}`, "", "event: stdout", `data: {"data":`, `data: "b"}`, "",
			"id: 7", `data: {"plain":true}`, "", "event: exit", `data: {"exit_code":0}`)
	}))
	var got []string
	err := c.Stream(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/s"}, time.Second,
		func(ev client.Event) error {
			got = append(got, ev.Name+" "+string(ev.Data))
			return nil
		})
	want := []string{`stdout {"data":"a"}`, "stdout {\"data\":\n\"b\"}", `message {"plain":true}`, `exit {"exit_code":0}`}
	if err != nil || strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("%v\n got %q\nwant %q", err, got, want)
	}
}

func TestStreamErrorEventAndEOF(t *testing.T) {
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sse(w, "event: stdout", `data: {"data":"a"}`, "", "event: error", `data: {"error":"node lost","code":"node_unreachable"}`, "")
	}))
	n := 0
	err := c.Stream(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/s"}, 0,
		func(client.Event) error { n++; return nil })
	if !client.IsCode(err, "node_unreachable") || client.StatusOf(err) != 502 || n != 1 {
		t.Fatalf("%v after %d events", err, n)
	}
	err = c.Stream(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/s"}, 0,
		func(client.Event) error { return io.EOF })
	if err != nil {
		t.Fatalf("io.EOF from fn: %v", err)
	}
}

func TestStreamIdleAndNoRetryOnceStarted(t *testing.T) {
	var calls atomic.Int32
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		sse(w, "event: stdout", `data: {"data":"a"}`, "")
		<-r.Context().Done()
	}))
	err := c.Stream(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/s"}, 100*time.Millisecond,
		func(client.Event) error { return nil })
	if !errors.Is(err, client.ErrStreamIdle) || calls.Load() != 1 {
		t.Fatalf("%v, %d calls", err, calls.Load())
	}
}

func TestStreamWantsAnEventStream(t *testing.T) {
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `{}`) }),
		client.WithRetry(client.NoRetry))
	err := c.Stream(context.Background(), &client.Request{Method: http.MethodPost, Path: "/v1/s"}, 0,
		func(client.Event) error { return nil })
	if client.StatusOf(err) != 502 {
		t.Fatal(err)
	}
}

func TestList(t *testing.T) {
	var queries []string
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		switch r.URL.Query().Get("cursor") {
		case "":
			_, _ = io.WriteString(w, `{"items":[1,2],"next_cursor":"c2"}`)
		case "c2":
			_, _ = io.WriteString(w, `{"items":[3,4],"next_cursor":"c3"}`)
		default:
			_, _ = io.WriteString(w, `{"items":[5],"next_cursor":null}`)
		}
	}))
	r := client.Request{Method: http.MethodGet, Path: "/v1/l", Query: map[string][]string{"label": {"a:b"}}}
	all, err := client.List[int](context.Background(), c, r, 0)
	if err != nil || fmt.Sprint(all) != "[1 2 3 4 5]" {
		t.Fatalf("%v %v", all, err)
	}
	if queries[0] != "label=a%3Ab&limit=500" || queries[1] != "cursor=c2&label=a%3Ab&limit=500" {
		t.Errorf("queries %q", queries)
	}
	queries = nil
	some, err := client.List[int](context.Background(), c, r, 3)
	if err != nil || fmt.Sprint(some) != "[1 2 3]" || len(queries) != 2 || queries[1] != "cursor=c2&label=a%3Ab&limit=1" {
		t.Fatalf("%v %v %q", some, err, queries)
	}
	if len(r.Query["cursor"]) != 0 {
		t.Error("List changed the caller's query")
	}
}

func TestHeadersIdempotencyKeyAndResponse(t *testing.T) {
	var calls atomic.Int32
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" || r.Header.Get("X-Extra") != "1" ||
			r.Header.Get("Idempotency-Key") != "task:t-81:1" {
			http.Error(w, "bad request headers", http.StatusTeapot)
			return
		}
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway) // the first try's answer is lost
			return
		}
		w.Header().Set("Idempotent-Replayed", "true")
		_, _ = io.WriteString(w, `{"id":"x"}`)
	}))
	var out struct{ ID string }
	// A POST with a key is retried on 502, as a PUT is; the extra headers
	// never replace the client's own Authorization.
	res, err := c.DoResponse(context.Background(), &client.Request{Method: http.MethodPost, Path: "/v1/x",
		IdempotencyKey: "task:t-81:1", Header: http.Header{"X-Extra": {"1"}, "Authorization": {"Bearer other"}}}, &out)
	if err != nil || calls.Load() != 2 || out.ID != "x" || res.Status != 200 || !res.Replayed() {
		t.Fatalf("%+v %+v %v after %d calls", res, out, err, calls.Load())
	}
	var none *client.Response
	if none.Replayed() {
		t.Error("a nil response is not replayed")
	}
	// Without a key, no Idempotency-Key is sent.
	calls.Store(0)
	err = c.Do(context.Background(), &client.Request{Method: http.MethodPost, Path: "/v1/x",
		Header: http.Header{"X-Extra": {"1"}}}, nil)
	if client.StatusOf(err) != http.StatusTeapot || calls.Load() != 0 {
		t.Fatalf("%v after %d calls", err, calls.Load())
	}
}

func TestStreamEventIDs(t *testing.T) {
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sse(w, "id: 7", "event: a", "data: {}", "", "event: b", "data: {}", "", "id: 9", "data: {}", "", "id: x\x00y", "data: {}", "")
	}))
	var ids []string
	err := c.Stream(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/s"}, 0,
		func(ev client.Event) error { ids = append(ids, ev.ID); return nil })
	if err != nil || strings.Join(ids, ",") != "7,7,9,9" {
		t.Fatalf("%v %q", err, ids)
	}
}

func TestFollowResumes(t *testing.T) {
	var conns []string
	var mu sync.Mutex
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		conns = append(conns, r.Header.Get("Last-Event-ID"))
		n := len(conns)
		mu.Unlock()
		switch n {
		case 1:
			sse(w, "id: 1", "data: {}", "", "id: 2", "data: {}", "") // then the server ends the stream
		case 2:
			w.WriteHeader(http.StatusServiceUnavailable) // a deploy
		case 3:
			sse(w, "id: 3", "data: {}", "")
			panic(http.ErrAbortHandler) // the connection drops
		default:
			sse(w, "id: 4", "data: {}", "")
		}
	}), client.WithRetry(client.RetryPolicy{MaxAttempts: 1, MinBackoff: time.Millisecond, MaxBackoff: time.Millisecond}))
	last := ""
	var got []string
	err := c.Follow(context.Background(), func() *client.Request {
		r := &client.Request{Method: http.MethodGet, Path: "/v1/sandboxes/events"}
		if last != "" {
			r.Header = http.Header{client.HeaderLastEventID: {last}}
		}
		return r
	}, time.Second, func(ev client.Event) error {
		last = ev.ID
		got = append(got, ev.ID)
		if ev.ID == "4" {
			return io.EOF
		}
		return nil
	})
	if err != nil || strings.Join(got, ",") != "1,2,3,4" || strings.Join(conns, ",") != ",2,2,3" {
		t.Fatalf("%v: events %q, Last-Event-ID per connection %q", err, got, conns)
	}
}

func TestFollowStops(t *testing.T) {
	// A refusal that will not pass ends it at once.
	var calls atomic.Int32
	c := newClient(t, statusSeq(&calls, []string{"cursor_expired"}, 410), client.WithRetry(client.NoRetry))
	open := func() *client.Request { return &client.Request{Method: http.MethodGet, Path: "/v1/sandboxes/events"} }
	err := c.Follow(context.Background(), open, 0, func(client.Event) error { return nil })
	if !client.IsCode(err, "cursor_expired") || calls.Load() != 1 {
		t.Fatalf("%v after %d calls", err, calls.Load())
	}
	// Failed connections, however many in a row, only delay it (an API
	// deploy): statusSeq answers {} after them, which is not a stream either,
	// so that last refusal ends it.
	calls.Store(0)
	c = newClient(t, statusSeq(&calls, nil, 502, 502, 502, 502, 502, 502, 502, 502),
		client.WithRetry(client.RetryPolicy{MaxAttempts: 1, MinBackoff: time.Millisecond, MaxBackoff: time.Millisecond}))
	err = c.Follow(context.Background(), open, 0, func(client.Event) error { return nil })
	if client.StatusOf(err) != 502 || calls.Load() != 9 {
		t.Fatalf("%v after %d calls; want 9: eight 502s, then the answer that is not a stream", err, calls.Load())
	}
	// fn's error is returned as it is.
	boom := errors.New("boom")
	c = newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { sse(w, "data: {}", "") }))
	if err := c.Follow(context.Background(), open, 0, func(client.Event) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("%v", err)
	}
	// A cancelled context ends it between connections.
	ctx, cancel := context.WithCancel(context.Background())
	n := 0
	err = c.Follow(ctx, open, 0, func(client.Event) error {
		if n++; n == 2 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
}

func TestStreamCutMidLineDropsTheEvent(t *testing.T) {
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sse(w, "event: stdout", `data: {"data":"a"}`, "")
		_, _ = io.WriteString(w, "event: exit\ndata: {\"exit_co") // the connection is cut here
	}))
	var got []string
	err := c.Stream(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/s"}, 0,
		func(ev client.Event) error { got = append(got, ev.Name+" "+string(ev.Data)); return nil })
	if err == nil || strings.Join(got, "|") != `stdout {"data":"a"}` {
		t.Fatalf("%v, events %q; want an error and only the complete event", err, got)
	}
}

func TestFollowReconnectsAfterAStreamCutMidEvent(t *testing.T) {
	var calls atomic.Int32
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			sse(w, "id: 1", `data: {"n":1}`, "")
			_, _ = io.WriteString(w, "id: 2\ndata: {\"n\"") // cut mid-event
			return
		}
		sse(w, "id: 2", `data: {"n":2}`, "")
	}))
	var got []string
	err := c.Follow(context.Background(), func() *client.Request {
		return &client.Request{Method: http.MethodGet, Path: "/v1/sandboxes/events"}
	}, time.Second, func(ev client.Event) error {
		var v struct{ N int }
		if err := json.Unmarshal(ev.Data, &v); err != nil {
			return err
		}
		got = append(got, ev.ID)
		if v.N == 2 {
			return io.EOF
		}
		return nil
	})
	if err != nil || strings.Join(got, ",") != "1,2" || calls.Load() != 2 {
		t.Fatalf("%v: events %q after %d connections", err, got, calls.Load())
	}
}

func TestStreamIdleCoversTheWaitForTheHeaders(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select { // the API stalls before it answers
		case <-r.Context().Done():
		case <-release:
		}
	}), client.WithRetry(client.NoRetry))
	t.Cleanup(func() { close(release) }) // runs before the server closes
	done := make(chan error, 1)
	go func() {
		done <- c.Stream(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/s"},
			100*time.Millisecond, func(client.Event) error { return nil })
	}()
	select {
	case err := <-done:
		if !errors.Is(err, client.ErrStreamIdle) || calls.Load() != 1 {
			t.Fatalf("%v after %d calls", err, calls.Load())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Stream waited for the headers past its idle limit")
	}
}

func TestFollowWaitsAfterAConnectionThatDeliveredEvents(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
	}{
		{"an event, then the end", []string{"data: {}", ""}},
		{"an event, then an error event", []string{"data: {}", "", "event: error", `data: {"error":"node lost"}`, ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				sse(w, tc.lines...)
			}), client.WithRetry(client.RetryPolicy{MaxAttempts: 1, MinBackoff: 50 * time.Millisecond, MaxBackoff: time.Second}))
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			err := c.Follow(ctx, func() *client.Request { return &client.Request{Method: http.MethodGet, Path: "/v1/s"} },
				0, func(client.Event) error { return nil })
			if !errors.Is(err, context.DeadlineExceeded) || calls.Load() > 8 {
				t.Fatalf("%v after %d connections in 300 ms; want at most 8 (50 ms apart at least)", err, calls.Load())
			}
		})
	}
}

func TestAPostWhoseAnswerIsLostIsNotSentAgain(t *testing.T) {
	cases := []struct {
		name      string
		method    string
		key       string
		wantCalls int32
	}{
		{"post without a key", http.MethodPost, "", 1},
		{"post with a key", http.MethodPost, "k-1", 3},
		{"get", http.MethodGet, "", 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				_, _ = io.ReadAll(r.Body)   // the request arrived whole...
				panic(http.ErrAbortHandler) // ...and its answer is lost
			}))
			err := c.Do(context.Background(), &client.Request{Method: tc.method, Path: "/v1/x", JSON: map[string]int{"a": 1},
				IdempotencyKey: tc.key}, nil)
			if err == nil || calls.Load() != tc.wantCalls {
				t.Fatalf("%v after %d requests; want %d", err, calls.Load(), tc.wantCalls)
			}
		})
	}
}

func TestAStreamThatStartedIsNotReopenedWhateverItsAccept(t *testing.T) {
	var calls atomic.Int32
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		sse(w, "data: {}", "")
		panic(http.ErrAbortHandler) // the connection drops after an event
	}))
	n := 0
	err := c.Stream(context.Background(), &client.Request{Method: http.MethodGet, Path: "/v1/s",
		Accept: "text/event-stream, application/json;q=0.5"}, 0, func(client.Event) error { n++; return nil })
	if err == nil || calls.Load() != 1 || n != 1 {
		t.Fatalf("%v after %d calls and %d events; want one call, one event", err, calls.Load(), n)
	}
}

func TestListStopsOnARepeatedCursor(t *testing.T) {
	var calls atomic.Int32
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) > 10 {
			_, _ = io.WriteString(w, `{"items":[],"next_cursor":null}`) // a guard for the test itself
			return
		}
		if r.URL.Query().Get("cursor") == "" {
			_, _ = io.WriteString(w, `{"items":[1],"next_cursor":"a"}`)
			return
		}
		_, _ = io.WriteString(w, `{"items":[2],"next_cursor":"a"}`) // the same cursor again
	}))
	_, err := client.List[int](context.Background(), c, client.Request{Method: http.MethodGet, Path: "/v1/l"}, 0)
	if err == nil || calls.Load() != 2 {
		t.Fatalf("%v after %d calls; want an error after 2", err, calls.Load())
	}
}

func TestWhatCountsAsNeverSent(t *testing.T) {
	// A proxy that refuses the connection: the request reached no server, so
	// even a POST is tried again.
	proxy := httptest.NewServer(http.NotFoundHandler())
	proxyURL, _ := url.Parse(proxy.URL)
	proxy.Close()
	var tries atomic.Int32
	c, err := client.New(client.WithBaseURL("http://api.example"), client.WithRetry(fastRetry),
		client.WithHTTPClient(&http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}),
		client.WithTokenSource(client.TokenFunc(func(context.Context) (string, error) { tries.Add(1); return "tok", nil })))
	if err != nil {
		t.Fatal(err)
	}
	err = c.Do(context.Background(), &client.Request{Method: http.MethodPost, Path: "/v1/x"}, nil)
	if err == nil || tries.Load() != 3 {
		t.Fatalf("a refused proxy: %v after %d tries; want 3", err, tries.Load())
	}

	// A name that does not resolve on a redirect's next hop: the first server
	// had the POST, so it is not sent again.
	var calls atomic.Int32
	c = newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Redirect(w, r, "http://no-such-host.invalid/v1/x", http.StatusTemporaryRedirect)
	}))
	err = c.Do(context.Background(), &client.Request{Method: http.MethodPost, Path: "/v1/x", JSON: map[string]int{}}, nil)
	if err == nil || calls.Load() != 1 {
		t.Fatalf("a redirect to a name that does not resolve: %v after %d calls; want 1", err, calls.Load())
	}
}

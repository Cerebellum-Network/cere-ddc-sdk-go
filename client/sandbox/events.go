package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
)

// Event types.
const (
	// EventSandboxRunning: the sandbox started (once).
	EventSandboxRunning = "sandbox.running"
	// EventSandboxExited: the sandbox ended, however (once). A sandbox that
	// never ran gets only this one, with status failed.
	EventSandboxExited = "sandbox.exited"
	// EventExecExited: a detached exec ended.
	EventExecExited = "exec.exited"
)

// FromOldest, as EventQuery.After, reads from the oldest event kept.
const FromOldest = "0"

// EventsKept is how long the API keeps events, and so how long a cursor
// stays valid; an older one answers CodeCursorExpired.
const EventsKept = 72 * time.Hour

// Event is a change in the life of a sandbox of the bucket, as the event feed
// and webhooks carry it. ID, Type, At, SandboxID, Bucket and Labels are set
// on every event; the others by type:
//
//   - sandbox.running: Status (running), StartedAt, ReadyAt, StartupMS, Image;
//   - sandbox.exited: Status (terminated or failed), ExitReason, ExitKind,
//     ExitCode, EndedAt, Retained, NotRetained, RetainedUntil, Usage (the
//     final usage);
//   - exec.exited: ExecID, Status (exited), ExitCode (nil when the sandbox
//     ended first, as Error says), TimedOut, DurationMS, StdoutBytes,
//     StderrBytes, IdempotencyKey.
//
// Delivery is at least once: dedupe on ID. IDs are decimal and grow with
// every event: one with a larger ID happened later.
type Event struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	At        time.Time         `json:"at"`
	SandboxID string            `json:"sandbox_id"`
	Bucket    uint64            `json:"bucket"`
	Labels    map[string]string `json:"labels"`
	Status    string            `json:"status"`

	StartedAt *time.Time `json:"started_at"`
	ReadyAt   *time.Time `json:"ready_at"`
	StartupMS int64      `json:"startup_ms"`
	Image     *Image     `json:"image"`

	ExitReason    string         `json:"exit_reason"`
	ExitKind      string         `json:"exit_kind"`
	ExitCode      *int           `json:"exit_code"`
	EndedAt       *time.Time     `json:"ended_at"`
	Retained      []RetainedFile `json:"retained"`
	NotRetained   []NotRetained  `json:"not_retained"`
	RetainedUntil *time.Time     `json:"retained_until"`
	Usage         *Usage         `json:"usage"`

	ExecID         string `json:"exec_id"`
	TimedOut       bool   `json:"timed_out"`
	DurationMS     int64  `json:"duration_ms"`
	StdoutBytes    int64  `json:"stdout_bytes"`
	StderrBytes    int64  `json:"stderr_bytes"`
	IdempotencyKey string `json:"idempotency_key"`
	Error          string `json:"error"`
}

// EventQuery selects events; every field is optional.
type EventQuery struct {
	// After is the cursor to read after: the ID of the last event handled,
	// or the next cursor a poll returned. Empty: from now on; FromOldest:
	// from the oldest kept (EventsKept). A cursor older than the events kept
	// answers CodeCursorExpired: read the sandboxes you follow with Get, then
	// go on without one.
	After string
	// Types are event types (the Event constants); all when empty.
	Types []string
	// Labels must all match the sandbox's labels.
	Labels map[string]string
	// SandboxID narrows to one sandbox.
	SandboxID string
	// Limit is the most events one poll returns: 1 to 500, 0 for 100.
	// Events only.
	Limit int
	// WaitS is how long, in seconds (0 to 60), a poll waits for a first
	// event when there is none yet. Events only.
	WaitS int
}

func (q EventQuery) values() url.Values {
	v := labelQuery(q.Labels)
	if len(q.Types) > 0 {
		v.Set("types", strings.Join(q.Types, ","))
	}
	if q.SandboxID != "" {
		v.Set("sandbox_id", q.SandboxID)
	}
	return v
}

func (q EventQuery) validate() error {
	if q.Limit < 0 || q.Limit > client.MaxPageSize {
		return fmt.Errorf("sandbox: an events limit is 1 to %d (0 for 100)", client.MaxPageSize)
	}
	if q.WaitS < 0 || q.WaitS > 60 {
		return errors.New("sandbox: an events wait is 0 to 60 seconds")
	}
	return nil
}

// Events polls the bucket's events after q.After, oldest first, and returns
// them with the cursor to poll from next: the ID of the last event returned
// or, when there is none, where the next poll should start (keep it and pass
// it as After). With WaitS, the call waits up to that long for a first
// event.
func (s *Service) Events(ctx context.Context, q EventQuery) ([]Event, string, error) {
	if err := q.validate(); err != nil {
		return nil, "", err
	}
	v := q.values()
	if q.After != "" {
		v.Set("after", q.After)
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.WaitS > 0 {
		v.Set("wait_s", strconv.Itoa(q.WaitS))
	}
	var out struct {
		Events []Event `json:"events"`
		Next   string  `json:"next"`
	}
	err := s.c.Do(ctx, &client.Request{Method: http.MethodGet, Path: "/v1/sandboxes/events", Query: v,
		Timeout: client.DefaultTimeout + time.Duration(q.WaitS)*time.Second}, &out)
	if err != nil {
		return nil, "", err
	}
	return out.Events, out.Next, nil
}

// StreamEvents follows the bucket's events after q.After as they happen and
// calls fn with each, oldest first, until fn returns an error (io.EOF ends it
// without one), the API refuses the stream (CodeCursorExpired, for one) or
// ctx ends. q.Limit and q.WaitS do not apply.
//
// The API ends a stream now and then, and a connection may drop: StreamEvents
// reconnects with the last event's ID (Last-Event-ID) and misses nothing. It
// keeps reconnecting, with backoff (see client.Client.Follow), for as long
// as ctx lives: an outage of the API, a deploy say, delays it but does not
// end it. Bound it with ctx.
// With an empty After it first takes the feed's current position, so that
// "from now on" holds across reconnects too. An event delivered twice is
// passed to fn once. A caller that must not miss events across its own
// restarts keeps the ID of the last event it handled and passes it as After.
func (s *Service) StreamEvents(ctx context.Context, q EventQuery, fn func(Event) error) error {
	last := q.After
	if last == "" {
		evs, next, err := s.Events(ctx, EventQuery{Types: q.Types, Labels: q.Labels, SandboxID: q.SandboxID})
		if err != nil {
			return err
		}
		for _, ev := range evs {
			if err := fn(ev); err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
		}
		last = next
	}
	base := q.values()
	seen := "" // the ID of the last event received: what a reconnect sends as Last-Event-ID
	return s.c.Follow(ctx, func() *client.Request {
		r := &client.Request{Method: http.MethodGet, Path: "/v1/sandboxes/events", Query: base}
		if last != "" {
			v := url.Values{}
			for k, vs := range base {
				v[k] = vs
			}
			v.Set("after", last)
			r.Query = v
		}
		if seen != "" {
			r.Header = http.Header{client.HeaderLastEventID: {seen}}
		}
		return r
	}, streamIdle, func(raw client.Event) error {
		var ev Event
		if err := json.Unmarshal(raw.Data, &ev); err != nil {
			return fmt.Errorf("sandbox: an event (id %q): %w", raw.ID, err)
		}
		if ev.ID == "" {
			ev.ID = raw.ID
		}
		if ev.ID == "" {
			return nil // not an event of the feed
		}
		if !after(ev.ID, last) {
			return nil // delivered before
		}
		if err := fn(ev); err != nil {
			return err
		}
		last, seen = ev.ID, ev.ID
		return nil
	})
}

// after reports whether the event id comes after the cursor: always, unless
// both are decimal and id is not larger.
func after(id, cursor string) bool {
	i, err1 := strconv.ParseUint(id, 10, 64)
	c, err2 := strconv.ParseUint(cursor, 10, 64)
	return err1 != nil || err2 != nil || i > c
}

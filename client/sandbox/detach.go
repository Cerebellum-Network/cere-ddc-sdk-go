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
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
)

// Statuses of a detached exec.
const (
	ExecStatusRunning = "running"
	ExecStatusExited  = "exited"
)

// Limits of detached execs.
const (
	// MaxDetachedExecs is how many detached execs may run at once in a
	// sandbox; one more answers client.CodeTooManyRequests.
	MaxDetachedExecs = 16
	// MaxExecOutputKept is how much of each stream of a detached exec its
	// node keeps: the last 4 MiB (the oldest exec's output goes first when a
	// sandbox's execs pass 64 MiB in all).
	MaxExecOutputKept = 4 << 20
	// MaxExecOutputRead is how much of each stream one Output call returns.
	MaxExecOutputRead = 1 << 20
)

// Exec is a detached exec as the API last reported it, and the handle to
// read its output and wait for it. Refresh reads it again.
//
// A detached exec runs in the sandbox whoever is connected: its caller may
// go away and come back (Sandbox.GetExec) to read the output from the byte
// offsets it got to. The output and status are kept for the sandbox's life
// and 10 minutes after it ends (then CodeExecGone).
type Exec struct {
	ExecID    string     `json:"exec_id"`
	Status    string     `json:"status"`
	Command   []string   `json:"command"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
	// DurationMS is how long the command ran; 0 while it runs.
	DurationMS int64 `json:"duration_ms"`
	// ExitCode is the command's exit code: nil while it runs, and when the
	// sandbox ended first (Error says so). A command still running at its
	// timeout is reported exited with 124 and TimedOut.
	ExitCode *int `json:"exit_code"`
	TimedOut bool `json:"timed_out"`
	// Error is set when the sandbox ended while the command ran.
	Error string `json:"error"`
	// StdoutBytes and StderrBytes count what each stream wrote.
	StdoutBytes int64 `json:"stdout_bytes"`
	StderrBytes int64 `json:"stderr_bytes"`
	// StdoutTruncatedBefore and StderrTruncatedBefore are where the output
	// the node keeps begins (MaxExecOutputKept): what came before is gone.
	StdoutTruncatedBefore int64 `json:"stdout_truncated_before"`
	StderrTruncatedBefore int64 `json:"stderr_truncated_before"`
	// OutputInterrupted is set when the node stopped capturing the output
	// (an upgrade of the node while it ran); the status and exit code are
	// still followed.
	OutputInterrupted bool `json:"output_interrupted"`
	// IdempotencyKey is the key it was started with.
	IdempotencyKey string `json:"idempotency_key"`
	// SandboxID is the sandbox it runs in.
	SandboxID string `json:"-"`

	svc      *Service
	replayed bool
}

// Running reports whether the exec was running when last read.
func (e *Exec) Running() bool { return e.Status == ExecStatusRunning }

// Replayed reports whether ExecDetach found the exec an earlier call with the
// same IdempotencyKey had started (the API answered Idempotent-Replayed),
// rather than starting one.
func (e *Exec) Replayed() bool { return e.replayed }

// api is the client the exec's calls go through.
func (e *Exec) api() (*client.Client, error) {
	if e == nil || e.svc == nil || e.svc.c == nil {
		return nil, ErrNoService
	}
	return e.svc.c, nil
}

func (e *Exec) path(rest string) string {
	return "/v1/sandboxes/" + url.PathEscape(e.SandboxID) + "/exec/" + url.PathEscape(e.ExecID) + rest
}

func (e *Exec) update(from *Exec) {
	svc, id, replayed := e.svc, e.SandboxID, e.replayed
	*e = *from
	e.svc, e.SandboxID, e.replayed = svc, id, replayed
}

// ExecDetach starts a command in the sandbox and returns at once, with the
// exec running (ExecID, Status, StartedAt; Refresh reads the rest). The
// command runs until it ends, its Timeout (zero: DefaultExecTimeout) or the
// sandbox ends, whoever is connected; an exec.exited event announces its end
// (Service.Events, webhooks). A running detached exec holds off the
// sandbox's idle timeout. At most MaxDetachedExecs run at once.
//
// With req.IdempotencyKey, a repeat of the call on the same sandbox starts
// nothing and answers the exec the first one started, as it is now, with
// Replayed set (the same key with another body answers
// CodeIdempotencyConflict); the call is then also retried after a lost
// answer. Keys are kept for the sandbox's life.
func (sb *Sandbox) ExecDetach(ctx context.Context, req ExecRequest) (*Exec, error) {
	body, err := req.body(true)
	if err != nil {
		return nil, err
	}
	c, err := sb.api()
	if err != nil {
		return nil, err
	}
	ex := &Exec{SandboxID: sb.ID, svc: sb.svc}
	res, err := c.DoResponse(ctx, &client.Request{
		Method: http.MethodPost, Path: sb.path("/exec"), JSON: body, IdempotencyKey: req.IdempotencyKey,
	}, ex)
	if err != nil {
		return nil, err
	}
	if ex.ExecID == "" {
		return nil, errors.New("sandbox: exec detach: the API answered no exec")
	}
	ex.replayed = res.Replayed()
	return ex, nil
}

// Execs lists the sandbox's detached execs, newest first.
func (sb *Sandbox) Execs(ctx context.Context) ([]*Exec, error) {
	c, err := sb.api()
	if err != nil {
		return nil, err
	}
	var page client.Page[*Exec]
	if err := c.Do(ctx, &client.Request{Method: http.MethodGet, Path: sb.path("/exec")}, &page); err != nil {
		return nil, err
	}
	for _, ex := range page.Items {
		ex.SandboxID, ex.svc = sb.ID, sb.svc
	}
	return page.Items, nil
}

// GetExec reads one of the sandbox's detached execs by its ID; an unknown ID
// answers CodeExecNotFound.
func (sb *Sandbox) GetExec(ctx context.Context, execID string) (*Exec, error) {
	if execID == "" {
		return nil, errors.New("sandbox: GetExec needs an exec ID")
	}
	if _, err := sb.api(); err != nil {
		return nil, err
	}
	ex := &Exec{ExecID: execID, SandboxID: sb.ID, svc: sb.svc}
	if err := ex.Refresh(ctx); err != nil {
		return nil, err
	}
	return ex, nil
}

// Refresh reads the exec again.
func (e *Exec) Refresh(ctx context.Context) error {
	c, err := e.api()
	if err != nil {
		return err
	}
	var out Exec
	if err := c.Do(ctx, &client.Request{Method: http.MethodGet, Path: e.path("")}, &out); err != nil {
		return err
	}
	e.update(&out)
	return nil
}

// Wait polls (every second) until the exec has exited and returns its exit
// code: -1 when it has none, because the sandbox ended while it ran (Error
// says so).
func (e *Exec) Wait(ctx context.Context) (int, error) {
	for {
		if err := e.Refresh(ctx); err != nil && client.StatusOf(err) < 500 {
			return 0, err
		}
		if e.Status == ExecStatusExited {
			if e.ExitCode != nil {
				return *e.ExitCode, nil
			}
			return -1, nil
		}
		if err := pause(ctx, time.Second); err != nil {
			return 0, err
		}
	}
}

// OutputQuery is where to read a detached exec's output from: byte offsets
// of each stream from the exec's start (0, 0 for the start).
type OutputQuery struct {
	StdoutOffset int64
	StderrOffset int64
}

func (q OutputQuery) values() url.Values {
	return url.Values{
		"stdout_offset": {strconv.FormatInt(q.StdoutOffset, 10)},
		"stderr_offset": {strconv.FormatInt(q.StderrOffset, 10)},
	}
}

// ExecOutput is what a detached exec wrote from the offsets asked up to now
// (at most MaxExecOutputRead of each stream).
type ExecOutput struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	// StdoutOffset and StderrOffset are where this output ends: read on from
	// there (Next).
	StdoutOffset int64 `json:"stdout_offset"`
	StderrOffset int64 `json:"stderr_offset"`
	// StdoutTruncatedBefore and StderrTruncatedBefore are where the output
	// kept begins: an offset asked below it read from there instead.
	StdoutTruncatedBefore int64 `json:"stdout_truncated_before"`
	StderrTruncatedBefore int64 `json:"stderr_truncated_before"`
	// Status and ExitCode are the exec's when it was read.
	Status   string `json:"status"`
	ExitCode *int   `json:"exit_code"`
}

// Next is the query that reads on from where o ends.
func (o *ExecOutput) Next() OutputQuery {
	return OutputQuery{StdoutOffset: o.StdoutOffset, StderrOffset: o.StderrOffset}
}

// Output reads what the exec wrote from the offsets in from up to now, at
// most MaxExecOutputRead of each stream: call again from o.Next() for more.
// It returns at once, whether the exec runs or has ended.
func (e *Exec) Output(ctx context.Context, from OutputQuery) (*ExecOutput, error) {
	c, err := e.api()
	if err != nil {
		return nil, err
	}
	var out ExecOutput
	if err := c.Do(ctx, &client.Request{Method: http.MethodGet, Path: e.path("/output"),
		Query: from.values()}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ExecExit is how a detached exec ended.
type ExecExit struct {
	// ExitCode is the command's exit code; nil when the sandbox ended while
	// it ran (Error says so).
	ExitCode   *int   `json:"exit_code"`
	DurationMS int64  `json:"duration_ms"`
	TimedOut   bool   `json:"timed_out"`
	Error      string `json:"error"`
}

// OutputEvent is one event of a detached exec's output stream.
type OutputEvent struct {
	// Name is "stdout" or "stderr" (Data is a chunk of that stream),
	// "truncated" (the output of Stream before TruncatedBefore is no longer
	// kept, and the stream goes on from there) or "exit" (the exec ended:
	// Exit; the last event).
	Name            string
	Data            string
	Stream          string
	TruncatedBefore int64
	Exit            *ExecExit
	// Next is where the stream goes on after this event. A caller that keeps
	// its place keeps Next, and gives it to Follow, Stream or Output to read
	// on: nothing is sent twice, and nothing skipped but output the node
	// dropped.
	Next OutputQuery
}

// Follow streams the exec's output from the offsets in from, calling fn with
// each event, as the command writes it, until the exec ends: it returns how.
// A connection that drops or ends early is opened again from the last
// offsets received, so fn sees each byte once; it is reopened, with backoff,
// for as long as ctx lives (an outage of the API delays it but does not end
// it), so bound it with ctx. fn runs on the caller's
// goroutine; its error ends the call, and io.EOF ends it early without one
// (Follow then returns a nil ExecExit).
func (e *Exec) Follow(ctx context.Context, from OutputQuery, fn func(OutputEvent) error) (*ExecExit, error) {
	c, err := e.api()
	if err != nil {
		return nil, err
	}
	at := from
	var exit *ExecExit
	err = c.Follow(ctx, func() *client.Request {
		q := at.values()
		q.Set("stream", "1")
		return &client.Request{Method: http.MethodGet, Path: e.path("/output"), Query: q}
	}, streamIdle, func(ev client.Event) error {
		out := OutputEvent{Name: ev.Name}
		switch ev.Name {
		case "stdout", "stderr":
			var chunk struct {
				Data   string `json:"data"`
				Offset int64  `json:"offset"`
			}
			if err := json.Unmarshal(ev.Data, &chunk); err != nil {
				return fmt.Errorf("sandbox: a %s event: %w", ev.Name, err)
			}
			out.Data = chunk.Data
			if ev.Name == "stdout" {
				at.StdoutOffset = chunk.Offset
			} else {
				at.StderrOffset = chunk.Offset
			}
		case "truncated":
			var t struct {
				Stream          string `json:"stream"`
				TruncatedBefore int64  `json:"truncated_before"`
			}
			if err := json.Unmarshal(ev.Data, &t); err != nil {
				return fmt.Errorf("sandbox: a truncated event: %w", err)
			}
			out.Stream, out.TruncatedBefore = t.Stream, t.TruncatedBefore
			switch t.Stream {
			case "stdout":
				at.StdoutOffset = max(at.StdoutOffset, t.TruncatedBefore)
			case "stderr":
				at.StderrOffset = max(at.StderrOffset, t.TruncatedBefore)
			}
		case "exit":
			var x ExecExit
			if err := json.Unmarshal(ev.Data, &x); err != nil {
				return fmt.Errorf("sandbox: the exit event: %w", err)
			}
			out.Exit = &x
		default:
			return nil // an event this version does not know
		}
		out.Next = at
		if err := fn(out); err != nil {
			return err
		}
		if out.Exit != nil {
			exit = out.Exit
			return io.EOF
		}
		return nil
	})
	if err != nil || exit == nil {
		return nil, err
	}
	return exit, nil
}

// Stream writes the exec's output from the offsets in from to stdout and
// stderr (either may be nil; one writer for both keeps the order) as the
// command writes it, resuming a dropped connection as Follow does, and
// returns how the exec ended. Output the node no longer keeps is skipped
// silently; use Follow to see where.
func (e *Exec) Stream(ctx context.Context, from OutputQuery, stdout, stderr io.Writer) (*ExecExit, error) {
	exit, err := e.Follow(ctx, from, func(ev OutputEvent) error {
		w := stdout
		switch ev.Name {
		case "stdout":
		case "stderr":
			w = stderr
		default:
			return nil
		}
		if w == nil || ev.Data == "" {
			return nil
		}
		_, err := io.WriteString(w, ev.Data)
		return err
	})
	if err == nil && exit == nil {
		return nil, errors.New("sandbox: the exec's output stream ended without an exit event")
	}
	return exit, err
}

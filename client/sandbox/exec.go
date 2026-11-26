package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
)

// MaxExecOutputBytes is how much of each of stdout and stderr a synchronous
// exec keeps; use ExecStream for more.
const MaxExecOutputBytes = 16 << 20

// ExecRequest is a command to run in a sandbox.
type ExecRequest struct {
	// Command is the program and its arguments. Required.
	Command []string
	// Stdin is written to the command's standard input.
	Stdin string
	// Timeout is how long to wait for the command, rounded up to whole
	// seconds: zero for DefaultExecTimeout, NoTimeout for none. After it the
	// call ends with TimedOut set; the command may still run in the sandbox.
	Timeout time.Duration
	// Workdir is the command's working directory.
	Workdir string
	// Env is added to the command's environment (not secret).
	Env map[string]string
	// IdempotencyKey (1 to 255 printable ASCII characters, for example
	// "<task id>:<attempt>") makes ExecDetach safe to repeat: the same key on
	// the same sandbox never starts a second process. ExecDetach only.
	IdempotencyKey string
}

// httpTimeout bounds a synchronous exec's HTTP call: the command's timeout
// and some slack, or none for NoTimeout.
func (r ExecRequest) httpTimeout() time.Duration {
	switch {
	case r.Timeout == NoTimeout:
		return -1
	case r.Timeout > 0:
		return r.Timeout + execSlack
	}
	return DefaultExecTimeout + execSlack
}

// body is the exec's JSON body; detach asks for a detached exec.
func (r ExecRequest) body(detach bool) (any, error) {
	if len(r.Command) == 0 {
		return nil, errors.New("sandbox: exec needs a command")
	}
	if r.Timeout < 0 && r.Timeout != NoTimeout {
		return nil, errors.New("sandbox: an exec timeout must not be negative (NoTimeout asks for none)")
	}
	if r.IdempotencyKey != "" {
		if !detach {
			return nil, errors.New("sandbox: an exec's IdempotencyKey is for ExecDetach only")
		}
		if err := checkIdempotencyKey(r.IdempotencyKey); err != nil {
			return nil, err
		}
	}
	type wire struct {
		Command  []string          `json:"command"`
		Stdin    *string           `json:"stdin"`
		TimeoutS *int64            `json:"timeout_s,omitempty"`
		Workdir  string            `json:"workdir,omitempty"`
		Env      map[string]string `json:"env,omitempty"`
		Detach   bool              `json:"detach,omitempty"`
	}
	w := wire{Command: r.Command, TimeoutS: timeoutSeconds(r.Timeout), Workdir: r.Workdir, Env: r.Env, Detach: detach}
	if r.Stdin != "" {
		stdin := r.Stdin
		w.Stdin = &stdin
	}
	return w, nil
}

// ExitStatus is how a command ended.
type ExitStatus struct {
	// ExitCode is the command's exit code; -1 when it was not waited for
	// (TimedOut or Truncated).
	ExitCode   int   `json:"exit_code"`
	DurationMS int64 `json:"duration_ms"`
	// TimedOut is set when the exec's timeout passed first: the command may
	// still be running in the sandbox (terminate it to end it).
	TimedOut bool `json:"timed_out"`
}

// ExecResult is the answer of a synchronous exec.
type ExecResult struct {
	ExitStatus
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	// Truncated is set when stdout or stderr outgrew MaxExecOutputBytes: the
	// call ended there, with what was read and ExitCode -1, and the command
	// may still be running.
	Truncated bool `json:"truncated"`
}

// Exec runs a command and waits for it, returning its output whole (each
// stream kept up to MaxExecOutputBytes). A sandbox that is not running
// answers CodeNotRunning. Use ExecStream for long output or to see it as it
// comes, and ExecDetach for a command that must outlive its caller.
func (sb *Sandbox) Exec(ctx context.Context, req ExecRequest) (*ExecResult, error) {
	body, err := req.body(false)
	if err != nil {
		return nil, err
	}
	var res ExecResult
	err = sb.svc.c.Do(ctx, &client.Request{
		Method: http.MethodPost, Path: sb.path("/exec"), JSON: body,
		Timeout: req.httpTimeout(), MaxResponse: 3 * MaxExecOutputBytes,
	}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// ExecStream runs a command and writes its output to stdout and stderr
// (either may be nil; one writer for both keeps the order) as the sandbox
// produces it, then returns how it ended. Writes happen on the caller's
// goroutine; a writer's error ends the call. The command may run for as long
// as its Timeout allows; the stream is given up only if it sends nothing, not
// even the API's keep-alive (every 15 s), for 2 minutes. A dropped stream
// cannot be reattached: for that, start the command with ExecDetach and read
// it with Exec.Stream.
func (sb *Sandbox) ExecStream(ctx context.Context, req ExecRequest, stdout, stderr io.Writer) (*ExitStatus, error) {
	body, err := req.body(false)
	if err != nil {
		return nil, err
	}
	var exit *ExitStatus
	err = sb.svc.c.Stream(ctx, &client.Request{
		Method: http.MethodPost, Path: sb.path("/exec"), Query: url.Values{"stream": {"1"}}, JSON: body,
	}, streamIdle, func(ev client.Event) error {
		switch ev.Name {
		case "stdout", "stderr":
			return writeChunk(ev, stdout, stderr)
		case "exit":
			var st ExitStatus
			if err := json.Unmarshal(ev.Data, &st); err != nil {
				return fmt.Errorf("sandbox: the exit event: %w", err)
			}
			exit = &st
			return io.EOF
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if exit == nil {
		return nil, errors.New("sandbox: the exec stream ended without an exit event")
	}
	return exit, nil
}

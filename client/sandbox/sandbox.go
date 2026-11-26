package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
)

// How long calls of the sandbox API may take, beyond the client's timeout.
const (
	// createTimeout bounds a create: the API may wait up to 3 minutes for a
	// placement over several nodes.
	createTimeout = 200 * time.Second
	// pollInterval is how often a starting sandbox is polled.
	pollInterval = 500 * time.Millisecond
	// defaultWaitRunning bounds the wait for a sandbox to run when the
	// context has no deadline.
	defaultWaitRunning = 10 * time.Minute
	// execSlack is added to an exec's own timeout before the client gives up
	// on a synchronous exec.
	execSlack = 30 * time.Second
	// streamIdle is the longest silence of an exec or log stream: the API
	// sends a keep-alive every 15 s, so a stream silent for longer has lost
	// its connection. A command may run far longer.
	streamIdle = 2 * time.Minute
)

// Service is the sandbox API of the developer platform.
type Service struct {
	c *client.Client
}

// New returns the sandbox service of c.
func New(c *client.Client) *Service { return &Service{c: c} }

// Client is the client the service calls with.
func (s *Service) Client() *client.Client { return s.c }

// Sandbox is a sandbox as the API last reported it, and the handle to act on
// it. Refresh reads it again.
//
// Exec, ExecStream, WriteFile, ReadFile, Logs, CurrentUsage and SetSecret only
// read the handle, so they may run on several goroutines at once. The calls
// that read the sandbox again (Refresh, WaitRunning, Wait, the egress calls,
// Terminate) update its fields: run them while no other call uses the same
// handle, or give each goroutine its own (Service.Get).
type Sandbox struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Bucket    uint64    `json:"bucket"`
	Node      string    `json:"node"`
	Runtime   string    `json:"runtime"`
	Image     string    `json:"image"`
	Command   []string  `json:"command"`
	Workdir   string    `json:"workdir"`
	Resources Resources `json:"resources"`
	// TimeoutS and IdleTimeoutS are in seconds.
	TimeoutS     int64 `json:"timeout_s"`
	IdleTimeoutS int64 `json:"idle_timeout_s"`
	// EnvKeys are the keys of Env; the values stay on the node.
	EnvKeys []string `json:"env_keys"`
	// Egress is the allow-list, as created or as last replaced.
	Egress Egress            `json:"egress"`
	Labels map[string]string `json:"labels"`
	// Secrets are the sandbox's own secrets, without values.
	Secrets   []SecretInfo `json:"secrets"`
	CreatedAt time.Time    `json:"created_at"`
	StartedAt *time.Time   `json:"started_at"`
	EndedAt   *time.Time   `json:"ended_at"`
	// StartupMS is how long the sandbox took to start; 0 until it has.
	StartupMS int64 `json:"startup_ms"`
	// ExitReason says why it ended: "terminated by API[: reason]", "timeout
	// (Ns)", "idle_timeout (Ns)", "oom_killed", "memory_reclaimed",
	// "disk_limit (...)", "insufficient_balance", "exited with code N",
	// "failed: ...", "lost: ...", or "token_expiry" (clusters before the
	// lifetime change of 26 Nov 2026). Empty while it runs.
	ExitReason string `json:"exit_reason"`
	// ExitCode is the main process's exit code, when it has one.
	ExitCode *int `json:"exit_code"`
	// Retained lists the files kept readable after the end (Terminate).
	Retained []string `json:"retained"`
	// Usage is the usage as last reported: live while it runs, final after.
	Usage Usage `json:"usage"`

	svc *Service
}

// Running reports whether the sandbox was running when last read.
func (sb *Sandbox) Running() bool { return sb.Status == StatusRunning }

// Ended reports whether the sandbox had ended (terminated or failed) when
// last read. An ended sandbox never changes again.
func (sb *Sandbox) Ended() bool { return sb.Status == StatusTerminated || sb.Status == StatusFailed }

func (sb *Sandbox) path(rest string) string { return "/v1/sandboxes/" + url.PathEscape(sb.ID) + rest }

func (sb *Sandbox) update(from *Sandbox) {
	svc := sb.svc
	*sb = *from
	sb.svc = svc
}

// StartError is returned by Create and WaitRunning when a sandbox ended
// instead of starting.
type StartError struct {
	ID, Status, ExitReason string
}

func (e *StartError) Error() string {
	return fmt.Sprintf("sandbox: %s did not start: %s: %s", e.ID, e.Status, e.ExitReason)
}

// CreateOption changes how Create behaves.
type CreateOption func(*createOptions)

type createOptions struct{ noWait bool }

// NoWait makes Create return as soon as the API has accepted the sandbox
// (status starting), without waiting for it to run. WaitRunning waits later.
func NoWait() CreateOption { return func(o *createOptions) { o.noWait = true } }

// Create creates a sandbox and waits until it runs (see NoWait). A sandbox
// that fails to start is returned with a *StartError.
//
// The create's token must be valid when the node checks it; the sandbox then
// runs until its Timeout or another end, and stops early only if its bucket's
// owner runs out of balance. Clusters before the lifetime change of 26 Nov
// 2026 also end a sandbox 60 s before its create token expires, so the create
// asks the client's token source for a token that lasts the sandbox's Timeout
// (DefaultTimeout for none) and that margin: a client.Delegation mints one,
// which is harmless where the token no longer bounds the sandbox. A static
// token is sent as it is. Later calls take any valid token for the bucket.
//
// A create that no node has room for (CodeNoCapacity) did nothing and is
// retried under the client's retry policy; a bucket at its most sandboxes
// answers CodeLimitActiveSandboxes.
func (s *Service) Create(ctx context.Context, req CreateRequest, opts ...CreateOption) (*Sandbox, error) {
	var o createOptions
	for _, opt := range opts {
		opt(&o)
	}
	if err := req.validate(); err != nil {
		return nil, err
	}
	life := req.Timeout
	if life <= 0 { // the default, or none: a cluster before the change reads both as its default
		life = DefaultTimeout
	}
	life += TokenExpiryMargin
	sb := &Sandbox{svc: s}
	err := s.c.Do(ctx, &client.Request{
		Method: http.MethodPost, Path: "/v1/sandboxes", JSON: req,
		Timeout: createTimeout, TokenLife: life, RetryOn: []string{CodeNoCapacity},
	}, sb)
	if err != nil {
		return nil, err
	}
	if o.noWait {
		return sb, nil
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultWaitRunning)
		defer cancel()
	}
	return sb, sb.WaitRunning(ctx)
}

// Get reads a sandbox of the bucket by its ID.
func (s *Service) Get(ctx context.Context, id string) (*Sandbox, error) {
	sb := &Sandbox{ID: id, svc: s}
	if err := sb.Refresh(ctx); err != nil {
		return nil, err
	}
	return sb, nil
}

// ListOptions selects sandboxes; every field is optional.
type ListOptions struct {
	// Status is one of the Status constants.
	Status string
	// Labels must all match.
	Labels map[string]string
	// Limit is the most sandboxes returned: 0 is 100, a negative value all.
	Limit int
}

// List lists the bucket's sandboxes, newest first.
func (s *Service) List(ctx context.Context, o ListOptions) ([]*Sandbox, error) {
	q := labelQuery(o.Labels)
	if o.Status != "" {
		q.Set("status", o.Status)
	}
	items, err := client.List[*Sandbox](ctx, s.c,
		client.Request{Method: http.MethodGet, Path: "/v1/sandboxes", Query: q}, limit(o.Limit))
	if err != nil {
		return nil, err
	}
	for _, sb := range items {
		sb.svc = s
	}
	return items, nil
}

// labelQuery is labels as the API's filters, label=key:value, in key order.
func labelQuery(labels map[string]string) url.Values {
	q := url.Values{}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		q.Add("label", k+":"+labels[k])
	}
	return q
}

// Refresh reads the sandbox again; Usage is live while it runs.
func (sb *Sandbox) Refresh(ctx context.Context) error {
	var out Sandbox
	if err := sb.svc.c.Do(ctx, &client.Request{Method: http.MethodGet, Path: sb.path("")}, &out); err != nil {
		return err
	}
	sb.update(&out)
	return nil
}

// WaitRunning polls until the sandbox leaves starting (and unknown). It
// returns nil once it runs, and a *StartError if it ended instead.
func (sb *Sandbox) WaitRunning(ctx context.Context) error {
	for sb.Status == StatusStarting || sb.Status == StatusUnknown || sb.Status == "" {
		if err := pause(ctx, pollInterval); err != nil {
			return fmt.Errorf("sandbox: %s still %s: %w", sb.ID, sb.Status, err)
		}
		if err := sb.Refresh(ctx); err != nil && client.StatusOf(err) < 500 {
			return err
		}
	}
	if sb.Status != StatusRunning {
		return &StartError{ID: sb.ID, Status: sb.Status, ExitReason: sb.ExitReason}
	}
	return nil
}

// Wait polls (every second) until the sandbox ends, by its command exiting, a
// timeout or Terminate, and returns its exit code (-1 when it has none).
func (sb *Sandbox) Wait(ctx context.Context) (int, error) {
	for {
		if err := sb.Refresh(ctx); err != nil && client.StatusOf(err) < 500 {
			return 0, err
		}
		if sb.Ended() {
			if sb.ExitCode != nil {
				return *sb.ExitCode, nil
			}
			return -1, nil
		}
		if err := pause(ctx, time.Second); err != nil {
			return 0, err
		}
	}
}

func pause(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// WriteFile writes a file in the sandbox, creating its parent directories. A
// file over MaxFileBytes is refused before anything is sent, with an
// *client.Error coded client.CodeTooLarge, as the API would.
func (sb *Sandbox) WriteFile(ctx context.Context, path string, data []byte) error {
	if len(data) > MaxFileBytes {
		return &client.Error{Status: http.StatusRequestEntityTooLarge, Code: client.CodeTooLarge,
			Message: fmt.Sprintf("the file is %d bytes, at most %d (16 MiB)", len(data), MaxFileBytes)}
	}
	if data == nil {
		data = []byte{}
	}
	return sb.svc.c.Do(ctx, &client.Request{
		Method: http.MethodPut, Path: sb.path("/files"), Query: url.Values{"path": {path}}, Body: data,
		Timeout: client.DefaultTimeout + time.Duration(len(data)>>20)*time.Second,
	}, nil)
}

// ReadFile reads a file of a running sandbox, or a retained file of an ended
// one for 24 hours. A missing file answers CodeFileNotFound; a file of an
// ended sandbox that was not retained, or no longer is, CodeFileGone.
func (sb *Sandbox) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return sb.svc.c.Bytes(ctx, &client.Request{
		Method: http.MethodGet, Path: sb.path("/files"), Query: url.Values{"path": {path}}, Accept: "*/*",
		Timeout: client.DefaultTimeout + 16*time.Second, MaxResponse: MaxFileBytes + 1<<10,
	})
}

// Logs streams the main process's output to stdout and stderr (either may be
// nil) until it ends. Logs are kept 10 minutes after the sandbox ends, then
// answer CodeLogsGone.
func (sb *Sandbox) Logs(ctx context.Context, stdout, stderr io.Writer) error {
	return sb.svc.c.Stream(ctx, &client.Request{Method: http.MethodGet, Path: sb.path("/logs")}, streamIdle,
		func(ev client.Event) error {
			switch ev.Name {
			case "stdout", "stderr":
				return writeChunk(ev, stdout, stderr)
			case "end":
				return io.EOF
			}
			return nil
		})
}

// writeChunk writes the data of a stdout or stderr event.
func writeChunk(ev client.Event, stdout, stderr io.Writer) error {
	var chunk struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(ev.Data, &chunk); err != nil {
		return fmt.Errorf("sandbox: a %s event: %w", ev.Name, err)
	}
	w := stdout
	if ev.Name == "stderr" {
		w = stderr
	}
	if w == nil || chunk.Data == "" {
		return nil
	}
	_, err := io.WriteString(w, chunk.Data)
	return err
}

// CurrentUsage reads what the sandbox has used so far: live while it runs.
func (sb *Sandbox) CurrentUsage(ctx context.Context) (*Usage, error) {
	var u Usage
	if err := sb.svc.c.Do(ctx, &client.Request{Method: http.MethodGet, Path: sb.path("/usage")}, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// TerminateOptions say what to keep when a sandbox ends.
type TerminateOptions struct {
	// Retain lists files (each at most 16 MiB) that stay readable with
	// ReadFile for 24 hours after the end.
	Retain []string
	// Reason (at most 120 characters) is recorded as "terminated by API:
	// <reason>".
	Reason string
}

// Terminate ends the sandbox and updates sb with its final state and usage.
// Terminating an ended sandbox does nothing.
func (sb *Sandbox) Terminate(ctx context.Context, o TerminateOptions) error {
	q := url.Values{}
	if len(o.Retain) > 0 {
		for _, p := range o.Retain {
			if strings.Contains(p, ",") {
				return fmt.Errorf("sandbox: retain %q: a path with a comma cannot be retained", p)
			}
		}
		q.Set("retain", strings.Join(o.Retain, ","))
	}
	if o.Reason != "" {
		q.Set("reason", o.Reason)
	}
	var out Sandbox
	err := sb.svc.c.Do(ctx, &client.Request{Method: http.MethodDelete, Path: sb.path(""), Query: q,
		Timeout: client.DefaultTimeout + time.Duration(16*len(o.Retain))*time.Second}, &out)
	if err != nil {
		return err
	}
	if out.ID == "" {
		return errors.New("sandbox: terminate: the API answered no sandbox")
	}
	sb.update(&out)
	return nil
}

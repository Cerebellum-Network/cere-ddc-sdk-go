package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// Sandbox statuses. Terminated and failed are final; unknown (the node has
// been silent for a while) is not.
const (
	StatusStarting   = "starting"
	StatusRunning    = "running"
	StatusTerminated = "terminated"
	StatusFailed     = "failed"
	StatusUnknown    = "unknown"
)

// Codes of the sandbox API (beside the shared ones in package client).
const (
	// CodeNoCapacity: no node has room for the sandbox's request; nothing
	// was created (503). A create is retried on it.
	CodeNoCapacity = "no_capacity"
	// CodeLimitActiveSandboxes: the bucket has its most sandboxes running (409).
	CodeLimitActiveSandboxes = "limit_active_sandboxes"
	// CodeNotRunning: the sandbox is starting or has ended (409).
	CodeNotRunning = "not_running"
	// CodeFileNotFound: a running sandbox has no file at that path (404).
	CodeFileNotFound = "file_not_found"
	// CodeFileGone: the sandbox has ended and the file was not retained, or
	// no longer is (410).
	CodeFileGone = "file_gone"
	// CodeLogsGone: the sandbox ended more than 10 minutes ago (410).
	CodeLogsGone = "logs_gone"
	// CodeSecretNotFound: the sandbox was not created with a secret of that
	// name (404).
	CodeSecretNotFound = "secret_not_found"
	// CodeUnknownCredential: an egress rule names a credential that does not
	// resolve (400).
	CodeUnknownCredential = "unknown_credential"
	// CodeCredentialNotBound: the credential is not bound to the rule's host (400).
	CodeCredentialNotBound = "credential_not_bound"
	// CodeEgressFailure: the egress proxy could not be reached or failed (502).
	CodeEgressFailure = "egress_failure"
	// CodeNodeUnreachable: the sandbox's node could not be reached (502), or
	// no node confirmed a create in time (504).
	CodeNodeUnreachable = "node_unreachable"
	// CodeRuntimeFailure: the sandbox's node failed (502).
	CodeRuntimeFailure = "runtime_failure"
	// CodeIdempotencyConflict: an Idempotency-Key used before with another
	// body (409); nothing was done.
	CodeIdempotencyConflict = "idempotency_conflict"
	// CodeExecNotFound: the sandbox has no detached exec with that ID (404).
	CodeExecNotFound = "exec_not_found"
	// CodeExecGone: the sandbox ended more than 10 minutes ago; its detached
	// execs are no longer kept (410).
	CodeExecGone = "exec_gone"
	// CodeCursorExpired: an events cursor older than the events kept (72
	// hours) (410). Read the sandboxes you follow with Get, then go on without
	// a cursor.
	CodeCursorExpired = "cursor_expired"
	// CodeLimitWebhooks: the bucket has its 10 webhooks (409).
	CodeLimitWebhooks = "limit_webhooks"
)

// Exit kinds: the class of a sandbox's exit reason, for programs
// (Sandbox.ExitKind, Event.ExitKind).
const (
	// ExitKindMainExited: the main process exited, with any code.
	ExitKindMainExited = "main_exited"
	// ExitKindTimeout: the sandbox reached its timeout.
	ExitKindTimeout = "timeout"
	// ExitKindIdleTimeout: no exec and no file call for its idle timeout.
	ExitKindIdleTimeout = "idle_timeout"
	// ExitKindOOM: the sandbox reached its own memory limit.
	ExitKindOOM = "oom"
	// ExitKindMemoryReclaimed: it used more memory than its request while its
	// node was short of memory.
	ExitKindMemoryReclaimed = "memory_reclaimed"
	// ExitKindDiskLimit: the disk backstop, or its node's disk ran low.
	ExitKindDiskLimit = "disk_limit"
	// ExitKindInsufficientBalance: the bucket's owner ran out of balance.
	ExitKindInsufficientBalance = "insufficient_balance"
	// ExitKindTerminated: terminated through the API.
	ExitKindTerminated = "terminated"
	// ExitKindFailed: it never ran (an image pull, create or start failed).
	ExitKindFailed = "failed"
	// ExitKindLost: its runtime or host restarted.
	ExitKindLost = "lost"
)

// Limits and defaults of the sandbox API.
const (
	// DefaultTimeout is how long a sandbox runs when its create names no
	// timeout.
	DefaultTimeout = 5 * time.Minute
	// DefaultExecTimeout is how long an exec waits for its command when the
	// request names no timeout.
	DefaultExecTimeout = time.Hour
	// NoTimeout, as the Timeout of a CreateRequest or an ExecRequest, asks for
	// none: the sandbox runs until its command exits, it is terminated, its
	// idle timeout or a limit ends it, or its bucket's owner runs out of
	// balance; a command runs until it finishes or the sandbox ends.
	NoTimeout time.Duration = -1
	// MaxFileBytes is the largest file WriteFile sends and ReadFile reads, a
	// retained file included.
	MaxFileBytes = 16 << 20
	// RetainFor is how long retained files stay readable after the sandbox
	// ends, unless its Retain.TTLS says otherwise.
	RetainFor = 24 * time.Hour
	// MaxRetainedBytes is the most a sandbox keeps of its retained files in
	// all (and at most 1000 files): files are taken in the order of the paths,
	// a directory's in lexical order, and those past it are not kept.
	MaxRetainedBytes = 64 << 20
	// MaxIdempotencyKeyLen is the longest Idempotency-Key.
	MaxIdempotencyKeyLen = 255
)

// When retained files are kept (Retain.On).
const (
	// RetainOnAny keeps them however the sandbox ends: a timeout, a limit,
	// the main process exiting with any code, Terminate.
	RetainOnAny = "any"
	// RetainOnFailure keeps them on every end except the main process
	// exiting with code 0 and Terminate (whose own Retain paths are still
	// kept).
	RetainOnFailure = "failure"
)

// Retain asks, at create, for files to be kept after the sandbox ends,
// whatever ends it: an out-of-memory kill, a timeout and the main process
// exiting included. After the end the sandbox lists what was kept in
// Retained and what was not in NotRetained, and ReadFile reads each kept
// file until RetainedUntil. Retained files are not billed.
//
// A sandbox created with Retain keeps its filesystem on its node's disk, so
// that it outlives the sandbox's process; the image needs no shell or tools
// for this.
type Retain struct {
	// Paths are 1 to 32 absolute paths (at most 1024 bytes each, without "."
	// or ".." segments, not "/"). A path that is a directory when the
	// sandbox ends is kept with everything under it.
	Paths []string `json:"paths"`
	// On is RetainOnAny (the default when empty) or RetainOnFailure.
	On string `json:"on,omitempty"`
	// TTLS is how long the files are kept after the end, in seconds: 60 to
	// 604800 (7 days); 0 for the default, 86400.
	TTLS int64 `json:"ttl_s,omitempty"`
}

func (r *Retain) validate() error {
	if len(r.Paths) == 0 {
		return errors.New("sandbox: Retain needs at least one path")
	}
	for _, p := range r.Paths {
		if !strings.HasPrefix(p, "/") || strings.Trim(p, "/") == "" {
			return fmt.Errorf("sandbox: retain path %q: give an absolute path below /, for example /workspace/out.json", p)
		}
		for _, seg := range strings.Split(p, "/") {
			if seg == "." || seg == ".." {
				return fmt.Errorf("sandbox: retain path %q: no . or .. segments", p)
			}
		}
	}
	if r.On != "" && r.On != RetainOnAny && r.On != RetainOnFailure {
		return fmt.Errorf("sandbox: retain on %q: want %q or %q", r.On, RetainOnAny, RetainOnFailure)
	}
	if r.TTLS < 0 {
		return errors.New("sandbox: retain ttl_s must not be negative (0 is the default, 24 hours)")
	}
	return nil
}

// RetainedFile is a file kept after the sandbox ended, with its size in
// bytes. A retained directory's files are listed one by one.
type RetainedFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// NotRetained is a path (or a file under a directory) asked for and not
// kept, with why.
type NotRetained struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// Image is the image a sandbox runs: the reference it was created with and,
// once its node has the image, the registry digest, its size on the node,
// whether it was already there (Cached: nothing downloaded) and the time
// spent getting it (PullMS, 0 when cached). The fields other than Ref are
// zero until the node has the image.
type Image struct {
	Ref       string `json:"ref"`
	Digest    string `json:"digest"`
	SizeBytes int64  `json:"size_bytes"`
	Cached    bool   `json:"cached"`
	PullMS    int64  `json:"pull_ms"`
}

// String is the image's reference.
func (i Image) String() string { return i.Ref }

// Resources is a sandbox's request and limits. The request (CPU, Memory) is
// guaranteed, counts for placement, and is the floor billed while the sandbox
// runs; the limits (CPULimit, MemoryLimit) are the most it can use, and use
// above the request (the burst) is billed as used. A zero field is left to
// the API's default: 0.25 vCPU and 128 MiB requested, a CPU limit of the
// node's sandbox CPU (at most 8), a memory limit of 1024 MiB or the request if
// more. A sandbox that reaches its memory limit ends with oom_killed.
type Resources struct {
	// CPU is the CPU request, in vCPU (cores).
	CPU float64 `json:"cpu,omitempty"`
	// Memory is the memory request, in MiB (at least 128).
	Memory int `json:"memory,omitempty"`
	// CPULimit is the most CPU the sandbox can use, in vCPU.
	CPULimit float64 `json:"cpu_limit,omitempty"`
	// MemoryLimit is the most memory the sandbox can use, in MiB (no swap).
	MemoryLimit int `json:"memory_limit,omitempty"`
	// PIDs is the most processes (default 2048).
	PIDs int `json:"pids,omitempty"`
	// Disk is the sandbox's writable disk, in MiB (0: the default, 512).
	// There is no fixed maximum, only what a node can give. A write past it
	// fails with ENOSPC inside the sandbox, which runs on. /tmp is kept in
	// memory and counts against MemoryLimit, not Disk.
	Disk int `json:"disk,omitempty"`
}

func (r Resources) validate() error {
	if r.CPU < 0 || r.Memory < 0 || r.CPULimit < 0 || r.MemoryLimit < 0 || r.PIDs < 0 || r.Disk < 0 {
		return errors.New("sandbox: resources must not be negative")
	}
	if r.CPU > 0 && r.CPULimit > 0 && r.CPULimit < r.CPU {
		return fmt.Errorf("sandbox: cpu_limit %g is below cpu %g: the limit is the most the sandbox can use, "+
			"at least its request", r.CPULimit, r.CPU)
	}
	if r.Memory > 0 && r.MemoryLimit > 0 && r.MemoryLimit < r.Memory {
		return fmt.Errorf("sandbox: memory_limit %d is below memory %d: the limit is the most the sandbox can use, "+
			"at least its request", r.MemoryLimit, r.Memory)
	}
	return nil
}

// CreateRequest describes a sandbox to create.
type CreateRequest struct {
	// Image is a registry reference, for example "python:3.12-slim".
	// Required. An image in a private registry needs RegistryCredential.
	Image string
	// RegistryCredential names a secret of type registry (RegistrySecret)
	// to pull Image with: the sandbox's agent's (with a "cere.agent" label),
	// else the bucket's. It must be bound to the image's registry host
	// ("docker.io" for Docker Hub). It is used for the pull only, on the
	// node, and never reaches the sandbox, its environment or its egress.
	RegistryCredential string
	// Command is the main process (default ["sleep", "infinity"]).
	Command []string
	// Workdir is the working directory (default /workspace).
	Workdir string
	// Resources is the request and limits; zero fields take the defaults.
	Resources Resources
	// Timeout is how long the sandbox may run, rounded up to whole seconds:
	// zero for DefaultTimeout, NoTimeout for none.
	Timeout time.Duration
	// IdleTimeout ends the sandbox after this long with no exec and no file
	// call in flight (0: never).
	IdleTimeout time.Duration
	// Env is the main process's environment. It is not secret: it stays on
	// the node, and only its keys are reported back. Credentials go in
	// Secrets.
	Env map[string]string
	// Egress lists the hosts the sandbox may reach; nothing else is reachable.
	Egress []Allow
	// Labels are at most 32 pairs, for listing and for usage by label (for
	// example "user": "<id>"). The "cere." prefix is reserved: "cere.agent"
	// makes the sandbox a run of that agent, which may use the agent's
	// secrets.
	Labels map[string]string
	// Secrets are the sandbox's own credentials (at most 32): the egress
	// proxy holds them for this sandbox only and deletes them when it ends.
	// An Egress rule names them; the sandbox never sees a value. They are
	// header secrets; a registry credential is the bucket's or an agent's.
	Secrets []Secret
	// Retain names files to keep after the sandbox ends, whatever ends it.
	Retain *Retain
	// IdempotencyKey (1 to 255 printable ASCII characters), when set, makes
	// the create safe to repeat, for example after a crash between the
	// create and saving the sandbox's ID: a later create of the bucket with
	// the same key creates nothing and answers the sandbox the first one
	// made, as it is now (it may have started, failed or ended), with
	// Sandbox.Replayed set. The same key with another body (other than the
	// secrets' values) answers CodeIdempotencyConflict. It is sent as the
	// Idempotency-Key header, and a create with a key is also retried after
	// a lost answer.
	IdempotencyKey string
}

// MarshalJSON writes the create body of the API.
func (r CreateRequest) MarshalJSON() ([]byte, error) {
	type egress struct {
		Allow []Allow `json:"allow"`
	}
	body := struct {
		Image        string            `json:"image"`
		Command      []string          `json:"command,omitempty"`
		Runtime      string            `json:"runtime"`
		Workdir      string            `json:"workdir,omitempty"`
		Resources    *Resources        `json:"resources,omitempty"`
		TimeoutS     *int64            `json:"timeout_s,omitempty"`
		IdleTimeoutS int64             `json:"idle_timeout_s,omitempty"`
		Env          map[string]string `json:"env,omitempty"`
		Egress       *egress           `json:"egress,omitempty"`
		Labels       map[string]string `json:"labels,omitempty"`
		Secrets      []Secret          `json:"secrets,omitempty"`
		Retain       *Retain           `json:"retain,omitempty"`
		Registry     string            `json:"registry_credential,omitempty"`
	}{
		Image: r.Image, Command: r.Command, Runtime: "gvisor", Workdir: r.Workdir,
		TimeoutS: timeoutSeconds(r.Timeout), IdleTimeoutS: seconds(r.IdleTimeout),
		Env: r.Env, Labels: r.Labels, Secrets: r.Secrets, Retain: r.Retain, Registry: r.RegistryCredential,
	}
	if r.Resources != (Resources{}) {
		res := r.Resources
		body.Resources = &res
	}
	if len(r.Egress) > 0 {
		rules := make([]Allow, len(r.Egress))
		for i, a := range r.Egress {
			rules[i] = a.normalize()
		}
		body.Egress = &egress{Allow: rules}
	}
	return json.Marshal(body)
}

func (r CreateRequest) validate() error {
	if r.Image == "" {
		return errors.New(`sandbox: an image is required, for example Image: "python:3.12-slim"`)
	}
	if (r.Timeout < 0 && r.Timeout != NoTimeout) || r.IdleTimeout < 0 {
		return errors.New("sandbox: timeouts must not be negative (NoTimeout asks for none)")
	}
	if err := r.Resources.validate(); err != nil {
		return err
	}
	for _, a := range r.Egress {
		if err := a.validate(); err != nil {
			return err
		}
	}
	for _, s := range r.Secrets {
		if s.typ() != SecretTypeHeader {
			return fmt.Errorf("sandbox: secret %s: a sandbox's own secrets are header secrets; store a registry "+
				"credential with SetSecret or SetAgentSecret and name it in RegistryCredential", s.Name)
		}
		if err := s.validate(); err != nil {
			return err
		}
	}
	if r.Retain != nil {
		if err := r.Retain.validate(); err != nil {
			return err
		}
	}
	if r.IdempotencyKey != "" {
		if err := checkIdempotencyKey(r.IdempotencyKey); err != nil {
			return err
		}
	}
	return nil
}

// checkIdempotencyKey checks a key is 1 to 255 printable ASCII characters.
func checkIdempotencyKey(k string) error {
	if len(k) > MaxIdempotencyKeyLen {
		return fmt.Errorf("sandbox: an idempotency key is at most %d characters", MaxIdempotencyKeyLen)
	}
	for i := 0; i < len(k); i++ {
		if k[i] < 0x20 || k[i] > 0x7e {
			return fmt.Errorf("sandbox: idempotency key %q: printable ASCII characters only", k)
		}
	}
	return nil
}

// timeoutSeconds is a timeout as the API takes it: nil (absent) for the
// default, 0 for NoTimeout, else d rounded up to whole seconds.
func timeoutSeconds(d time.Duration) *int64 {
	switch d {
	case 0:
		return nil
	case NoTimeout:
		zero := int64(0)
		return &zero
	}
	s := seconds(d)
	return &s
}

// seconds rounds d up to whole seconds.
func seconds(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	return int64(math.Ceil(d.Seconds()))
}

// Egress is a sandbox's egress policy as the API reports it.
type Egress struct {
	Allow []Allow `json:"allow"`
}

// Usage is what a sandbox has used, measured on its node, and what it is
// billed. Measured: CPUSeconds, RAMGiBSeconds (memory.current, page cache
// included), MemoryPeakMiB, the network and disk figures, CPUPeak. Billed
// (BillingBasis "request_or_usage"): each second, the request or the use
// above it, whichever is more; ReservedCPUSeconds is the request over
// WallSeconds (the floor) and BurstCPUSeconds what was billed above it, and
// the same for memory. Billed memory is the working set, so it can be below
// measured memory. Older records carry the basis "reserved" or "measured".
type Usage struct {
	WallSeconds           float64 `json:"wall_s"`
	CPUSeconds            float64 `json:"cpu_s"`
	MemoryPeakMiB         float64 `json:"memory_peak_mib"`
	MemoryMiBSeconds      float64 `json:"memory_mib_s"`
	RAMGiBSeconds         float64 `json:"ram_gib_s"`
	GPUSeconds            float64 `json:"gpu_s"`
	NetEgressBytes        int64   `json:"net_egress_bytes"`
	NetIngressBytes       int64   `json:"net_ingress_bytes"`
	DiskPeakMiB           float64 `json:"disk_peak_mib"`
	BillingBasis          string  `json:"billing_basis"`
	ReservedCPUSeconds    float64 `json:"reserved_cpu_s"`
	ReservedRAMGiBSeconds float64 `json:"reserved_ram_gib_s"`
	BilledCPUSeconds      float64 `json:"billed_cpu_s"`
	BilledRAMGiBSeconds   float64 `json:"billed_ram_gib_s"`
	BurstCPUSeconds       float64 `json:"burst_cpu_s"`
	BurstRAMGiBSeconds    float64 `json:"burst_ram_gib_s"`
	// CPUPeak is the highest CPU use in one sample, in vCPU.
	CPUPeak float64 `json:"cpu_peak"`
	// CPUAvg is CPUSeconds / WallSeconds: the cores used on average.
	CPUAvg float64 `json:"cpu_avg"`
}

// UsageRecord is one per-minute record of a sandbox (and a final one when it
// ends), as its node wrote it. CPUUnits and RAMUnits (decimal strings) are the
// billed amounts × 1e11, what the DAC record carries.
type UsageRecord struct {
	RecordID    string    `json:"record_id"`
	SandboxID   string    `json:"sandbox_id"`
	Node        string    `json:"node"`
	Bucket      uint64    `json:"bucket"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	Final       bool      `json:"final"`
	ExitReason  string    `json:"exit_reason"`

	CPUSeconds      float64 `json:"cpu_s"`
	RAMGiBSeconds   float64 `json:"ram_gib_s"`
	MemoryPeakMiB   float64 `json:"memory_peak_mib"`
	NetEgressBytes  int64   `json:"net_egress_bytes"`
	NetIngressBytes int64   `json:"net_ingress_bytes"`
	DiskPeakMiB     float64 `json:"disk_peak_mib"`
	CPUPeak         float64 `json:"cpu_peak"`

	BillingBasis          string  `json:"billing_basis"`
	WallSeconds           float64 `json:"wall_s"`
	ReservedCPU           float64 `json:"reserved_cpu"`
	ReservedMemoryMiB     int     `json:"reserved_memory_mib"`
	ReservedCPUSeconds    float64 `json:"reserved_cpu_s"`
	ReservedRAMGiBSeconds float64 `json:"reserved_ram_gib_s"`
	BilledCPUSeconds      float64 `json:"billed_cpu_s"`
	BilledRAMGiBSeconds   float64 `json:"billed_ram_gib_s"`
	CPULimit              float64 `json:"cpu_limit"`
	MemoryLimit           int     `json:"memory_limit"`
	BurstCPUSeconds       float64 `json:"burst_cpu_s"`
	BurstRAMGiBSeconds    float64 `json:"burst_ram_gib_s"`

	CPUUnits     string     `json:"cpu_units"`
	RAMUnits     string     `json:"ram_units"`
	TCAID        int64      `json:"tca_id"`
	DACStatus    string     `json:"dac_status"`
	DACUpdatedAt *time.Time `json:"dac_updated_at"`
}

// EgressEntry is one request a sandbox made through the egress proxy, or one
// change of its rules (Decision "policy", Reason "rule added" or "rule
// removed").
type EgressEntry struct {
	ID        int64     `json:"id"`
	TS        time.Time `json:"ts"`
	Bucket    uint64    `json:"bucket"`
	SandboxID string    `json:"sandbox_id"`
	Node      string    `json:"node"`
	Host      string    `json:"host"`
	// Method is the request's method, or the rule's methods joined with ","
	// for a policy entry. A host that is not listed is refused at CONNECT.
	Method string `json:"method"`
	// Path is without the query string; empty for a CONNECT.
	Path string `json:"path"`
	// Status is the upstream's status; 0 when there was none.
	Status int `json:"status"`
	// Decision is "allowed", "denied", "error" or "policy".
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
	// Credential names the credentials the request used.
	Credential string `json:"credential"`
	BytesUp    int64  `json:"bytes_up"`
	BytesDown  int64  `json:"bytes_down"`
	DurationMS int64  `json:"duration_ms"`
}

// Decisions of the egress log.
const (
	DecisionAllowed = "allowed"
	DecisionDenied  = "denied"
	DecisionError   = "error"
	DecisionPolicy  = "policy"
)

package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
)

// Limits and defaults of the sandbox API.
const (
	// DefaultTimeout is how long a sandbox runs when its create names no
	// timeout.
	DefaultTimeout = 10 * time.Minute
	// DefaultExecTimeout is how long an exec waits for its command when the
	// request names no timeout.
	DefaultExecTimeout = time.Hour
	// NoTimeout, as the Timeout of a CreateRequest or an ExecRequest, asks for
	// none: the sandbox runs until its command exits, it is terminated, its
	// idle timeout or a limit ends it, or its bucket's owner runs out of
	// balance; a command runs until it finishes or the sandbox ends. Clusters
	// before the lifetime change of 26 Nov 2026 read it as their default (a
	// sandbox) or their maximum (an exec).
	NoTimeout time.Duration = -1
	// TokenExpiryMargin: on clusters before the lifetime change of 26 Nov
	// 2026, a sandbox ends this long before the token it was created with
	// expires (exit reason token_expiry). Since then the token only has to be
	// valid at create.
	TokenExpiryMargin = 60 * time.Second
	// MaxFileBytes is the largest file WriteFile sends and a retained file.
	MaxFileBytes = 16 << 20
	// RetainFor is how long a retained file stays readable after the sandbox ends.
	RetainFor = 24 * time.Hour
)

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
	// Disk is the most the sandbox may write to its filesystem, in MiB (0:
	// the node's default, 2048; at most 10240). A sandbox that writes more
	// ends with "disk_limit (N MiB)".
	Disk int `json:"disk_mib,omitempty"`
}

// UnmarshalJSON reads memory and memory_limit, or memory_mib and
// memory_limit_mib from APIs before 26 Nov 2026.
func (r *Resources) UnmarshalJSON(b []byte) error {
	type plain Resources
	var aux struct {
		plain
		MemoryMiB      int `json:"memory_mib"`
		MemoryLimitMiB int `json:"memory_limit_mib"`
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	*r = Resources(aux.plain)
	if r.Memory == 0 {
		r.Memory = aux.MemoryMiB
	}
	if r.MemoryLimit == 0 {
		r.MemoryLimit = aux.MemoryLimitMiB
	}
	return nil
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
	// Image is a public registry reference, for example "python:3.12-slim".
	// Required.
	Image string
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
	// An Egress rule names them; the sandbox never sees a value.
	Secrets []Secret
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
	}{
		Image: r.Image, Command: r.Command, Runtime: "gvisor", Workdir: r.Workdir,
		TimeoutS: timeoutSeconds(r.Timeout), IdleTimeoutS: seconds(r.IdleTimeout),
		Env: r.Env, Labels: r.Labels, Secrets: r.Secrets,
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
		if err := s.validate(); err != nil {
			return err
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

// UnmarshalJSON reads memory_limit, or memory_limit_mib from APIs before
// 26 Nov 2026.
func (r *UsageRecord) UnmarshalJSON(b []byte) error {
	type plain UsageRecord
	var aux struct {
		plain
		MemoryLimitMiB int `json:"memory_limit_mib"`
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	*r = UsageRecord(aux.plain)
	if r.MemoryLimit == 0 {
		r.MemoryLimit = aux.MemoryLimitMiB
	}
	return nil
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

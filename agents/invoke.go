package agents

import "encoding/json"

// BundleRef identifies an agent bundle in DDC storage by content hash.
//
// BucketID and CID are required. Cluster is optional: when absent, the runtime
// fetches the bundle from its own cluster's storage.
type BundleRef struct {
	Cluster  string `json:"cluster,omitempty"`
	BucketID string `json:"bucketId" validate:"required"`
	CID      string `json:"cid" validate:"required"`
}

// InvokeRequest is the body of POST /v1/invoke.
//
// The gateway routes the call to an agent runtime, which runs Handler from the
// bundle in a sandboxed V8 isolate. Event, Context and Labels belong to the
// caller: Event is passed to the handler as its first argument, Context is
// exposed as globalThis.context, and Labels are attached to the run's log
// records for correlation.
type InvokeRequest struct {
	Bundle  BundleRef         `json:"bundle" validate:"required"`
	Handler string            `json:"handler" validate:"required"`
	Event   json.RawMessage   `json:"event,omitempty"`
	Context json.RawMessage   `json:"context,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
	Limits  *InvokeLimits     `json:"limits,omitempty"`
	// Caller names who the run's inference calls are made for. The runtime
	// sends it as the X-Caller header on its inference calls, where the
	// inference gateway labels the request's metrics with it. Empty sends no
	// header.
	Caller string `json:"caller,omitempty"`
}

// InvokeLimits bounds one run.
//
// Exceeding TimeoutMs returns HTTP 408. Exceeding MemoryBytes ends the run with
// HTTP 200, Status "error" and Code CodeMemoryLimitExceeded, and the isolate is
// discarded. MemoryBytes 0 takes the runtime's default; a value above the
// runtime's cap is clamped to it. CPUSeconds is not enforced yet.
type InvokeLimits struct {
	TimeoutMs   int   `json:"timeoutMs"`
	CPUSeconds  int   `json:"cpuSeconds"`
	MemoryBytes int64 `json:"memoryBytes"`
}

// InvokeResponse is the body of a 200 answer to POST /v1/invoke.
//
// Status is "success" or "error". A 200 with Status "error" is a handler-level
// outcome (the handler threw, was not exported, or ran out of memory).
// Runtime-level failures return a 4xx or 5xx status instead; HTTPClient
// reports those as a *StatusError.
type InvokeResponse struct {
	Result  json.RawMessage    `json:"result,omitempty"`
	Status  string             `json:"status"`
	Error   *InvokeError       `json:"error,omitempty"`
	Metrics *InvocationMetrics `json:"metrics,omitempty"`
}

// Values of InvokeError.Code.
const (
	CodeBundleNotFound   = "BUNDLE_NOT_FOUND"
	CodeBundleInitFailed = "BUNDLE_INIT_FAILED"
	CodeHandlerNotFound  = "HANDLER_NOT_FOUND"
	CodeHandlerThrew     = "HANDLER_THREW"
	CodeTimeout          = "TIMEOUT"
	// CodeMemoryLimitExceeded: the run's V8 heap stayed over
	// InvokeLimits.MemoryBytes after a full GC. Not retryable: the same input
	// grows the same heap.
	CodeMemoryLimitExceeded = "MEMORY_LIMIT_EXCEEDED"
)

// InvokeError is the machine-readable failure of a run. Retryable hints whether
// invoking again could succeed.
type InvokeError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

// InvocationMetrics is the resource use of one run. Times are in milliseconds.
type InvocationMetrics struct {
	CPUTimeMs       int64 `json:"cpuTimeMs"`
	WallClockMs     int64 `json:"wallClockMs"`
	MemoryPeakBytes int64 `json:"memoryPeakBytes"`
}

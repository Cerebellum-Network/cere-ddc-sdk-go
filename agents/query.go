package agents

import (
	"encoding/json"
	"time"
)

// LogQuery is the GET /v1/logs request: the log records of one agent bundle.
// The gateway serves one cluster, so a query names no cluster.
type LogQuery struct {
	BucketID string
	CID      string
	Since    time.Time
	Until    time.Time
	Level    string // debug, info, warn or error (optional)
	Q        string // free text (optional)
	Limit    int    // default 200, max 1000
	Cursor   string // opaque, from the previous page
	// Labels are equality matchers on the labels the runs were invoked with
	// (InvokeRequest.Labels), each sent as its own query parameter. Without
	// them the query returns the bundle's merged stream across all runs. A key
	// that collides with one of the query's own parameters is dropped.
	Labels map[string]string
}

// LogQueryResponse is one page of log records.
type LogQueryResponse struct {
	Records []map[string]any `json:"records"`
	Cursor  string           `json:"cursor,omitempty"`
}

// MetricsQuery is the GET /v1/metrics request: approximate resource use of one
// agent bundle, from the cluster's metrics store. Without Step it is an instant
// query; with Step it is a range query over [From, To].
type MetricsQuery struct {
	Bucket  string
	CID     string
	Metric  string            // an allowed runtime metric name (required)
	Labels  map[string]string // optional matchers; each key must be an allowed label of Metric; keys named like a field above are dropped
	GroupBy string            // optional allowed label
	From    time.Time         // optional
	To      time.Time         // optional
	Step    string            // optional; selects a range query
}

// MetricsQueryResponse carries the metrics store's Prometheus-format result in
// Data.
type MetricsQueryResponse struct {
	Metric string          `json:"metric"`
	Query  string          `json:"query"`
	Data   json.RawMessage `json:"data"`
}

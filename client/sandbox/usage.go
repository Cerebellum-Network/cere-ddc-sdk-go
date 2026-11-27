package sandbox

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
)

// UsageQuery selects usage records; every field is optional.
type UsageQuery struct {
	SandboxID string
	// Labels select the records of sandboxes that carry all of them, for
	// example {"user": "fred"}: how an application splits its bucket's usage
	// among its own users.
	Labels map[string]string
	// Since bounds the records' time.
	Since time.Time
	// Limit is the most records returned: 0 is 100 for UsageRecords and all
	// for UsageByLabel, a negative value all.
	Limit int
}

// UsageRecords lists the bucket's usage records, one per sandbox per minute
// plus a final one, newest first.
func (s *Service) UsageRecords(ctx context.Context, q UsageQuery) ([]UsageRecord, error) {
	return s.usageRecords(ctx, q, limit(q.Limit))
}

func (s *Service) usageRecords(ctx context.Context, q UsageQuery, n int) ([]UsageRecord, error) {
	v := labelQuery(q.Labels)
	if q.SandboxID != "" {
		v.Set("sandbox_id", q.SandboxID)
	}
	if !q.Since.IsZero() {
		v.Set("since", rfc3339(q.Since))
	}
	return client.List[UsageRecord](ctx, s.c,
		client.Request{Method: http.MethodGet, Path: "/v1/usage/records", Query: v}, n)
}

// LabelUsage is the billed and measured usage of the sandboxes that share one
// value of a label.
type LabelUsage struct {
	Sandboxes           int
	Records             int
	BilledCPUSeconds    float64
	BilledRAMGiBSeconds float64
	CPUSeconds          float64
	RAMGiBSeconds       float64
}

// UsageByLabel sums the bucket's usage records per value of the label key:
// for "user", {"fred": {...}, "ann": {...}}. Sandboxes without the label count
// under "". q.Labels narrows to sandboxes that also carry those labels (for
// example {"cere.agent": "research"}). The bucket is billed; splitting the
// bill among an application's own users is the application's, on labels it
// sets on each sandbox. To sum the sandboxes of one label value, UsageSummary
// asks the API instead.
func (s *Service) UsageByLabel(ctx context.Context, key string, q UsageQuery) (map[string]LabelUsage, error) {
	n := q.Limit
	if n < 0 {
		n = 0
	}
	records, err := s.usageRecords(ctx, q, n)
	if err != nil {
		return nil, err
	}
	out := map[string]LabelUsage{}
	seen := map[string]map[string]bool{}
	for _, r := range records {
		v := r.Labels[key]
		u := out[v]
		if seen[v] == nil {
			seen[v] = map[string]bool{}
		}
		if !seen[v][r.SandboxID] {
			seen[v][r.SandboxID] = true
			u.Sandboxes++
		}
		u.Records++
		u.BilledCPUSeconds += r.BilledCPUSeconds
		u.BilledRAMGiBSeconds += r.BilledRAMGiBSeconds
		u.CPUSeconds += r.CPUSeconds
		u.RAMGiBSeconds += r.RAMGiBSeconds
		out[v] = u
	}
	return out, nil
}

// UsageSummaryQuery selects the sandboxes whose usage records
// UsageSummary sums; every field is optional.
type UsageSummaryQuery struct {
	// Labels must all match the sandbox's labels, for example
	// {"task": "t-81"}.
	Labels map[string]string
	// SandboxID narrows to one sandbox.
	SandboxID string
	// Since and Until bound the records' period end: [Since, Until).
	Since, Until time.Time
	// Limit is the most sandboxes listed: 0 is 100, a negative value all.
	// Totals covers every matching sandbox whatever the limit.
	Limit int
}

// UsageSums is usage summed over usage records: measured (CPUSeconds,
// RAMGiBSeconds, network), billed (BilledCPUSeconds, BilledRAMGiBSeconds),
// the request's floor (Reserved*) and what was billed above it (Burst*),
// with the highest CPU and memory use of any record.
type UsageSums struct {
	Records               int     `json:"records"`
	WallSeconds           float64 `json:"wall_s"`
	CPUSeconds            float64 `json:"cpu_s"`
	RAMGiBSeconds         float64 `json:"ram_gib_s"`
	BilledCPUSeconds      float64 `json:"billed_cpu_s"`
	BilledRAMGiBSeconds   float64 `json:"billed_ram_gib_s"`
	ReservedCPUSeconds    float64 `json:"reserved_cpu_s"`
	ReservedRAMGiBSeconds float64 `json:"reserved_ram_gib_s"`
	BurstCPUSeconds       float64 `json:"burst_cpu_s"`
	BurstRAMGiBSeconds    float64 `json:"burst_ram_gib_s"`
	NetEgressBytes        int64   `json:"net_egress_bytes"`
	NetIngressBytes       int64   `json:"net_ingress_bytes"`
	CPUPeak               float64 `json:"cpu_peak"`
	MemoryPeakMiB         float64 `json:"memory_peak_mib"`
}

// UsageTotals is the usage of every sandbox a summary matched.
type UsageTotals struct {
	Sandboxes int `json:"sandboxes"`
	UsageSums
}

// SandboxUsage is one sandbox's usage in a summary.
type SandboxUsage struct {
	SandboxID string            `json:"sandbox_id"`
	Labels    map[string]string `json:"labels"`
	Status    string            `json:"status"`
	StartedAt *time.Time        `json:"started_at"`
	EndedAt   *time.Time        `json:"ended_at"`
	UsageSums
	// CPUAvg is CPUSeconds / WallSeconds: the cores used on average.
	CPUAvg float64 `json:"cpu_avg"`
}

// UsageSummary is the bucket's usage summed per sandbox, for the sandboxes
// a query matched, and in all.
type UsageSummary struct {
	Totals UsageTotals `json:"totals"`
	// Sandboxes are newest first, at most the query's Limit.
	Sandboxes []SandboxUsage `json:"sandboxes"`
}

// UsageSummary sums the bucket's usage records per sandbox, for the
// sandboxes whose labels match q.Labels (for example everything one task
// ran), and in all: the API does the sums.
func (s *Service) UsageSummary(ctx context.Context, q UsageSummaryQuery) (*UsageSummary, error) {
	v := labelQuery(q.Labels)
	if q.SandboxID != "" {
		v.Set("sandbox_id", q.SandboxID)
	}
	if !q.Since.IsZero() {
		v.Set("since", rfc3339(q.Since))
	}
	if !q.Until.IsZero() {
		v.Set("until", rfc3339(q.Until))
	}
	n := limit(q.Limit)
	var out *UsageSummary
	seen := map[string]bool{}
	for {
		want := client.MaxPageSize
		if n > 0 {
			want = min(want, n-len(sandboxesOf(out)))
		}
		page := url.Values{}
		for k, vs := range v {
			page[k] = vs
		}
		page.Set("limit", strconv.Itoa(want))
		var res struct {
			UsageSummary
			NextCursor *string `json:"next_cursor"`
		}
		if err := s.c.Do(ctx, &client.Request{Method: http.MethodGet, Path: "/v1/usage/summary", Query: page}, &res); err != nil {
			return nil, err
		}
		if out == nil {
			out = &UsageSummary{Totals: res.Totals, Sandboxes: []SandboxUsage{}}
		}
		out.Sandboxes = append(out.Sandboxes, res.Sandboxes...)
		if res.NextCursor == nil || *res.NextCursor == "" || len(res.Sandboxes) == 0 || (n > 0 && len(out.Sandboxes) >= n) {
			if n > 0 && len(out.Sandboxes) > n {
				out.Sandboxes = out.Sandboxes[:n]
			}
			return out, nil
		}
		if seen[*res.NextCursor] {
			return nil, fmt.Errorf("sandbox: usage summary: the API gave cursor %q a second time", *res.NextCursor)
		}
		seen[*res.NextCursor] = true
		v.Set("cursor", *res.NextCursor)
	}
}

func sandboxesOf(s *UsageSummary) []SandboxUsage {
	if s == nil {
		return nil
	}
	return s.Sandboxes
}

// Limits is how many sandboxes the bucket may have active (starting,
// running or unknown) at once, and how many it has. One more create answers
// CodeLimitActiveSandboxes; the cluster's operator can raise the limit for
// a bucket.
type Limits struct {
	Bucket             uint64 `json:"bucket"`
	MaxActiveSandboxes int    `json:"max_active_sandboxes"`
	ActiveSandboxes    int    `json:"active_sandboxes"`
}

// Limits reads the bucket's sandbox limits.
func (s *Service) Limits(ctx context.Context) (*Limits, error) {
	var out Limits
	if err := s.c.Do(ctx, &client.Request{Method: http.MethodGet, Path: "/v1/limits"}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Billing states of a usage record, from its DACStatus.
const (
	BillingCounted    = "Counted"
	BillingWaiting    = "Waiting"
	BillingNotCounted = "Not counted"
)

// BillingState says whether a usage record has reached billing, as the `cere`
// CLI's BILLING column says it: Counted (dac_status "pushed"), Waiting
// ("pending": not sent yet) or Not counted ("failed"). A status this SDK does
// not know is returned as it is, and none as "-".
func BillingState(dacStatus string) string {
	switch dacStatus {
	case "pushed":
		return BillingCounted
	case "pending":
		return BillingWaiting
	case "failed":
		return BillingNotCounted
	case "":
		return "-"
	}
	return dacStatus
}

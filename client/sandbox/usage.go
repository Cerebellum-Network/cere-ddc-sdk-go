package sandbox

import (
	"context"
	"net/http"
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
// sets on each sandbox.
func (s *Service) UsageByLabel(ctx context.Context, key string, q UsageQuery) (map[string]LabelUsage, error) {
	sandboxes, err := s.List(ctx, ListOptions{Labels: q.Labels, Limit: -1})
	if err != nil {
		return nil, err
	}
	value := make(map[string]string, len(sandboxes))
	for _, sb := range sandboxes {
		value[sb.ID] = sb.Labels[key]
	}
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
		v := value[r.SandboxID]
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

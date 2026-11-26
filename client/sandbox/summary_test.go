package sandbox_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client/sandbox"
)

func TestUsageSummary(t *testing.T) {
	_, sbx := newFake(t)
	var ids []string
	for _, task := range []string{"t-81", "t-82", "t-81", "t-81"} {
		sb, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Labels: map[string]string{"task": task, "app": "demo"}})
		_ = sb.Terminate(ctx, sandbox.TerminateOptions{})
		ids = append(ids, sb.ID)
	}
	sum, err := sbx.UsageSummary(ctx, sandbox.UsageSummaryQuery{Labels: map[string]string{"task": "t-81"},
		Since: time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC), Limit: -1})
	if err != nil {
		t.Fatal(err)
	}
	if sum.Totals.Sandboxes != 3 || sum.Totals.Records != 3 || fmt.Sprintf("%.1f", sum.Totals.BilledCPUSeconds) != "9.6" || len(sum.Sandboxes) != 3 {
		t.Fatalf("%+v", sum)
	}
	first := sum.Sandboxes[0]
	if first.SandboxID != ids[3] || first.Labels["task"] != "t-81" || first.Status != sandbox.StatusTerminated ||
		first.CPUAvg != 0.04 || first.WallSeconds != 12.5 || first.Records != 1 {
		t.Errorf("%+v", first)
	}
	// Totals cover every match whatever the limit; the pages are followed.
	two, err := sbx.UsageSummary(ctx, sandbox.UsageSummaryQuery{Labels: map[string]string{"task": "t-81"}, Limit: 2})
	if err != nil || len(two.Sandboxes) != 2 || two.Totals.Sandboxes != 3 {
		t.Fatalf("%+v %v", two, err)
	}
	one, err := sbx.UsageSummary(ctx, sandbox.UsageSummaryQuery{SandboxID: ids[1]})
	if err != nil || len(one.Sandboxes) != 1 || one.Sandboxes[0].Labels["task"] != "t-82" {
		t.Fatalf("%+v %v", one, err)
	}

	// Usage records carry their sandbox's labels, and UsageByLabel groups by them.
	records, err := sbx.UsageRecords(ctx, sandbox.UsageQuery{SandboxID: ids[1]})
	if err != nil || len(records) != 1 || records[0].Labels["task"] != "t-82" {
		t.Fatalf("records %+v %v", records, err)
	}
	by, err := sbx.UsageByLabel(ctx, "task", sandbox.UsageQuery{Labels: map[string]string{"app": "demo"}})
	if err != nil || by["t-81"].Sandboxes != 3 || by["t-82"].Records != 1 || len(by) != 2 {
		t.Fatalf("%+v %v", by, err)
	}
}

func TestEgressSummaryAndLimits(t *testing.T) {
	f, sbx := newFake(t)
	sb, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"})
	other, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"})
	f.egressLog = append(f.egressLog,
		map[string]any{"sandbox_id": sb.ID, "decision": "allowed", "host": "api.github.com", "credential": "github-readonly", "bytes_down": 800},
		map[string]any{"sandbox_id": sb.ID, "decision": "allowed", "host": "api.github.com", "credential": "github-readonly", "bytes_up": 10},
		map[string]any{"sandbox_id": sb.ID, "decision": "denied", "host": "pypi.org"},
		map[string]any{"sandbox_id": sb.ID, "decision": "error", "host": "api.github.com", "credential": "github-readonly"},
		map[string]any{"sandbox_id": sb.ID, "decision": "policy", "host": "pypi.org"},
		map[string]any{"sandbox_id": other.ID, "decision": "allowed", "host": "pypi.org"},
	)
	s, err := sbx.EgressSummary(ctx, sb.ID, time.Now().Add(-time.Hour), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if s.SandboxID != sb.ID || s.Requests != 4 || s.Allowed != 2 || s.Denied != 1 || s.Errors != 1 || s.Policy != 1 ||
		s.BytesUp != 10 || s.BytesDown != 800 || len(s.ByHost) != 2 || len(s.ByCredential) != 1 {
		t.Fatalf("%+v", s)
	}
	if h := s.ByHost[0]; h.Host != "api.github.com" || h.Requests != 3 || h.Errors != 1 || h.BytesDown != 800 {
		t.Errorf("by host %+v", h)
	}
	if c := s.ByCredential[0]; c.Credential != "github-readonly" || c.Requests != 3 || c.Allowed != 2 {
		t.Errorf("by credential %+v", c)
	}
	if _, err := sbx.EgressSummary(ctx, "sb-nope", time.Time{}, time.Time{}); !client.IsCode(err, client.CodeNotFound) {
		t.Errorf("another bucket's sandbox: %v", err)
	}
	if _, err := sbx.EgressSummary(ctx, "", time.Time{}, time.Time{}); err == nil {
		t.Error("no sandbox: no error")
	}

	_ = other.Terminate(ctx, sandbox.TerminateOptions{})
	l, err := sbx.Limits(ctx)
	if err != nil || *l != (sandbox.Limits{Bucket: 4116, MaxActiveSandboxes: 20, ActiveSandboxes: 1}) {
		t.Fatalf("%+v %v", l, err)
	}
}

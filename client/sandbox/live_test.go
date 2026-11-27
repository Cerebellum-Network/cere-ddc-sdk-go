//go:build live

// The live test runs every call of the package against a real cluster. It is
// not part of `go test ./...`: run it with
//
//	CERE_TOKEN_FILE=<file with a COMPUTE token> go test -tags live -run Live -v ./sandbox/
//
// or with what `cere login` saved (CERE_TOKEN, CERE_API_URL and the config
// file are read as client.LoadConfig reads them). TestLive covers a sandbox's
// calls, detached exec included; TestLiveRetainAndEvents a sandbox whose
// command exits keeping files, its events and its usage by label. With
// CERE_GRANT_FILE (an encoded grant) and CERE_KEY_FILE (the grant's subject,
// an ed25519 seed in hex), TestLiveDelegation also creates a sandbox with
// tokens minted beneath the grant. Everything it creates is labelled
// purpose=cere-sdk-go-live and terminated or deleted at the end. Values it
// sends are dummies; it never prints a token.
package sandbox_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client/sandbox"
)

func liveClient(t *testing.T) *client.Client {
	t.Helper()
	cfg, err := client.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	token := string(cfg.Token)
	if f := os.Getenv("CERE_TOKEN_FILE"); f != "" {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		token = strings.TrimSpace(string(b))
	}
	if token == "" {
		t.Skip("no token: set CERE_TOKEN_FILE or CERE_TOKEN, or run `cere login`")
	}
	c, err := client.New(client.WithBaseURL(cfg.BaseURL), client.WithToken(token),
		client.WithUserAgent("cere-sdk-go-live-test"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// probe runs in the sandbox: GET url, print status and the Authorization
// header httpbin saw (or the error).
const probe = `
import json, sys, urllib.request, urllib.error
try:
    with urllib.request.urlopen(sys.argv[1], timeout=20) as r:
        body = r.read()
        try:
            seen = json.loads(body)["headers"].get("Authorization")
        except Exception:
            seen = "-"
        print(r.status, seen)
except urllib.error.HTTPError as e:
    print(e.code, "http error")
except Exception as e:
    print("refused", type(e).__name__)
`

func TestLive(t *testing.T) {
	c := liveClient(t)
	sbx := sandbox.New(c)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	user := fmt.Sprintf("u%06d", rand.IntN(1_000_000))
	labels := map[string]string{"purpose": "cere-sdk-go-live", "user": user}

	start := time.Now()
	sb, err := sbx.Create(ctx, sandbox.CreateRequest{
		Image:     "python:3.12-slim",
		Timeout:   5 * time.Minute,
		Resources: sandbox.Resources{CPU: 0.25, Memory: 256, CPULimit: 1, MemoryLimit: 512},
		Env:       map[string]string{"GREETING": "hello"},
		Labels:    labels,
		Egress:    []sandbox.Allow{{Host: "httpbin.org", Methods: []string{"GET"}, Credential: "gosdk-run"}},
		Secrets:   []sandbox.Secret{sandbox.BearerSecret("gosdk-run", "dummy-run-token-1", "httpbin.org")},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Logf("created %s on node %.12s… in %s: %+v", sb.ID, sb.Node, time.Since(start).Round(time.Millisecond), sb.Resources)
	defer func() {
		if !sb.Ended() {
			_ = sb.Terminate(context.Background(), sandbox.TerminateOptions{Reason: "cleanup"})
		}
	}()
	if !sb.Running() {
		t.Fatalf("status %s", sb.Status)
	}
	if sb.Resources.Memory != 256 || sb.Resources.CPU != 0.25 {
		t.Errorf("resources: got %+v, want the request cpu 0.25, memory 256", sb.Resources)
	}
	if sb.Labels["user"] != user || len(sb.Secrets) != 1 || sb.Secrets[0].Name != "gosdk-run" {
		t.Errorf("labels %v, secrets %+v", sb.Labels, sb.Secrets)
	}

	// Get and List.
	got, err := sbx.Get(ctx, sb.ID)
	if err != nil || got.ID != sb.ID {
		t.Fatalf("get: %v", err)
	}
	listed, err := sbx.List(ctx, sandbox.ListOptions{Labels: labels, Status: sandbox.StatusRunning})
	if err != nil || len(listed) != 1 || listed[0].ID != sb.ID {
		t.Fatalf("list by label: %d sandboxes, %v", len(listed), err)
	}

	// Exec, synchronous and streamed.
	res, err := sb.Exec(ctx, sandbox.ExecRequest{Command: []string{"python", "-c",
		"import os,sys; print(os.environ['GREETING'], sys.stdin.read().strip())"}, Stdin: "from stdin"})
	if err != nil || res.ExitCode != 0 || res.Stdout != "hello from stdin\n" {
		t.Fatalf("exec: %+v %v", res, err)
	}
	t.Logf("exec: %q in %d ms", res.Stdout, res.DurationMS)
	var out, errs bytes.Buffer
	st, err := sb.ExecStream(ctx, sandbox.ExecRequest{Command: []string{"sh", "-c",
		"for i in 1 2 3; do echo out $i; echo err $i >&2; sleep 1; done; exit 3"}}, &out, &errs)
	if err != nil || st.ExitCode != 3 || out.String() != "out 1\nout 2\nout 3\n" || errs.String() != "err 1\nerr 2\nerr 3\n" {
		t.Fatalf("exec stream: %+v %q %q %v", st, out.String(), errs.String(), err)
	}
	t.Logf("exec stream: exit %d after %d ms", st.ExitCode, st.DurationMS)
	if _, err := sb.Exec(ctx, sandbox.ExecRequest{Command: nil}); err == nil {
		t.Error("exec without a command: no error")
	}

	// Detached exec: started with a key (a repeat starts nothing), waited
	// for, then read back whole and from an offset.
	detach := sandbox.ExecRequest{Command: []string{"sh", "-c",
		"for i in 1 2 3 4 5; do echo line $i; sleep 1; done; echo done >&2; exit 4"}, IdempotencyKey: "gosdk-live:" + user + ":1"}
	ex, err := sb.ExecDetach(ctx, detach)
	if err != nil {
		t.Fatalf("exec detach: %v", err)
	}
	again, err := sb.ExecDetach(ctx, detach)
	if err != nil || !again.Replayed() || again.ExecID != ex.ExecID {
		t.Errorf("exec detach again with the key: %+v %v", again, err)
	}
	if code, err := ex.Wait(ctx); err != nil || code != 4 || ex.StdoutBytes != 35 || ex.StderrBytes != 5 {
		t.Fatalf("wait: %d %v %+v", code, err, ex)
	}
	whole, err := ex.Output(ctx, sandbox.OutputQuery{})
	if err != nil || whole.Stdout != "line 1\nline 2\nline 3\nline 4\nline 5\n" || whole.Stderr != "done\n" ||
		whole.StdoutOffset != 35 || whole.Status != sandbox.ExecStatusExited {
		t.Fatalf("output: %+v %v", whole, err)
	}
	from := sandbox.OutputQuery{StdoutOffset: 14, StderrOffset: whole.StderrOffset} // after "line 2\n"
	if rest, err := ex.Output(ctx, from); err != nil || rest.Stdout != "line 3\nline 4\nline 5\n" || rest.Stderr != "" {
		t.Errorf("output from offset 14: %+v %v", rest, err)
	}
	var tail bytes.Buffer
	if exit, err := ex.Stream(ctx, from, &tail, nil); err != nil || exit.ExitCode == nil || *exit.ExitCode != 4 ||
		tail.String() != "line 3\nline 4\nline 5\n" {
		t.Errorf("stream from offset 14: %+v %q %v", exit, tail.String(), err)
	}
	t.Logf("detached exec %s: exit 4 after %d ms; read back from offset 14", ex.ExecID, ex.DurationMS)
	// Followed while it runs.
	ticks, err := sb.ExecDetach(ctx, sandbox.ExecRequest{Command: []string{"sh", "-c", "for i in 1 2 3; do echo tick $i; sleep 1; done"}})
	if err != nil {
		t.Fatal(err)
	}
	var live bytes.Buffer
	if exit, err := ticks.Stream(ctx, sandbox.OutputQuery{}, &live, nil); err != nil || exit.ExitCode == nil || *exit.ExitCode != 0 || live.String() != "tick 1\ntick 2\ntick 3\n" {
		t.Errorf("stream a running detached exec: %+v %q %v", exit, live.String(), err)
	}
	if execs, err := sb.Execs(ctx); err != nil || len(execs) != 2 || execs[0].ExecID != ticks.ExecID {
		t.Errorf("execs: %d %v", len(execs), err)
	}
	if got, err := sb.GetExec(ctx, ex.ExecID); err != nil || got.IdempotencyKey != detach.IdempotencyKey || got.ExitCode == nil || *got.ExitCode != 4 {
		t.Errorf("get exec: %+v %v", got, err)
	}

	// Files.
	if err := sb.WriteFile(ctx, "/workspace/probe.py", []byte(probe)); err != nil {
		t.Fatal(err)
	}
	if err := sb.WriteFile(ctx, "/workspace/out/result.json", []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	if b, err := sb.ReadFile(ctx, "/workspace/out/result.json"); err != nil || string(b) != `{"ok":true}` {
		t.Fatalf("read file: %q %v", b, err)
	}
	if _, err := sb.ReadFile(ctx, "/workspace/missing"); !client.IsCode(err, sandbox.CodeFileNotFound) {
		t.Errorf("read a missing file: %v", err)
	}
	if err := sb.WriteFile(ctx, "/workspace/big", make([]byte, sandbox.MaxFileBytes+1)); !client.IsCode(err, client.CodeTooLarge) {
		t.Errorf("write a file over 16 MiB: %v", err)
	}

	// Egress with the sandbox's own secret, then a new value mid-run.
	call := func(url string) string {
		r, err := sb.Exec(ctx, sandbox.ExecRequest{Command: []string{"python", "/workspace/probe.py", url}, Timeout: time.Minute})
		if err != nil {
			t.Fatalf("probe %s: %v", url, err)
		}
		return strings.TrimSpace(r.Stdout)
	}
	if got := call("https://httpbin.org/headers"); got != "200 Bearer dummy-run-token-1" {
		t.Errorf("header mode: %q", got)
	}
	if err := sb.SetSecret(ctx, "gosdk-run", "Bearer dummy-run-token-2"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(6 * time.Second) // other proxy replicas within 5 s
	if got := call("https://httpbin.org/headers"); got != "200 Bearer dummy-run-token-2" {
		t.Errorf("after SetSecret: %q", got)
	}
	if err := sb.SetSecret(ctx, "no-such-secret", "Bearer x"); !client.IsCode(err, sandbox.CodeSecretNotFound) {
		t.Errorf("SetSecret of an unknown name: %v", err)
	}
	if got := call("https://pypi.org/simple/"); !strings.HasPrefix(got, "refused") && !strings.HasPrefix(got, "403") {
		t.Errorf("pypi before a rule: %q", got)
	}

	// Egress rules: read, add, remove, replace.
	rules, err := sb.EgressRules(ctx)
	if err != nil || len(rules) != 1 {
		t.Fatalf("egress rules: %+v %v", rules, err)
	}
	rules, err = sb.Allow(ctx, sandbox.MustParseAllow("pypi.org:GET:/simple/"), sandbox.MustParseAllow("httpbin.org:HEAD::gosdk-run"))
	if err != nil || len(rules) != 2 || rules[0].Host != "pypi.org" || strings.Join(rules[1].Methods, ",") != "GET,HEAD" {
		t.Fatalf("allow: %+v %v", rules, err)
	}
	if got := call("https://pypi.org/simple/requests/"); !strings.HasPrefix(got, "200") {
		t.Errorf("pypi after the rule: %q", got)
	}
	if rules, err = sb.RemoveEgress(ctx, "pypi.org"); err != nil || len(rules) != 1 {
		t.Fatalf("remove: %+v %v", rules, err)
	}
	if rules, err = sb.SetEgress(ctx, nil); err != nil || len(rules) != 0 {
		t.Fatalf("set egress []: %+v %v", rules, err)
	}
	if got := call("https://httpbin.org/headers"); !strings.HasPrefix(got, "refused") && !strings.HasPrefix(got, "403") {
		t.Errorf("httpbin after removing every rule: %q", got)
	}
	_, err = sb.SetEgress(ctx, []sandbox.Allow{{Host: "httpbin.org", Methods: []string{"GET"}, Credential: "no-such-secret"}})
	if !client.IsCode(err, sandbox.CodeUnknownCredential) {
		t.Errorf("a rule naming an unknown credential: %v", err)
	}

	// The bucket's and an agent's secrets.
	bucketSecret := "gosdk-live-" + user
	if err := sbx.SetSecret(ctx, sandbox.Secret{Name: bucketSecret, Value: "Bearer dummy", Hosts: []string{"httpbin.org"}}); err != nil {
		t.Fatal(err)
	}
	if err := sbx.SetAgentSecret(ctx, "gosdk-live", sandbox.Secret{Name: bucketSecret, Header: "X-Api-Key", Value: "dummy", Hosts: []string{"httpbin.org"}}); err != nil {
		t.Fatal(err)
	}
	secrets, err := sbx.ListSecrets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var bucketSeen, agentSeen bool
	for _, s := range secrets {
		if s.Name == bucketSecret && s.Agent == "" {
			bucketSeen = true
		}
		if s.Name == bucketSecret && s.Agent == "gosdk-live" && s.Header == "X-Api-Key" {
			agentSeen = true
		}
	}
	if !bucketSeen || !agentSeen {
		t.Errorf("list secrets: bucket %v, agent %v", bucketSeen, agentSeen)
	}
	if err := sbx.DeleteAgentSecret(ctx, "gosdk-live", bucketSecret); err != nil {
		t.Error(err)
	}
	if err := sbx.DeleteSecret(ctx, bucketSecret); err != nil {
		t.Error(err)
	}

	// Live usage, then terminate with a retained file.
	u, err := sb.CurrentUsage(ctx)
	if err != nil || u.WallSeconds <= 0 {
		t.Fatalf("usage: %+v %v", u, err)
	}
	t.Logf("live usage: wall %.1fs, cpu %.2fs, billed %.1f CPU-s %.1f GiB-s (%s)", u.WallSeconds, u.CPUSeconds,
		u.BilledCPUSeconds, u.BilledRAMGiBSeconds, u.BillingBasis)
	if err := sb.Terminate(ctx, sandbox.TerminateOptions{Retain: []string{"/workspace/out/result.json"}, Reason: "go sdk live test"}); err != nil {
		t.Fatal(err)
	}
	if sb.Status != sandbox.StatusTerminated || sb.ExitReason != "terminated by API: go sdk live test" {
		t.Errorf("after terminate: %s %q", sb.Status, sb.ExitReason)
	}
	t.Logf("terminated: %s; final usage billed %.1f CPU-s (burst %.1f), %.1f GiB-s; retained %v", sb.ExitReason,
		sb.Usage.BilledCPUSeconds, sb.Usage.BurstCPUSeconds, sb.Usage.BilledRAMGiBSeconds, sb.Retained)
	if b, err := sb.ReadFile(ctx, "/workspace/out/result.json"); err != nil || string(b) != `{"ok":true}` {
		t.Errorf("read a retained file: %q %v", b, err)
	}
	if _, err := sb.ReadFile(ctx, "/workspace/probe.py"); !client.IsCode(err, sandbox.CodeFileGone) {
		t.Errorf("read a file that was not retained: %v", err)
	}
	if _, err := sb.Exec(ctx, sandbox.ExecRequest{Command: []string{"true"}}); !client.IsCode(err, sandbox.CodeNotRunning) {
		t.Errorf("exec after the end: %v", err)
	}
	if err := sb.Terminate(ctx, sandbox.TerminateOptions{}); err != nil {
		t.Errorf("terminate twice: %v", err)
	}

	// Usage records, usage by label, the egress log.
	var records []sandbox.UsageRecord
	for i := 0; i < 20; i++ {
		records, err = sbx.UsageRecords(ctx, sandbox.UsageQuery{SandboxID: sb.ID})
		if err != nil {
			t.Fatal(err)
		}
		if len(records) > 0 && records[0].Final {
			break
		}
		time.Sleep(3 * time.Second)
	}
	if len(records) == 0 {
		t.Fatal("no usage records")
	}
	t.Logf("usage records: %d, newest final=%v, billing %s", len(records), records[0].Final, sandbox.BillingState(records[0].DACStatus))
	byUser, err := sbx.UsageByLabel(ctx, "user", sandbox.UsageQuery{Labels: map[string]string{"purpose": "cere-sdk-go-live"}})
	if err != nil {
		t.Fatal(err)
	}
	if mine := byUser[user]; mine.Sandboxes != 1 || mine.Records != len(records) {
		t.Errorf("usage by label: %+v (records %d)", mine, len(records))
	}
	t.Logf("usage by label user=%s: %+v", user, byUser[user])
	entries, err := sbx.EgressLog(ctx, sandbox.EgressLogQuery{SandboxID: sb.ID, Limit: -1})
	if err != nil {
		t.Fatal(err)
	}
	decisions := map[string]int{}
	for _, e := range entries {
		decisions[e.Decision]++
	}
	if decisions[sandbox.DecisionAllowed] == 0 || decisions[sandbox.DecisionDenied] == 0 || decisions[sandbox.DecisionPolicy] == 0 {
		t.Errorf("egress log decisions: %v", decisions)
	}
	t.Logf("egress log: %v", decisions)
	if raw, _ := json.Marshal(entries); bytes.Contains(raw, []byte("dummy-run-token")) {
		t.Error("the egress log carries a secret value")
	}
	summary, err := sbx.EgressSummary(ctx, sb.ID, time.Time{}, time.Time{})
	if err != nil || summary.Requests != int64(decisions[sandbox.DecisionAllowed]+decisions[sandbox.DecisionDenied]+decisions[sandbox.DecisionError]) ||
		summary.Policy != int64(decisions[sandbox.DecisionPolicy]) || len(summary.ByHost) == 0 {
		t.Errorf("egress summary: %+v %v (log %v)", summary, err, decisions)
	}
}

// TestLiveRetainAndEvents runs a sandbox whose main process writes files and
// exits: the files named at create are kept and read after the end, and the
// sandbox's events are read with one long-poll by its label.
func TestLiveRetainAndEvents(t *testing.T) {
	c := liveClient(t)
	sbx := sandbox.New(c)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	task := fmt.Sprintf("t-%06d", rand.IntN(1_000_000))
	labels := map[string]string{"purpose": "cere-sdk-go-live", "task": task}

	limits, err := sbx.Limits(ctx)
	if err != nil || limits.MaxActiveSandboxes <= 0 {
		t.Fatalf("limits: %+v %v", limits, err)
	}
	t.Logf("limits: bucket %d, %d of %d sandboxes active", limits.Bucket, limits.ActiveSandboxes, limits.MaxActiveSandboxes)

	req := sandbox.CreateRequest{
		Image: "python:3.12-slim",
		Command: []string{"sh", "-c", `mkdir -p /workspace/out && printf '{"ok":true}' > /workspace/out/result.json && ` +
			`printf x > /workspace/out/b.txt && sleep 3`},
		Timeout:        2 * time.Minute,
		Labels:         labels,
		Retain:         &sandbox.Retain{Paths: []string{"/workspace/out/", "/workspace/missing.json"}, TTLS: 600},
		IdempotencyKey: "gosdk-live:" + task,
	}
	sb, err := sbx.Create(ctx, req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() {
		if !sb.Ended() {
			_ = sb.Terminate(context.Background(), sandbox.TerminateOptions{Reason: "cleanup"})
		}
	}()
	if sb.Replayed() || sb.Image.Ref != req.Image || sb.IdempotencyKey != req.IdempotencyKey || sb.Retain == nil {
		t.Errorf("created: %+v", sb)
	}
	again, err := sbx.Create(ctx, req, sandbox.NoWait())
	if err != nil || !again.Replayed() || again.ID != sb.ID {
		t.Errorf("create again with the key: %+v %v", again, err)
	}

	code, err := sb.Wait(ctx)
	if err != nil || code != 0 || sb.ExitKind != sandbox.ExitKindMainExited || sb.RetainedUntil == nil {
		t.Fatalf("wait: %d %v; %s %q until %v", code, err, sb.ExitKind, sb.ExitReason, sb.RetainedUntil)
	}
	t.Logf("%s ended: %q; image %s (digest %.19s…, %d bytes, cached %v, pulled in %d ms); retained %v; not retained %v",
		sb.ID, sb.ExitReason, sb.Image.Ref, sb.Image.Digest, sb.Image.SizeBytes, sb.Image.Cached, sb.Image.PullMS,
		sb.Retained, sb.NotRetained)
	kept := map[string]int64{}
	for _, f := range sb.Retained {
		kept[f.Path] = f.Size
	}
	if kept["/workspace/out/result.json"] != 11 || kept["/workspace/out/b.txt"] != 1 {
		t.Errorf("retained %v", sb.Retained)
	}
	if len(sb.NotRetained) != 1 || sb.NotRetained[0].Path != "/workspace/missing.json" {
		t.Errorf("not retained %v", sb.NotRetained)
	}
	if b, err := sb.ReadFile(ctx, "/workspace/out/result.json"); err != nil || string(b) != `{"ok":true}` {
		t.Errorf("read a retained file: %q %v", b, err)
	}

	// One long-poll of the bucket's events, by the task's label.
	evs, next, err := sbx.Events(ctx, sandbox.EventQuery{After: sandbox.FromOldest, Labels: map[string]string{"task": task}, WaitS: 30})
	if err != nil || len(evs) == 0 {
		t.Fatalf("events of task=%s: %d, %v", task, len(evs), err)
	}
	var running, exited *sandbox.Event
	for i, ev := range evs {
		if ev.SandboxID != sb.ID || ev.Labels["task"] != task {
			t.Errorf("an event of another sandbox: %+v", ev)
		}
		switch ev.Type {
		case sandbox.EventSandboxRunning:
			running = &evs[i]
		case sandbox.EventSandboxExited:
			exited = &evs[i]
		}
	}
	if running == nil || running.Image == nil || running.Image.Ref != req.Image {
		t.Errorf("no sandbox.running event: %+v", evs)
	}
	if exited == nil || exited.ExitKind != sandbox.ExitKindMainExited || exited.ExitCode == nil || *exited.ExitCode != 0 ||
		len(exited.Retained) != len(sb.Retained) || exited.Usage == nil || next != evs[len(evs)-1].ID {
		t.Errorf("sandbox.exited event: %+v (next %s)", exited, next)
	}
	t.Logf("events of task=%s: %d, next %s", task, len(evs), next)

	// Usage by the task's label, once the final record is in.
	for i := 0; i < 20; i++ {
		sum, err := sbx.UsageSummary(ctx, sandbox.UsageSummaryQuery{Labels: map[string]string{"task": task}})
		if err != nil {
			t.Fatal(err)
		}
		if sum.Totals.Sandboxes == 1 && sum.Sandboxes[0].Status == sandbox.StatusTerminated {
			t.Logf("usage of task=%s: %d records, %.1f s, %.2f CPU-s (avg %.3f cores), billed %.1f CPU-s", task,
				sum.Totals.Records, sum.Totals.WallSeconds, sum.Totals.CPUSeconds, sum.Sandboxes[0].CPUAvg, sum.Totals.BilledCPUSeconds)
			break
		}
		if i == 19 {
			t.Errorf("usage summary of task=%s: %+v", task, sum)
		}
		time.Sleep(3 * time.Second)
	}

	// A webhook, created and deleted at once; its label matches no sandbox, so
	// nothing is ever sent.
	wh, err := sbx.CreateWebhook(ctx, sandbox.WebhookRequest{URL: "https://hooks.example.com/cere-sdk-go-live",
		Types: []string{sandbox.EventSandboxExited}, Label: "task=none-" + task, Secret: "dummy-secret-" + task})
	if err != nil {
		t.Fatalf("create webhook: %v", err)
	}
	if hooks, err := sbx.ListWebhooks(ctx); err != nil || !slices.ContainsFunc(hooks, func(h sandbox.Webhook) bool { return h.ID == wh.ID }) {
		t.Errorf("list webhooks: %v", err)
	}
	if _, err := sbx.WebhookDeliveries(ctx, wh.ID, 10); err != nil {
		t.Errorf("webhook deliveries: %v", err)
	}
	if err := sbx.DeleteWebhook(ctx, wh.ID); err != nil {
		t.Errorf("delete webhook: %v", err)
	}
}

func TestLiveDelegation(t *testing.T) {
	grantFile, keyFile := os.Getenv("CERE_GRANT_FILE"), os.Getenv("CERE_KEY_FILE")
	if grantFile == "" || keyFile == "" {
		t.Skip("set CERE_GRANT_FILE and CERE_KEY_FILE")
	}
	grant, err := os.ReadFile(grantFile)
	if err != nil {
		t.Fatal(err)
	}
	k, err := os.ReadFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := hex.DecodeString(strings.TrimSpace(string(k)))
	if err != nil || len(seed) != ed25519.SeedSize {
		t.Fatal("CERE_KEY_FILE: want an ed25519 seed in hex")
	}
	d, err := client.ParseDelegation(strings.TrimSpace(string(grant)), auth.NewEd25519Signer(ed25519.NewKeyFromSeed(seed)),
		client.WithCallTTL(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := client.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	c, err := client.New(client.WithBaseURL(cfg.BaseURL), client.WithTokenSource(d))
	if err != nil {
		t.Fatal(err)
	}
	sbx := sandbox.New(c)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	sb, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "python:3.12-slim", Timeout: 2 * time.Minute,
		Labels: map[string]string{"purpose": "cere-sdk-go-live", "auth": "delegation"}})
	if err != nil {
		t.Fatalf("create with a delegation (bucket %d, grant until %s): %v", d.Bucket(), d.ExpiresAt().UTC(), err)
	}
	t.Logf("created %s with a token minted beneath the grant (bucket %d)", sb.ID, d.Bucket())
	res, err := sb.Exec(ctx, sandbox.ExecRequest{Command: []string{"echo", "delegated"}})
	if err != nil || res.Stdout != "delegated\n" {
		t.Errorf("exec: %+v %v", res, err)
	}
	if err := sb.Terminate(ctx, sandbox.TerminateOptions{Reason: "go sdk live delegation"}); err != nil {
		t.Fatal(err)
	}
	t.Logf("terminated: %s", sb.ExitReason)
	// No timeout: 0.
	sb, err = sbx.Create(ctx, sandbox.CreateRequest{Image: "python:3.12-slim", Timeout: sandbox.NoTimeout,
		Labels: map[string]string{"purpose": "cere-sdk-go-live", "auth": "delegation"}})
	if err != nil {
		t.Fatalf("create with NoTimeout: %v", err)
	}
	t.Logf("created %s with NoTimeout: timeout_s %d", sb.ID, sb.TimeoutS)
	if sb.TimeoutS != 0 {
		t.Errorf("NoTimeout resolved to %d", sb.TimeoutS)
	}
	if err := sb.Terminate(ctx, sandbox.TerminateOptions{Reason: "go sdk live no timeout"}); err != nil {
		t.Fatal(err)
	}
}

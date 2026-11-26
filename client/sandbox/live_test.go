//go:build live

// The live test runs every call of the package against a real cluster. It is
// not part of `go test ./...`: run it with
//
//	CERE_TOKEN_FILE=<file with a COMPUTE token> go test -tags live -run Live -v ./sandbox/
//
// or with what `cere login` saved (CERE_TOKEN, CERE_API_URL and the config
// file are read as client.LoadConfig reads them). With CERE_GRANT_FILE (an
// encoded grant) and CERE_KEY_FILE (the grant's subject, an ed25519 seed in
// hex), TestLiveDelegation also creates a sandbox with tokens minted beneath
// the grant. Everything it creates is labelled purpose=cere-sdk-go-live and
// terminated or deleted at the end. Values it sends are dummies; it never
// prints a token.
package sandbox_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
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
	token := cfg.Token
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
	// A sandbox that would outlive the grant is refused before the call.
	tooLong := time.Until(d.ExpiresAt()) + time.Hour
	if _, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "python:3.12-slim", Timeout: tooLong}); !errors.Is(err, client.ErrTokenExpiresTooSoon) {
		t.Errorf("a sandbox that would outlive the grant: %v", err)
	}
}

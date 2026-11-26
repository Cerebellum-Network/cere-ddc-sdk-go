package sandbox_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client/sandbox"
)

var ctx = context.Background()

func TestCreateSendsTheAPIsBody(t *testing.T) {
	f, sbx := newFake(t)
	sb, err := sbx.Create(ctx, sandbox.CreateRequest{
		Image:       "python:3.12-slim",
		Command:     []string{"python", "-m", "http.server"},
		Resources:   sandbox.Resources{CPU: 0.5, Memory: 256, CPULimit: 2, MemoryLimit: 1024, Disk: 512},
		Timeout:     90*time.Second + time.Millisecond,
		IdleTimeout: time.Minute,
		Env:         map[string]string{"A": "1"},
		Labels:      map[string]string{"user": "fred"},
		Egress: []sandbox.Allow{
			{Host: "api.github.com", Methods: []string{"get"}, PathPrefix: "/repos/", Credential: "gh"},
			{Host: "mcp.slack.com", Methods: []string{"*"}, Placeholders: []string{"slack-a", "slack-b"}},
			{Host: "pypi.org"},
		},
		Secrets: []sandbox.Secret{sandbox.BearerSecret("slack-a", "xoxp-1", "mcp.slack.com"),
			{Name: "gh", Header: "Authorization", Value: "token g", Hosts: []string{"api.github.com"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"command":["python","-m","http.server"],"egress":{"allow":[` +
		`{"credential":"gh","host":"api.github.com","methods":["GET"],"path_prefix":"/repos/"},` +
		`{"host":"mcp.slack.com","methods":["*"],"placeholders":["slack-a","slack-b"]},` +
		`{"host":"pypi.org","methods":["GET"]}]},"env":{"A":"1"},"idle_timeout_s":60,"image":"python:3.12-slim",` +
		`"labels":{"user":"fred"},"resources":{"cpu":0.5,"cpu_limit":2,"disk":512,"memory":256,"memory_limit":1024},` +
		`"runtime":"gvisor","secrets":[` +
		`{"header":"Authorization","hosts":["mcp.slack.com"],"name":"slack-a","type":"header","value":"Bearer xoxp-1"},` +
		`{"header":"Authorization","hosts":["api.github.com"],"name":"gh","type":"header","value":"token g"}],"timeout_s":91}`
	if got := mustJSON(f.creates[0]); got != want {
		t.Errorf("create body\n got %s\nwant %s", got, want)
	}
	if !sb.Running() || sb.Resources != (sandbox.Resources{CPU: 0.5, Memory: 256, CPULimit: 2, MemoryLimit: 1024, PIDs: 2048, Disk: 512}) {
		t.Errorf("sandbox: %s %+v", sb.Status, sb.Resources)
	}
	if len(sb.Secrets) != 2 || sb.Secrets[0].Name != "slack-a" || sb.Egress.Allow[1].Placeholders[1] != "slack-b" {
		t.Errorf("secrets %+v, egress %+v", sb.Secrets, sb.Egress)
	}

	// The defaults: only the image and the runtime are sent.
	if _, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"}); err != nil {
		t.Fatal(err)
	}
	if got := mustJSON(f.creates[1]); got != `{"image":"alpine","runtime":"gvisor"}` {
		t.Errorf("minimal create: %s", got)
	}
}

func TestTimeouts(t *testing.T) {
	f, sbx := newFake(t)
	for _, tc := range []struct {
		timeout time.Duration
		want    string
	}{{0, "absent"}, {sandbox.NoTimeout, "0"}, {90 * time.Second, "90"}, {1500 * time.Millisecond, "2"}} {
		sb, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Timeout: tc.timeout})
		if err != nil {
			t.Fatal(err)
		}
		got := "absent"
		if v, ok := f.creates[len(f.creates)-1]["timeout_s"]; ok {
			got = fmt.Sprint(v)
		}
		if got != tc.want {
			t.Errorf("create Timeout %s: timeout_s %s, want %s", tc.timeout, got, tc.want)
		}
		if _, err := sb.Exec(ctx, sandbox.ExecRequest{Command: []string{"true"}, Timeout: tc.timeout}); err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		_ = json.Unmarshal([]byte(f.execs[len(f.execs)-1]), &body)
		got = "absent"
		if v, ok := body["timeout_s"]; ok {
			got = fmt.Sprint(v)
		}
		if got != tc.want {
			t.Errorf("exec Timeout %s: timeout_s %s, want %s", tc.timeout, got, tc.want)
		}
	}
	if _, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Timeout: -2 * time.Second}); err == nil {
		t.Error("a negative timeout other than NoTimeout: no error")
	}
}

func TestCreateChecksBeforeSending(t *testing.T) {
	f, sbx := newFake(t)
	for name, req := range map[string]sandbox.CreateRequest{
		"no image":             {},
		"cpu limit below":      {Image: "a", Resources: sandbox.Resources{CPU: 1, CPULimit: 0.5}},
		"memory limit below":   {Image: "a", Resources: sandbox.Resources{Memory: 512, MemoryLimit: 256}},
		"negative":             {Image: "a", Resources: sandbox.Resources{Memory: -1}},
		"secret without hosts": {Image: "a", Secrets: []sandbox.Secret{{Name: "x", Value: "v"}}},
		"secret bad name":      {Image: "a", Secrets: []sandbox.Secret{{Name: "X_Y", Value: "v", Hosts: []string{"h"}}}},
		"secret two lines":     {Image: "a", Secrets: []sandbox.Secret{{Name: "x", Value: "a\nb", Hosts: []string{"h"}}}},
		"secret host a URL":    {Image: "a", Secrets: []sandbox.Secret{{Name: "x", Value: "v", Hosts: []string{"https://h"}}}},
		"rule host a URL":      {Image: "a", Egress: []sandbox.Allow{{Host: "https://pypi.org"}}},
		"rule with both":       {Image: "a", Egress: []sandbox.Allow{{Host: "h", Credential: "a", Placeholders: []string{"b"}}}},
	} {
		if _, err := sbx.Create(ctx, req); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if len(f.creates) != 0 {
		t.Errorf("%d creates reached the API", len(f.creates))
	}
}

func TestCreateWaitsAndReportsAFailedStart(t *testing.T) {
	f, sbx := newFake(t)
	f.startAs = sandbox.StatusStarting
	sb, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"})
	if err != nil || !sb.Running() {
		t.Fatalf("%v %v", sb, err)
	}
	f.failStart = true
	sb, err = sbx.Create(ctx, sandbox.CreateRequest{Image: "no/such:image"})
	var se *sandbox.StartError
	if !errors.As(err, &se) || se.Status != sandbox.StatusFailed || !strings.Contains(se.ExitReason, "image pull") || sb == nil {
		t.Fatalf("%v %v", sb, err)
	}
	sb, err = sbx.Create(ctx, sandbox.CreateRequest{Image: "no/such:image"}, sandbox.NoWait())
	if err != nil || sb.Status != sandbox.StatusStarting {
		t.Fatalf("NoWait: %v %v", sb, err)
	}
}

func TestCreateRetriesNoCapacityOnly(t *testing.T) {
	f, sbx := newFake(t)
	f.createFail = []int{503}
	if _, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"}); err != nil || len(f.creates) != 2 {
		t.Fatalf("no_capacity once: %v after %d creates", err, len(f.creates))
	}
	f.createFail = []int{409}
	if _, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"}); !client.IsCode(err, sandbox.CodeLimitActiveSandboxes) || len(f.creates) != 3 {
		t.Fatalf("limit: %v after %d creates", err, len(f.creates))
	}
	f.createFail = []int{502}
	if _, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"}); client.StatusOf(err) != 502 || len(f.creates) != 4 {
		t.Fatalf("a 502 on create is not retried: %v after %d creates", err, len(f.creates))
	}
}

func TestCreateTokens(t *testing.T) {
	f, sbx := newFake(t)
	_, owner, _ := ed25519.GenerateKey(rand.Reader)
	_, svcKey, _ := ed25519.GenerateKey(rand.Reader)

	// A static token is sent as it is, even one that expires before the
	// sandbox would end: the token only has to be valid at create.
	short := ownerToken(t, owner, time.Now().Add(5*time.Minute))
	static := sandbox.New(sbx.Client().WithToken(short))
	if _, err := static.Create(ctx, sandbox.CreateRequest{Image: "alpine", Timeout: time.Hour}); err != nil || len(f.creates) != 1 ||
		f.tokens[len(f.tokens)-1] != short {
		t.Fatalf("%v, %d creates", err, len(f.creates))
	}

	// A delegation mints the create's token for the sandbox's timeout + 60 s
	// (+ 2 min), and short tokens for the other calls.
	grant, err := client.NewGrant(auth.NewEd25519Signer(owner), svcKey.Public().(ed25519.PublicKey), 4116, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	d, err := client.ParseDelegation(grant, auth.NewEd25519Signer(svcKey))
	if err != nil {
		t.Fatal(err)
	}
	delegated := sandbox.New(sbx.Client().WithTokenSource(d))
	f.tokens = nil
	sb, err := delegated.Create(ctx, sandbox.CreateRequest{Image: "alpine", Timeout: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sb.Exec(ctx, sandbox.ExecRequest{Command: []string{"true"}}); err != nil {
		t.Fatal(err)
	}
	createExp, _ := client.StaticToken(f.tokens[0]).ExpiresAt()
	callExp, _ := client.StaticToken(f.tokens[len(f.tokens)-1]).ExpiresAt()
	// The create's token only has to be valid at create, whatever the timeout.
	if d := time.Until(createExp); d > client.DefaultCallTTL {
		t.Errorf("the create's token lives %s, want at most %s", d, client.DefaultCallTTL)
	}
	if d := time.Until(callExp); d > client.DefaultCallTTL {
		t.Errorf("an exec's token lives %s, want at most %s", d, client.DefaultCallTTL)
	}
}

func ownerToken(t *testing.T, owner ed25519.PrivateKey, exp time.Time) string {
	t.Helper()
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	grant, err := client.NewGrant(auth.NewEd25519Signer(owner), other.Public().(ed25519.PublicKey), 4116, exp)
	if err != nil {
		t.Fatal(err)
	}
	d, err := client.ParseDelegation(grant, auth.NewEd25519Signer(other))
	if err != nil {
		t.Fatal(err)
	}
	tok, _, err := d.Mint(time.Until(exp), nil)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestGetListAndMemoryNames(t *testing.T) {
	_, sbx := newFake(t)
	var ids []string
	for i := range 5 {
		sb, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Resources: sandbox.Resources{Memory: 256},
			Labels: map[string]string{"user": fmt.Sprint(i % 2), "app": "demo"}})
		if err != nil {
			t.Fatal(err)
		}
		if sb.Resources.Memory != 256 || sb.Resources.MemoryLimit != 1024 {
			t.Errorf("memory and memory_limit: %+v", sb.Resources)
		}
		ids = append(ids, sb.ID)
	}
	got, err := sbx.Get(ctx, ids[2])
	if err != nil || got.ID != ids[2] || got.Labels["user"] != "0" {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := sbx.Get(ctx, "sb-missing"); !client.IsCode(err, client.CodeNotFound) {
		t.Errorf("get a missing sandbox: %v", err)
	}
	all, err := sbx.List(ctx, sandbox.ListOptions{Limit: -1})
	if err != nil || len(all) != 5 || all[0].ID != ids[4] {
		t.Fatalf("list all: %d %v", len(all), err)
	}
	two, err := sbx.List(ctx, sandbox.ListOptions{Limit: 2})
	if err != nil || len(two) != 2 {
		t.Fatalf("list 2: %d %v", len(two), err)
	}
	evens, err := sbx.List(ctx, sandbox.ListOptions{Labels: map[string]string{"user": "0", "app": "demo"}, Status: "running"})
	if err != nil || len(evens) != 3 {
		t.Fatalf("list by labels: %d %v", len(evens), err)
	}
	// A listed sandbox is a handle.
	if _, err := evens[0].Exec(ctx, sandbox.ExecRequest{Command: []string{"true"}}); err != nil {
		t.Fatal(err)
	}
}

func TestExec(t *testing.T) {
	f, sbx := newFake(t)
	sb, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"})
	res, err := sb.Exec(ctx, sandbox.ExecRequest{Command: []string{"cat"}, Stdin: "in"})
	if err != nil || res.ExitCode != 0 || res.Stdout != "cat <in\n" || res.DurationMS != 4 {
		t.Fatalf("%+v %v", res, err)
	}
	var both bytes.Buffer
	var errs bytes.Buffer
	st, err := sb.ExecStream(ctx, sandbox.ExecRequest{Command: []string{"echo", "hi"}, Timeout: 42 * time.Second}, &both, &errs)
	// The fake answers the request's timeout_s as the exit code.
	if err != nil || st.ExitCode != 42 || both.String() != "out:echo hi\n" || errs.String() != "err\n" {
		t.Fatalf("%+v %q %q %v", st, both.String(), errs.String(), err)
	}
	if _, err := sb.ExecStream(ctx, sandbox.ExecRequest{Command: []string{"x"}}, nil, nil); err != nil {
		t.Fatalf("nil writers: %v", err)
	}
	if _, err := sb.Exec(ctx, sandbox.ExecRequest{}); err == nil {
		t.Error("no command: no error")
	}
	n := len(f.tokens)
	_ = sb.Terminate(ctx, sandbox.TerminateOptions{})
	if _, err := sb.Exec(ctx, sandbox.ExecRequest{Command: []string{"true"}}); !client.IsCode(err, sandbox.CodeNotRunning) {
		t.Errorf("exec after the end: %v", err)
	}
	if _, err := sb.ExecStream(ctx, sandbox.ExecRequest{Command: []string{"true"}}, nil, nil); !client.IsCode(err, sandbox.CodeNotRunning) {
		t.Errorf("exec stream after the end: %v", err)
	}
	if len(f.tokens) != n+3 {
		t.Errorf("a refused exec was retried: %d calls", len(f.tokens)-n)
	}
}

func TestFilesLogsUsageAndTerminate(t *testing.T) {
	f, sbx := newFake(t)
	sb, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"})
	if err := sb.WriteFile(ctx, "/workspace/a.txt", []byte("A")); err != nil {
		t.Fatal(err)
	}
	if err := sb.WriteFile(ctx, "/workspace/empty", nil); err != nil {
		t.Fatal(err)
	}
	if b, err := sb.ReadFile(ctx, "/workspace/a.txt"); err != nil || string(b) != "A" {
		t.Fatalf("%q %v", b, err)
	}
	if _, err := sb.ReadFile(ctx, "/nope"); !client.IsCode(err, sandbox.CodeFileNotFound) {
		t.Errorf("missing file: %v", err)
	}
	n := len(f.tokens)
	if err := sb.WriteFile(ctx, "/big", make([]byte, sandbox.MaxFileBytes+1)); !client.IsCode(err, client.CodeTooLarge) || len(f.tokens) != n {
		t.Errorf("a file over 16 MiB: %v (sent: %v)", err, len(f.tokens) != n)
	}

	var out, errs bytes.Buffer
	if err := sb.Logs(ctx, &out, &errs); err != nil || out.String() != "booted\n" || errs.String() != "warn\n" {
		t.Fatalf("logs: %q %q %v", out.String(), errs.String(), err)
	}
	u, err := sb.CurrentUsage(ctx)
	if err != nil || u.WallSeconds != 7.5 || u.BilledCPUSeconds != 1.9 || u.BillingBasis != "request_or_usage" {
		t.Fatalf("usage: %+v %v", u, err)
	}

	if err := sb.Terminate(ctx, sandbox.TerminateOptions{Retain: []string{"/workspace/a.txt"}, Reason: "done"}); err != nil {
		t.Fatal(err)
	}
	if !sb.Ended() || sb.ExitReason != "terminated by API: done" || sb.Usage.WallSeconds != 12.5 || sb.Retained[0] != "/workspace/a.txt" {
		t.Fatalf("after terminate: %+v", sb)
	}
	if b, err := sb.ReadFile(ctx, "/workspace/a.txt"); err != nil || string(b) != "A" {
		t.Errorf("a retained file: %q %v", b, err)
	}
	if _, err := sb.ReadFile(ctx, "/workspace/empty"); !client.IsCode(err, sandbox.CodeFileGone) {
		t.Errorf("a file not retained: %v", err)
	}
	if err := sb.Terminate(ctx, sandbox.TerminateOptions{}); err != nil {
		t.Errorf("terminate again: %v", err)
	}
	if code, err := sb.Wait(ctx); err != nil || code != -1 {
		t.Errorf("wait: %d %v", code, err)
	}
	if err := sb.Terminate(ctx, sandbox.TerminateOptions{Retain: []string{"/a,b"}}); err == nil {
		t.Error("a retained path with a comma: no error")
	}
}

func TestEgress(t *testing.T) {
	f, sbx := newFake(t)
	sb, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine",
		Egress: []sandbox.Allow{sandbox.MustParseAllow("api.github.com:GET::gh")}})
	rules, err := sb.EgressRules(ctx)
	if err != nil || len(rules) != 1 || rules[0].Credential != "gh" {
		t.Fatalf("%+v %v", rules, err)
	}
	rules, err = sb.Allow(ctx, sandbox.MustParseAllow("pypi.org:GET:/simple/"), sandbox.MustParseAllow("api.github.com:HEAD::gh"))
	if err != nil || mustJSON(rules) != `[{"host":"pypi.org","methods":["GET"],"path_prefix":"/simple/"},`+
		`{"host":"api.github.com","methods":["GET","HEAD"],"credential":"gh"}]` {
		t.Fatalf("allow: %s %v", mustJSON(rules), err)
	}
	rules, err = sb.RemoveEgress(ctx, "api.github.com", "head")
	if err != nil || len(rules) != 2 || strings.Join(rules[1].Methods, ",") != "GET" {
		t.Fatalf("remove a method: %+v %v", rules, err)
	}
	if _, err := sb.RemoveEgress(ctx, "example.com"); err == nil {
		t.Error("remove a host with no rule: no error")
	}
	if rules, err = sb.RemoveEgress(ctx, "PYPI.org"); err != nil || len(rules) != 1 {
		t.Fatalf("remove a host: %+v %v", rules, err)
	}
	if rules, err = sb.SetEgress(ctx, nil); err != nil || len(rules) != 0 {
		t.Fatalf("set none: %+v %v", rules, err)
	}
	if body := f.sandboxes[sb.ID]["egress"].(map[string]any)["allow"]; body == nil {
		t.Error("an empty list must be sent as allow: []")
	}
	if _, err := sb.SetEgress(ctx, []sandbox.Allow{{Host: "x.example", Credential: "nope"}}); !client.IsCode(err, sandbox.CodeUnknownCredential) {
		t.Errorf("unknown credential: %v", err)
	}
	entries, err := sbx.EgressLog(ctx, sandbox.EgressLogQuery{SandboxID: sb.ID, Since: time.Now().Add(-time.Hour), Limit: -1})
	if err != nil || len(entries) != 4 || entries[0].Decision != sandbox.DecisionPolicy || entries[0].Path != "" {
		t.Fatalf("egress log: %+v %v", entries, err)
	}
	_ = sb.Terminate(ctx, sandbox.TerminateOptions{})
	if _, err := sb.Allow(ctx, sandbox.MustParseAllow("pypi.org")); !client.IsCode(err, sandbox.CodeNotRunning) {
		t.Errorf("allow after the end: %v", err)
	}
}

func TestSecrets(t *testing.T) {
	f, sbx := newFake(t)
	if err := sbx.SetSecret(ctx, sandbox.Secret{Name: "github", Value: "Bearer g", Hosts: []string{"api.github.com"}}); err != nil {
		t.Fatal(err)
	}
	if err := sbx.SetAgentSecret(ctx, "research", sandbox.Secret{Name: "github", Header: "X-Token", Value: "a", Hosts: []string{"*.github.com"}}); err != nil {
		t.Fatal(err)
	}
	if err := sbx.SetAgentSecret(ctx, "Bad Agent", sandbox.Secret{Name: "x", Value: "v", Hosts: []string{"h"}}); err == nil {
		t.Error("a bad agent name: no error")
	}
	if got := mustJSON(f.secrets["/github"]); got != `{"header":"Authorization","hosts":["api.github.com"],"name":"github","type":"header","value":"Bearer g"}` {
		t.Errorf("bucket secret body: %s", got)
	}
	list, err := sbx.ListSecrets(ctx)
	if err != nil || len(list) != 2 || list[0].Agent != "" || list[1].Agent != "research" || list[1].Header != "X-Token" {
		t.Fatalf("list: %+v %v", list, err)
	}
	if err := sbx.DeleteAgentSecret(ctx, "research", "github"); err != nil {
		t.Fatal(err)
	}
	if err := sbx.DeleteSecret(ctx, "github"); err != nil {
		t.Fatal(err)
	}
	if err := sbx.DeleteSecret(ctx, "github"); !client.IsCode(err, client.CodeNotFound) {
		t.Errorf("delete twice: %v", err)
	}

	// The sandbox's own secret, replaced while it runs.
	sb, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine",
		Secrets: []sandbox.Secret{sandbox.BearerSecret("notion", "t1", "api.notion.com")}})
	if err := sb.SetSecret(ctx, "notion", "Bearer t2"); err != nil || f.runSecret[sb.ID]["notion"] != "Bearer t2" {
		t.Fatalf("%v %v", err, f.runSecret[sb.ID])
	}
	if err := sb.SetSecret(ctx, "slack", "Bearer x"); !client.IsCode(err, sandbox.CodeSecretNotFound) {
		t.Errorf("unknown name: %v", err)
	}
	if err := sb.SetSecret(ctx, "notion", ""); err == nil {
		t.Error("an empty value: no error")
	}

	// A secret never prints its value.
	s := sandbox.BearerSecret("notion", "very-secret", "api.notion.com")
	for _, out := range []string{fmt.Sprint(s), fmt.Sprintf("%v %+v %#v %s", s, s, s, s), fmt.Sprint([]sandbox.Secret{s})} {
		if strings.Contains(out, "very-secret") {
			t.Fatalf("printed: %s", out)
		}
	}
	if sandbox.Placeholder("slack-a") != "cere:slack-a" {
		t.Error("placeholder")
	}
}

func TestUsageRecordsAndByLabel(t *testing.T) {
	_, sbx := newFake(t)
	for _, user := range []string{"fred", "ann", "fred", ""} {
		labels := map[string]string{"app": "demo"}
		if user != "" {
			labels["user"] = user
		}
		sb, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Labels: labels})
		_ = sb.Terminate(ctx, sandbox.TerminateOptions{})
	}
	other, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Labels: map[string]string{"user": "fred"}})
	_ = other.Terminate(ctx, sandbox.TerminateOptions{})

	records, err := sbx.UsageRecords(ctx, sandbox.UsageQuery{SandboxID: other.ID})
	if err != nil || len(records) != 1 || records[0].MemoryLimit != 1024 || !records[0].Final {
		t.Fatalf("records: %+v %v", records, err)
	}
	if sandbox.BillingState(records[0].DACStatus) != sandbox.BillingCounted {
		t.Error("billing state")
	}
	by, err := sbx.UsageByLabel(ctx, "user", sandbox.UsageQuery{Labels: map[string]string{"app": "demo"}})
	if err != nil {
		t.Fatal(err)
	}
	if by["fred"].Sandboxes != 2 || by["fred"].Records != 2 || by["ann"].Sandboxes != 1 || by[""].Sandboxes != 1 ||
		len(by) != 3 || fmt.Sprintf("%.1f", by["fred"].BilledCPUSeconds) != "6.4" {
		t.Fatalf("by label: %+v", by)
	}
}

func TestBillingState(t *testing.T) {
	for in, want := range map[string]string{"pushed": "Counted", "pending": "Waiting", "failed": "Not counted", "": "-", "new": "new"} {
		if got := sandbox.BillingState(in); got != want {
			t.Errorf("%q: %q", in, got)
		}
	}
}

func TestParseAllow(t *testing.T) {
	for in, want := range map[string]string{
		"pypi.org":                           `{"host":"pypi.org","methods":["GET"]}`,
		"pypi.org:GET:/simple/":              `{"host":"pypi.org","methods":["GET"],"path_prefix":"/simple/"}`,
		"api.github.com:get,head::gh":        `{"host":"api.github.com","methods":["GET","HEAD"],"credential":"gh"}`,
		"*.example.com:*":                    `{"host":"*.example.com","methods":["*"]}`,
		"h:POST:/a:b:c":                      `{"host":"h","methods":["POST"],"path_prefix":"/a","credential":"b:c"}`,
		"files.pythonhosted.org::/packages/": `{"host":"files.pythonhosted.org","methods":["GET"],"path_prefix":"/packages/"}`,
	} {
		a, err := sandbox.ParseAllow(in)
		if err != nil || mustJSON(a) != want {
			t.Errorf("%q: %s %v", in, mustJSON(a), err)
		}
	}
	for _, bad := range []string{"", ":GET", "h:,", "h:GET:simple/"} {
		if _, err := sandbox.ParseAllow(bad); err == nil {
			t.Errorf("%q: no error", bad)
		}
	}
}

func TestSandboxDecodesTheAPIsShape(t *testing.T) {
	raw := `{"id":"sb-1","status":"terminated","bucket":4116,"resources":{"cpu":0.25,"memory":128,"cpu_limit":4,
		"memory_limit":1024,"pids":2048,"disk":512,"gpu":null},"timeout_s":300,
		"egress":{"allow":[{"host":"pypi.org","methods":["GET"]}]},"started_at":"2026-11-25T10:00:01Z","ended_at":null,
		"startup_ms":900,"exit_reason":"oom_killed","exit_code":137,"usage":{"wall_s":60,"cpu_s":1.5,"billing_basis":
		"request_or_usage","reserved_cpu_s":15,"burst_cpu_s":2.5,"cpu_peak":3.5}}`
	var sb sandbox.Sandbox
	if err := json.Unmarshal([]byte(raw), &sb); err != nil {
		t.Fatal(err)
	}
	if !sb.Ended() || sb.Bucket != 4116 || sb.Resources.Memory != 128 || sb.Resources.Disk != 512 || sb.TimeoutS != 300 || *sb.ExitCode != 137 ||
		sb.ExitReason != "oom_killed" || sb.EndedAt != nil || sb.StartedAt == nil || sb.Usage.BurstCPUSeconds != 2.5 ||
		sb.Usage.ReservedCPUSeconds != 15 || sb.Egress.Allow[0].Host != "pypi.org" {
		t.Fatalf("%+v", sb)
	}
}

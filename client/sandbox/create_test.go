package sandbox_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client/sandbox"
)

func TestIdempotentCreate(t *testing.T) {
	f, sbx := newFake(t)
	req := sandbox.CreateRequest{Image: "alpine", Labels: map[string]string{"task": "t-81"}, IdempotencyKey: "task:t-81:1",
		Secrets: []sandbox.Secret{sandbox.BearerSecret("gh", "v1", "api.github.com")}}
	first, err := sbx.Create(ctx, req)
	if err != nil || first.Replayed() || first.IdempotencyKey != "task:t-81:1" {
		t.Fatalf("%+v %v", first, err)
	}
	if _, inBody := f.creates[0]["idempotency_key"]; inBody {
		t.Error("the key is a header, not a field of the body")
	}
	// The same key again, a secret's value aside: the same sandbox, nothing created.
	req.Secrets = []sandbox.Secret{sandbox.BearerSecret("gh", "v2", "api.github.com")}
	again, err := sbx.Create(ctx, req)
	if err != nil || !again.Replayed() || again.ID != first.ID || len(f.order) != 1 {
		t.Fatalf("replay: %+v %v (%d sandboxes)", again, err, len(f.order))
	}
	// Replayed survives a refresh: it is about the create.
	if err := again.Refresh(ctx); err != nil || !again.Replayed() {
		t.Fatalf("after refresh: %v %v", again.Replayed(), err)
	}
	// The same key with another body.
	req.Image = "python:3.12-slim"
	if _, err := sbx.Create(ctx, req); !client.IsCode(err, sandbox.CodeIdempotencyConflict) {
		t.Errorf("conflict: %v", err)
	}
	// A create whose answer was lost: with a key it is retried, and the
	// retry answers the sandbox the first try made.
	f.loseCreate = 1
	n := len(f.creates)
	lost, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", IdempotencyKey: "task:t-82:1"})
	if err != nil || !lost.Replayed() || len(f.creates) != n+2 || len(f.order) != 2 {
		t.Fatalf("lost answer: %+v %v after %d creates", lost, err, len(f.creates)-n)
	}
	// Without a key it is not: the sandbox may have been made.
	f.loseCreate = 1
	n = len(f.creates)
	if _, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"}); client.StatusOf(err) != 502 || len(f.creates) != n+1 {
		t.Fatalf("lost answer without a key: %v after %d creates", err, len(f.creates)-n)
	}
	// A replayed create of a sandbox that ran and ended is no start error.
	f.exitMain(first.ID, 0)
	ended, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Labels: map[string]string{"task": "t-81"},
		IdempotencyKey: "task:t-81:1", Secrets: req.Secrets})
	if err != nil || !ended.Replayed() || !ended.Ended() || ended.ExitKind != sandbox.ExitKindMainExited {
		t.Fatalf("replay of an ended sandbox: %+v %v", ended, err)
	}
	for _, bad := range []string{strings.Repeat("k", 256), "tab\tkey", "é"} {
		if _, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", IdempotencyKey: bad}); err == nil {
			t.Errorf("key %q: no error", bad)
		}
	}
}

func TestRetainAtCreate(t *testing.T) {
	f, sbx := newFake(t)
	sb, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Command: []string{"python", "agent.py"},
		Retain: &sandbox.Retain{Paths: []string{"/workspace/output.json", "/workspace/out/", "/workspace/transcript.jsonl"}, TTLS: 600}})
	if err != nil {
		t.Fatal(err)
	}
	if got := mustJSON(f.creates[0]["retain"]); got != `{"paths":["/workspace/output.json","/workspace/out/","/workspace/transcript.jsonl"],"ttl_s":600}` {
		t.Errorf("retain sent as %s", got)
	}
	for path, data := range map[string]string{"/workspace/output.json": "{}", "/workspace/out/b.txt": "bb", "/workspace/out/a.txt": "a", "/workspace/scratch": "x"} {
		if err := sb.WriteFile(ctx, path, []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	// The main process exits: the files are kept whatever the code.
	f.exitMain(sb.ID, 3)
	code, err := sb.Wait(ctx)
	if err != nil || code != 3 || sb.ExitKind != sandbox.ExitKindMainExited || sb.RetainedUntil == nil {
		t.Fatalf("wait: %d %v %+v", code, err, sb)
	}
	if got := fmt.Sprint(sb.Retained); got != "[{/workspace/output.json 2} {/workspace/out/a.txt 1} {/workspace/out/b.txt 2}]" {
		t.Errorf("retained %s", got)
	}
	if len(sb.NotRetained) != 1 || sb.NotRetained[0].Path != "/workspace/transcript.jsonl" || sb.NotRetained[0].Reason == "" {
		t.Errorf("not retained %+v", sb.NotRetained)
	}
	if until := time.Until(*sb.RetainedUntil); until < 9*time.Minute || until > 11*time.Minute {
		t.Errorf("retained until %s from now, want the 600 s asked", until)
	}
	for _, kept := range sb.Retained {
		if b, err := sb.ReadFile(ctx, kept.Path); err != nil || int64(len(b)) != kept.Size {
			t.Errorf("read %s: %q %v", kept.Path, b, err)
		}
	}
	if _, err := sb.ReadFile(ctx, "/workspace/scratch"); !client.IsCode(err, sandbox.CodeFileGone) {
		t.Errorf("a file not retained: %v", err)
	}

	// On failure: nothing kept when the command succeeds; Terminate's own
	// paths are kept even so.
	ok, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine",
		Retain: &sandbox.Retain{Paths: []string{"/workspace/a"}, On: sandbox.RetainOnFailure}})
	_ = ok.WriteFile(ctx, "/workspace/a", []byte("a"))
	f.exitMain(ok.ID, 0)
	if _, err := ok.Wait(ctx); err != nil || len(ok.Retained) != 0 {
		t.Errorf("on failure, exit 0: %+v %v", ok.Retained, err)
	}
	failed, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine",
		Retain: &sandbox.Retain{Paths: []string{"/workspace/a"}, On: sandbox.RetainOnFailure}})
	_ = failed.WriteFile(ctx, "/workspace/a", []byte("a"))
	_ = failed.WriteFile(ctx, "/workspace/b", []byte("b"))
	if err := failed.Terminate(ctx, sandbox.TerminateOptions{Retain: []string{"/workspace/b"}}); err != nil ||
		fmt.Sprint(failed.Retained) != "[{/workspace/b 1}]" {
		t.Errorf("on failure, terminated: %+v %v", failed.Retained, err)
	}

	n := len(f.creates)
	for name, r := range map[string]*sandbox.Retain{
		"no paths":     {},
		"relative":     {Paths: []string{"workspace/a"}},
		"root":         {Paths: []string{"/"}},
		"dot dot":      {Paths: []string{"/workspace/../etc"}},
		"dot":          {Paths: []string{"/workspace/./a"}},
		"unknown on":   {Paths: []string{"/a"}, On: "success"},
		"negative ttl": {Paths: []string{"/a"}, TTLS: -1},
	} {
		if _, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Retain: r}); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if len(f.creates) != n {
		t.Errorf("%d creates with a bad retain reached the API", len(f.creates)-n)
	}
}

func TestCreateWaitsForASandboxThatRanAndEnded(t *testing.T) {
	f, sbx := newFake(t)
	f.startAs = sandbox.StatusStarting
	sb, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Command: []string{"true"}}, sandbox.NoWait())
	if err != nil {
		t.Fatal(err)
	}
	f.exitMain(sb.ID, 0)
	if err := sb.WaitRunning(ctx); err != nil || !sb.Ended() {
		t.Fatalf("a sandbox whose command exited at once: %v %s", err, sb.Status)
	}
}

func TestPrivateImages(t *testing.T) {
	f, sbx := newFake(t)
	if err := sbx.SetSecret(ctx, sandbox.RegistrySecret("ghcr-pull", "acme-bot", "ghp_pull", "ghcr.io")); err != nil {
		t.Fatal(err)
	}
	if got := mustJSON(f.secrets["/ghcr-pull"]); got != `{"hosts":["ghcr.io"],"name":"ghcr-pull","type":"registry","username":"acme-bot","value":"ghp_pull"}` {
		t.Errorf("registry secret body: %s", got)
	}
	list, err := sbx.ListSecrets(ctx)
	if err != nil || len(list) != 1 || list[0].Type != sandbox.SecretTypeRegistry || list[0].Username != "acme-bot" || list[0].Header != "" {
		t.Fatalf("list: %+v %v", list, err)
	}
	sb, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "ghcr.io/acme/agent:1.4", RegistryCredential: "ghcr-pull"})
	if err != nil || sb.RegistryCredential != "ghcr-pull" || sb.Image.Ref != "ghcr.io/acme/agent:1.4" || sb.Image.Digest == "" {
		t.Fatalf("%+v %v", sb, err)
	}
	if f.creates[0]["registry_credential"] != "ghcr-pull" {
		t.Errorf("create body %s", mustJSON(f.creates[0]))
	}
	if _, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "acme/agent:1.4", RegistryCredential: "ghcr-pull"}); !client.IsCode(err, sandbox.CodeCredentialNotBound) {
		t.Errorf("a Docker Hub image with a ghcr.io credential: %v", err)
	}
	if _, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "ghcr.io/acme/agent:1.4", RegistryCredential: "nope"}); !client.IsCode(err, sandbox.CodeUnknownCredential) {
		t.Errorf("an unknown credential: %v", err)
	}

	s := sandbox.RegistrySecret("ghcr-pull", "acme-bot", "very-secret", "ghcr.io")
	if out := fmt.Sprintf("%v %+v %#v", s, s, s); strings.Contains(out, "very-secret") || !strings.Contains(out, "acme-bot") {
		t.Errorf("printed: %s", out)
	}
	n := len(f.creates)
	for name, s := range map[string]sandbox.Secret{
		"registry without username": {Name: "r", Type: sandbox.SecretTypeRegistry, Value: "p", Hosts: []string{"ghcr.io"}},
		"registry with a header":    {Name: "r", Type: sandbox.SecretTypeRegistry, Username: "u", Header: "X", Value: "p", Hosts: []string{"ghcr.io"}},
		"header with a username":    {Name: "h", Username: "u", Value: "v", Hosts: []string{"h"}},
		"unknown type":              {Name: "h", Type: "oauth", Value: "v", Hosts: []string{"h"}},
	} {
		if err := sbx.SetSecret(ctx, s); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if _, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Secrets: []sandbox.Secret{s}}); err == nil || len(f.creates) != n {
		t.Errorf("a registry secret as a sandbox's own: %v", err)
	}
}

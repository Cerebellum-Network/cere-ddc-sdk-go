// Command quickstart runs a sandbox end to end: create it with a request,
// limits, egress rules and a credential of its own, run commands, move files,
// widen its egress while it runs, end it keeping one file, and read what it
// cost.
//
// With what `cere login` saved, or an access token from the developer console:
//
//	CERE_TOKEN=<token> go run github.com/cerebellum-network/cere-ddc-sdk-go/client/examples/quickstart@latest
//
// For a service that holds a grant from the bucket owner and its own key
// instead of a token, set CERE_GRANT_FILE (the encoded grant) and
// CERE_KEY_FILE (the ed25519 seed the grant names, in hex).
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client/sandbox"
)

// probe runs in the sandbox: it calls httpbin.org, which echoes the
// Authorization header the egress proxy added, and pypi.org.
const probe = `
import json, sys, urllib.request
for url in sys.argv[1:]:
    try:
        with urllib.request.urlopen(url, timeout=20) as r:
            body = r.read()
            seen = json.loads(body)["headers"].get("Authorization") if "httpbin" in url else len(body)
            print(url, r.status, seen)
    except Exception as e:
        print(url, "refused:", e)
`

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	c, err := newClient()
	if err != nil {
		log.Fatal(err)
	}
	sbx := sandbox.New(c)

	// 1. Create: 0.25 vCPU and 256 MiB guaranteed, bursting to 1 vCPU and
	// 512 MiB; httpbin.org reachable with a credential of the sandbox's own,
	// which the egress proxy adds and the sandbox never sees.
	sb, err := sbx.Create(ctx, sandbox.CreateRequest{
		Image:     "python:3.12-slim",
		Timeout:   5 * time.Minute,
		Resources: sandbox.Resources{CPU: 0.25, Memory: 256, CPULimit: 1, MemoryLimit: 512},
		Labels:    map[string]string{"app": "quickstart", "user": "demo"},
		Egress:    []sandbox.Allow{{Host: "httpbin.org", Methods: []string{"GET"}, Credential: "httpbin"}},
		Secrets:   []sandbox.Secret{sandbox.BearerSecret("httpbin", "demo-token-1", "httpbin.org")},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if !sb.Ended() {
			_ = sb.Terminate(context.Background(), sandbox.TerminateOptions{Reason: "quickstart failed"})
		}
	}()
	fmt.Printf("sandbox %s %s: cpu %g (up to %g) vCPU, memory %d (up to %d) MiB\n", sb.ID, sb.Status,
		sb.Resources.CPU, sb.Resources.CPULimit, sb.Resources.Memory, sb.Resources.MemoryLimit)

	// 2. Exec: whole output, then streamed as it comes.
	res, err := sb.Exec(ctx, sandbox.ExecRequest{Command: []string{"python", "-c", "import platform; print(platform.python_version())"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("python %s", res.Stdout)
	st, err := sb.ExecStream(ctx, sandbox.ExecRequest{Command: []string{"sh", "-c", "for i in 1 2 3; do echo tick $i; sleep 1; done"}},
		os.Stdout, os.Stderr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("exit %d after %d ms\n", st.ExitCode, st.DurationMS)

	// 3. Files, and a request through the egress proxy.
	if err := sb.WriteFile(ctx, "/workspace/probe.py", []byte(probe)); err != nil {
		log.Fatal(err)
	}
	run := func() {
		r, err := sb.Exec(ctx, sandbox.ExecRequest{Command: []string{"sh", "-c",
			"python /workspace/probe.py https://httpbin.org/headers https://pypi.org/simple/cere/ | tee /workspace/out.txt"}})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(r.Stdout)
	}
	run() // httpbin answers with the proxy's header; pypi.org is refused

	// 4. While it runs: a new value for its credential, and pypi.org allowed.
	if err := sb.SetSecret(ctx, "httpbin", "Bearer demo-token-2"); err != nil {
		log.Fatal(err)
	}
	if _, err := sb.Allow(ctx, sandbox.MustParseAllow("pypi.org:GET:/simple/")); err != nil {
		log.Fatal(err)
	}
	time.Sleep(6 * time.Second) // every proxy replica has it within 5 s
	run()

	// 5. End it, keeping one file for 24 hours, and read what it cost.
	if err := sb.Terminate(ctx, sandbox.TerminateOptions{Retain: []string{"/workspace/out.txt"}, Reason: "quickstart done"}); err != nil {
		log.Fatal(err)
	}
	kept, err := sb.ReadFile(ctx, "/workspace/out.txt")
	if err != nil {
		log.Fatal(err)
	}
	u := sb.Usage
	fmt.Printf("%s; kept %d bytes\n", sb.ExitReason, len(kept))
	fmt.Printf("billed %.1f CPU-s (%.1f request + %.1f burst; measured %.1f), %.1f GiB-s over %.1f s\n",
		u.BilledCPUSeconds, u.ReservedCPUSeconds, u.BurstCPUSeconds, u.CPUSeconds, u.BilledRAMGiBSeconds, u.WallSeconds)
	entries, err := sbx.EgressLog(ctx, sandbox.EgressLogQuery{SandboxID: sb.ID})
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		fmt.Printf("egress %-7s %-6s %s%s %s\n", e.Decision, e.Method, e.Host, e.Path, e.Reason)
	}
}

// newClient signs calls with tokens minted beneath a grant when
// CERE_GRANT_FILE and CERE_KEY_FILE are set, else with the token `cere login`
// saved or CERE_TOKEN.
func newClient() (*client.Client, error) {
	grantFile, keyFile := os.Getenv("CERE_GRANT_FILE"), os.Getenv("CERE_KEY_FILE")
	if grantFile == "" || keyFile == "" {
		return client.NewFromEnv(client.WithUserAgent("cere-quickstart"))
	}
	grant, err := os.ReadFile(grantFile)
	if err != nil {
		return nil, err
	}
	k, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	seed, err := hex.DecodeString(strings.TrimSpace(string(k)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("%s: want an ed25519 seed in hex", keyFile)
	}
	d, err := client.ParseDelegation(strings.TrimSpace(string(grant)), auth.NewEd25519Signer(ed25519.NewKeyFromSeed(seed)))
	if err != nil {
		return nil, err
	}
	cfg, err := client.LoadConfig()
	if err != nil {
		return nil, err
	}
	return client.New(client.WithBaseURL(cfg.BaseURL), client.WithTokenSource(d), client.WithUserAgent("cere-quickstart"))
}

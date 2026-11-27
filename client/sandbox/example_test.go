package sandbox_test

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client/sandbox"
)

// Run a command in a sandbox with a token from the developer console (or what
// `cere login` saved: client.NewFromEnv).
func Example() {
	ctx := context.Background()
	c, err := client.New(client.WithToken(os.Getenv("CERE_TOKEN")))
	if err != nil {
		log.Fatal(err)
	}
	sbx := sandbox.New(c)

	sb, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "python:3.12-slim", Timeout: 10 * time.Minute})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sb.Terminate(ctx, sandbox.TerminateOptions{}) }()

	res, err := sb.Exec(ctx, sandbox.ExecRequest{Command: []string{"python", "-c", "print('hello from a sandbox')"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(res.Stdout)
}

// A service that runs sandboxes for a bucket it does not own: the owner signs a
// grant to the service's key once, and the client mints a short token beneath
// it for every call, and for each create one that outlives the sandbox.
func Example_delegation() {
	var (
		grant      string             // encoded; signed by the bucket owner (client.NewGrant, or their wallet)
		serviceKey ed25519.PrivateKey // the key the grant names
	)
	d, err := client.ParseDelegation(grant, auth.NewEd25519Signer(serviceKey))
	if err != nil {
		log.Fatal(err)
	}
	c, err := client.New(client.WithTokenSource(d))
	if err != nil {
		log.Fatal(err)
	}
	sbx := sandbox.New(c)
	sb, err := sbx.Create(context.Background(), sandbox.CreateRequest{Image: "python:3.12-slim", Timeout: time.Hour})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sb.ID)
}

// Credentials for one run: the egress proxy adds each person's token to the
// requests its rules allow; the sandbox never sees a value. The run is labelled
// with the person, and its usage read by that label.
func Example_credentials() {
	ctx := context.Background()
	c, _ := client.NewFromEnv()
	sbx := sandbox.New(c)
	var slackA, slackB, notion string // access tokens the application holds for one person

	sb, err := sbx.Create(ctx, sandbox.CreateRequest{
		Image:  "ghcr.io/example/agent:1",
		Labels: map[string]string{"user": "fred"},
		Egress: []sandbox.Allow{
			{Host: "api.notion.com", Methods: []string{"GET", "POST"}, Credential: "notion"},
			// Two accounts on one host: the sandbox writes "Authorization: Bearer cere:slack-a"
			// (sandbox.Placeholder("slack-a")) and the proxy swaps in that account's token.
			{Host: "mcp.slack.com", Methods: []string{"*"}, Placeholders: []string{"slack-a", "slack-b"}},
		},
		Secrets: []sandbox.Secret{
			sandbox.BearerSecret("notion", notion, "api.notion.com"),
			sandbox.BearerSecret("slack-a", slackA, "mcp.slack.com"),
			sandbox.BearerSecret("slack-b", slackB, "mcp.slack.com"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	// The token was refreshed: the next request uses the new one.
	_ = sb.SetSecret(ctx, "notion", "Bearer "+notion)

	_ = sb.Terminate(ctx, sandbox.TerminateOptions{Retain: []string{"/workspace/output.json"}, Reason: "run done"})
	byUser, _ := sbx.UsageByLabel(ctx, "user", sandbox.UsageQuery{Since: time.Now().Add(-24 * time.Hour)})
	fmt.Printf("fred: %.1f CPU-s billed\n", byUser["fred"].BilledCPUSeconds)
}

// Change what a running sandbox may reach.
func ExampleSandbox_Allow() {
	ctx := context.Background()
	c, _ := client.NewFromEnv()
	sb, _ := sandbox.New(c).Get(ctx, "sb-01jdx3q4w8m2v6r9t0kz5c7b1n")

	// What pip install needs, added to the rules the sandbox has.
	rules, err := sb.Allow(ctx,
		sandbox.MustParseAllow("pypi.org:GET:/simple/"),
		sandbox.MustParseAllow("files.pythonhosted.org:GET:/packages/"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(rules), "rules")
	_, _ = sb.RemoveEgress(ctx, "pypi.org")
}

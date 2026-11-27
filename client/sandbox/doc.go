// Package sandbox is the sandbox service of Cere's developer platform:
// containers that run in gVisor sandboxes on the cluster's nodes, reach only
// the hosts their egress rules allow, and are billed per second for the CPU
// and memory they request and use.
//
//	c, err := client.New(client.WithToken(token))
//	sbx := sandbox.New(c)
//
//	sb, err := sbx.Create(ctx, sandbox.CreateRequest{
//		Image:     "python:3.12-slim",
//		Timeout:   10 * time.Minute,
//		Resources: sandbox.Resources{CPU: 0.5, Memory: 512, MemoryLimit: 1024},
//		Egress:    []sandbox.Allow{sandbox.MustParseAllow("pypi.org:GET:/simple/")},
//		Labels:    map[string]string{"user": "fred"},
//		Retain:    &sandbox.Retain{Paths: []string{"/workspace/out.json"}},
//	})
//	defer sb.Terminate(ctx, sandbox.TerminateOptions{})
//
//	res, err := sb.Exec(ctx, sandbox.ExecRequest{Command: []string{"python", "-c", "print(1)"}})
//	fmt.Print(res.Stdout)
//
// What a sandbox gets:
//
//   - Resources: a request (CPU, Memory) that is guaranteed and billed as a
//     floor while it runs, and limits (CPULimit, MemoryLimit), the most it can
//     use; use above the request is billed as used.
//   - Egress: nothing is reachable but the hosts its Allow rules list. A rule
//     may name a credential, which the egress proxy adds to the requests it
//     allows; the sandbox never sees the value.
//   - Secrets in three scopes: the sandbox's own (CreateRequest.Secrets,
//     replaced mid-run with Sandbox.SetSecret), an agent's
//     (Service.SetAgentSecret) and the bucket's (Service.SetSecret). A
//     registry secret (RegistrySecret) pulls a private image
//     (CreateRequest.RegistryCredential).
//   - Labels, to list its sandboxes, filter their events and split usage
//     (Service.UsageByLabel, Service.UsageSummary).
//
// A create with an IdempotencyKey is safe to repeat: a repeat answers the
// sandbox the first one made (Sandbox.Replayed). CreateRequest.Retain keeps
// files whatever ends the sandbox, its command exiting or a timeout included;
// after the end Sandbox.Retained lists them and ReadFile reads them until
// RetainedUntil.
//
// Exec runs a command and returns its output; ExecStream writes it as it
// comes. ExecDetach starts one that outlives its caller: Exec.Output,
// Exec.Stream and Exec.Follow read its output from byte offsets, resuming
// where a dropped connection or a restarted caller left off. WriteFile and
// ReadFile move files (up to 16 MiB).
//
// Service.Events and Service.StreamEvents read the bucket's lifecycle events
// (sandbox.running, sandbox.exited, exec.exited) after a cursor; a webhook
// (Service.CreateWebhook) gets them pushed, signed (VerifyWebhook).
// CurrentUsage, UsageRecords and UsageSummary say what was measured and
// billed; EgressLog lists every request the egress proxy allowed or refused,
// and EgressSummary counts them. Limits reads how many sandboxes the bucket
// may have active.
//
// The HTTP contract is the compute gateway's sandbox API (/v1/sandboxes, with
// its events, webhooks, egress, usage and limits under it, and /v1/secrets).
// Errors are *client.Error with the API's codes (CodeNotRunning,
// CodeNoCapacity, ...).
package sandbox

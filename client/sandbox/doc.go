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
//	})
//	defer sb.Terminate(ctx, sandbox.TerminateOptions{Retain: []string{"/workspace/out.json"}})
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
//     (Service.SetAgentSecret) and the bucket's (Service.SetSecret).
//   - Labels, to list its sandboxes and split usage (Service.UsageByLabel).
//
// Exec runs a command and returns its output; ExecStream writes it as it
// comes. WriteFile and ReadFile move files (up to 16 MiB); Terminate with
// Retain keeps files readable for 24 hours after the end. CurrentUsage and
// UsageRecords say what was measured and billed; EgressLog lists every request
// the egress proxy allowed or refused.
//
// The HTTP contract is the compute gateway's sandbox API (/v1/sandboxes,
// /v1/secrets, /v1/egress/log, /v1/usage/records). Errors are *client.Error
// with the API's codes (CodeNotRunning, CodeNoCapacity, ...).
package sandbox

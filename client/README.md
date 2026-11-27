# client

```
go get github.com/cerebellum-network/cere-ddc-sdk-go/client
```

The Go client of Cere's developer platform: one client for every service the platform serves. Sandboxes are the first; storage, CDN, functions and inference come later as packages beside them, on the same client.

| Package | What |
|---|---|
| `client` | The base: the API's URL, access tokens (a static token, or tokens minted per call from a delegation), errors, timeouts, retries, paged listings and event streams |
| `client/sandbox` | gVisor sandboxes: create with a request and limits (idempotent with a key, from a private registry), exec (whole, streamed, or detached and read back from byte offsets), files, files kept whatever ends the sandbox, logs, egress rules (read, replace, add, remove), secrets in three scopes, replacing a secret while it runs, lifecycle events (polled, streamed, or pushed to signed webhooks), usage and usage by label, the egress log and its summary, the bucket's limits |
| `client/storage`, `client/cdn`, `client/functions`, `client/inference` | Later. The `storage` and `agents` modules of this repository are the lower-level clients of storage nodes and of the V8 functions runtime |

It follows the same API contract, names and defaults as the `cere` CLI and the Python SDK (`pip install cere`), and reads the same login.

## Quickstart

```go
c, err := client.NewFromEnv() // what `cere login` saved, or CERE_TOKEN and CERE_API_URL
sbx := sandbox.New(c)

sb, err := sbx.Create(ctx, sandbox.CreateRequest{
	Image:     "python:3.12-slim",
	Timeout:   10 * time.Minute,
	Resources: sandbox.Resources{CPU: 0.25, Memory: 256, CPULimit: 1, MemoryLimit: 512},
	Egress:    []sandbox.Allow{sandbox.MustParseAllow("pypi.org:GET:/simple/")},
	Labels:    map[string]string{"user": "fred"},
})
defer sb.Terminate(ctx, sandbox.TerminateOptions{})

res, err := sb.Exec(ctx, sandbox.ExecRequest{Command: []string{"python", "-c", "print(1 + 1)"}})
fmt.Print(res.Stdout) // 2
```

`examples/quickstart` runs a sandbox end to end:

```
CERE_TOKEN=<token> go run github.com/cerebellum-network/cere-ddc-sdk-go/client/examples/quickstart@latest
```

`examples/detached-exec` starts a command that outlives its caller and, run again after a stop, resumes its output from the byte offsets it saved; `examples/events` follows the bucket's events with a cursor kept in a file.

The default API is the devnet cluster's, `https://agent-gateway.compute.dev.ddcdragon.com` (`client.WithBaseURL` or `CERE_API_URL` to change it).

## Access tokens

Every call carries a DDC access token (see the `auth` module) with `COMPUTE` on the bucket that pays.

| | How | When |
|---|---|---|
| A static token | `client.WithToken(tok)`, or `client.NewFromEnv()` for what `cere login` saved or `CERE_TOKEN` | a person, CI, a script. A token from the developer console lives 1, 7 or 30 days and cannot be revoked |
| A delegation | the bucket owner signs a grant to the service's key once (`client.NewGrant`, or their wallet); the service uses `client.ParseDelegation(grant, auth.NewEd25519Signer(key))` as the token source | a service acting for a bucket it does not own. The owner's key never reaches the service; each call gets a token minted for 10 minutes |

A create needs a token that is valid when the node checks it; after that the sandbox runs until its `Timeout` (none with `sandbox.NoTimeout`) or another end, and the bucket owner's balance, not the token, stops it early. A service that acts for many buckets makes one client and calls `c.WithToken(runToken)` (or `WithTokenSource`) per run.

```go
d, err := client.ParseDelegation(grant, auth.NewEd25519Signer(serviceKey),
	client.WithSubject(runKey)) // optional: names a key in every minted token, for attribution
c, err := client.New(client.WithTokenSource(d))
```

## Sandboxes

| Do | Call |
|---|---|
| create, wait until it runs | `sbx.Create(ctx, req)`; `sandbox.NoWait()` to return at `starting`; `Timeout` zero is the default (5 min), `sandbox.NoTimeout` none. With `IdempotencyKey` a repeat creates nothing and answers the first sandbox (`sb.Replayed()`) |
| a private image | `sbx.SetSecret(ctx, sandbox.RegistrySecret("ghcr-pull", user, token, "ghcr.io"))`, then `CreateRequest.RegistryCredential: "ghcr-pull"`; used for the pull only |
| find | `sbx.Get(ctx, id)`, `sbx.List(ctx, sandbox.ListOptions{Labels: ..., Status: ...})` |
| run a command | `sb.Exec(ctx, req)` (output whole, each stream up to 16 MiB), `sb.ExecStream(ctx, req, stdout, stderr)`; `Timeout` zero is the API's default (1 h), `sandbox.NoTimeout` none |
| run one that outlives its caller | `sb.ExecDetach(ctx, req)` (with `IdempotencyKey`, a repeat starts nothing); `ex.Wait`, `ex.Output(ctx, from)`, `ex.Stream(ctx, from, stdout, stderr)` or `ex.Follow(ctx, from, fn)` from byte offsets; `sb.Execs`, `sb.GetExec` |
| files | `sb.WriteFile`, `sb.ReadFile` (up to 16 MiB) |
| keep files after the end | `CreateRequest.Retain{Paths, On, TTLS}`: kept whatever ends the sandbox (`On: sandbox.RetainOnFailure` for failures only), listed in `sb.Retained` and `sb.NotRetained`, read with `sb.ReadFile` until `sb.RetainedUntil` |
| the main process's output | `sb.Logs(ctx, stdout, stderr)` |
| egress | `sb.EgressRules`, `sb.SetEgress` (replace all), `sb.Allow` (add), `sb.RemoveEgress`; `sbx.EgressLog`, `sbx.EgressSummary` (counts per host and credential) |
| secrets | the sandbox's own: `CreateRequest.Secrets`, `sb.SetSecret` mid-run; an agent's: `sbx.SetAgentSecret`; the bucket's: `sbx.SetSecret`; `sbx.ListSecrets`, `sbx.DeleteSecret` |
| end it | `sb.Terminate(ctx, sandbox.TerminateOptions{Retain: [...], Reason: ...})`; `Retain` adds to the create's; `sb.ExitKind` classes the end (`sandbox.ExitKindMainExited`, `ExitKindTimeout`, `ExitKindOOM`, ...) |
| learn what happened | `sbx.Events(ctx, q)` (long-poll; returns the cursor to poll from next), `sbx.StreamEvents(ctx, q, fn)` (reconnects with `Last-Event-ID`, with backoff, for as long as `ctx` lives); `sandbox.running`, `sandbox.exited`, `exec.exited` |
| have it pushed | `sbx.CreateWebhook`, `sbx.ListWebhooks`, `sbx.DeleteWebhook`, `sbx.WebhookDeliveries`; `sandbox.VerifyWebhook` in the receiving handler |
| usage | `sb.CurrentUsage`, `sb.Usage` (last reported, final after the end), `sbx.UsageRecords` (each with its sandbox's labels), `sbx.UsageSummary(ctx, q)` (summed by the API for a label), `sbx.UsageByLabel(ctx, "user", q)`, `sandbox.BillingState` |
| limits | `sbx.Limits(ctx)`: the most sandboxes the bucket may have active, and how many it has |

**Resources.** `CPU` (vCPU) and `Memory` (MiB) are the request: guaranteed, counted for placement, and billed as a floor while the sandbox runs (default 0.25 vCPU and 128 MiB). `CPULimit` and `MemoryLimit` are the most it can use (default: the node's sandbox CPU up to 8, and 1024 MiB or the request if more); use above the request is billed as used, and a sandbox that reaches its memory limit ends with `oom_killed`. `Disk` (MiB) is the sandbox's writable disk: default 512, no fixed maximum; a write past it fails with `ENOSPC` inside the sandbox, which keeps running (`/tmp` is in memory and counts against `MemoryLimit`).

**Egress.** A sandbox reaches only the hosts its `Allow` rules list. A rule may name a credential in one of two ways: `Credential` (the egress proxy sets that secret's header on every request the rule allows) or `Placeholders` (the sandbox writes `sandbox.Placeholder("slack-a")`, `cere:slack-a`, in a header, and the proxy swaps in that secret; this tells two accounts on one host apart). The sandbox never sees a value. A name resolves to the sandbox's own secret, then its agent's, then the bucket's, and the secret must be bound to the rule's host.

**Labels** select sandboxes, their events and their usage records. An application that runs sandboxes for its own users labels each one (`user=<id>`) and splits the bucket's usage with `UsageByLabel`, or sums one label's with `UsageSummary`. The `cere.` prefix is reserved.

**Detached exec.** A detached command runs whoever is connected; its node keeps the last 4 MiB of each stream, for the sandbox's life and 10 minutes after. Keep the sandbox and exec IDs and the offsets you got to, and read on from them after a restart: nothing is sent twice.

```go
ex, err := sb.ExecDetach(ctx, sandbox.ExecRequest{Command: []string{"python", "run.py"}, IdempotencyKey: "task:t-81:1"})
// ...later, perhaps in another process:
ex, err = sb.GetExec(ctx, execID)
exit, err := ex.Follow(ctx, saved, func(ev sandbox.OutputEvent) error {
	os.Stdout.WriteString(ev.Data)
	saved = ev.Next // where to go on from
	return nil
})
```

**Events.** Events are kept 72 hours. Keep the ID of the last event handled and pass it as `After`; a cursor older than that answers `sandbox.CodeCursorExpired`. Delivery is at least once: dedupe on the ID.

```go
err := sbx.StreamEvents(ctx, sandbox.EventQuery{After: cursor, Types: []string{sandbox.EventSandboxExited},
	Labels: map[string]string{"task": "t-81"}}, func(ev sandbox.Event) error {
	fmt.Println(ev.SandboxID, ev.ExitKind, ev.Retained)
	cursor = ev.ID
	return nil
})
```

A webhook gets the same events, one signed `POST` each; check each with `sandbox.VerifyWebhook(secret, r.Header, body, 0)` before trusting its body.

## Errors and retries

An answer outside 2xx is a `*client.Error` with the status and the API's stable code; match on those, never on the message:

```go
if client.IsCode(err, sandbox.CodeNotRunning) { ... }  // 409: the sandbox has ended
if client.IsCode(err, sandbox.CodeFileGone) { ... }    // 410: not retained, or no longer kept
if client.IsCode(err, sandbox.CodeCursorExpired) { ... } // 410: an events cursor older than 72 hours
if client.IsCode(err, sandbox.CodeIdempotencyConflict) { ... } // 409: the key was used with another body
if client.IsCode(err, client.CodeInsufficientBalance) { ... } // the bucket's owner must top up
if client.StatusOf(err) == http.StatusForbidden { ... }          // otherwise: not this bucket's token
```

A call is retried (3 tries, backoff with jitter, `Retry-After` honoured) only when that cannot make it take effect twice: reads and replacements (GET, PUT, DELETE), and a create or detached exec with an `IdempotencyKey`, on 429, 502, 503, 504 and transport errors; any call on 429, 503 `AUTH_UNAVAILABLE`, or a connection that was never made; a create also on 503 `no_capacity`, which creates nothing. An exec without a key is never repeated. `client.WithRetry(client.NoRetry)` turns retries off.

## Tests

`go test ./...` runs against an API in memory. `sandbox/live_test.go` runs the calls against a real cluster (`TestLive`: a sandbox's calls, detached exec included; `TestLiveRetainAndEvents`: files kept after the command exits, events by label, usage summary, limits, webhooks):

```
CERE_TOKEN_FILE=<file> go test -tags live -run Live -v ./sandbox/
CERE_GRANT_FILE=<grant> CERE_KEY_FILE=<seed hex> go test -tags live -run LiveDelegation -v ./sandbox/
```

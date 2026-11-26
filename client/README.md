# client

```
go get github.com/cerebellum-network/cere-ddc-sdk-go/client
```

The Go client of Cere's developer platform: one client for every service the platform serves. Sandboxes are the first; storage, CDN, functions and inference come later as packages beside them, on the same client.

| Package | What |
|---|---|
| `client` | The base: the API's URL, access tokens (a static token, or tokens minted per call from a delegation), errors, timeouts, retries, paged listings and event streams |
| `client/sandbox` | gVisor sandboxes: create with a request and limits, exec (whole or streamed), files, logs, egress rules (read, replace, add, remove), secrets in three scopes, replacing a secret while it runs, terminate keeping files, usage and usage by label, the egress log |
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

The default API is the devnet cluster's, `https://agent-gateway.compute.dev.ddcdragon.com` (`client.WithBaseURL` or `CERE_API_URL` to change it).

## Access tokens

Every call carries a DDC access token (see the `auth` module) with `COMPUTE` on the bucket that pays.

| | How | When |
|---|---|---|
| A static token | `client.WithToken(tok)`, or `client.NewFromEnv()` for what `cere login` saved or `CERE_TOKEN` | a person, CI, a script. A token from the developer console lives 1, 7 or 30 days and cannot be revoked |
| A delegation | the bucket owner signs a grant to the service's key once (`client.NewGrant`, or their wallet); the service uses `client.ParseDelegation(grant, auth.NewEd25519Signer(key))` as the token source | a service acting for a bucket it does not own. The owner's key never reaches the service; each call gets a token minted for 10 minutes |

A create needs a token that is valid when the node checks it; after that the sandbox runs until its `Timeout` (none with `sandbox.NoTimeout`) or another end, and the bucket owner's balance, not the token, stops it early. Clusters before the lifetime change of 26 Nov 2026 also end a sandbox 60 s before its create token expires (exit reason `token_expiry`), so `Create` asks the token source for a token that outlives the sandbox's `Timeout` and that margin: a delegation mints one (harmless where the token no longer bounds the sandbox); a static token is sent as it is. A service that acts for many buckets makes one client and calls `c.WithToken(runToken)` (or `WithTokenSource`) per run.

```go
d, err := client.ParseDelegation(grant, auth.NewEd25519Signer(serviceKey),
	client.WithSubject(runKey)) // optional: names a key in every minted token, for attribution
c, err := client.New(client.WithTokenSource(d))
```

## Sandboxes

| Do | Call |
|---|---|
| create, wait until it runs | `sbx.Create(ctx, req)`; `sandbox.NoWait()` to return at `starting`; `Timeout` zero is the default (10 min), `sandbox.NoTimeout` none |
| find | `sbx.Get(ctx, id)`, `sbx.List(ctx, sandbox.ListOptions{Labels: ..., Status: ...})` |
| run a command | `sb.Exec(ctx, req)` (output whole, each stream up to 16 MiB), `sb.ExecStream(ctx, req, stdout, stderr)`; `Timeout` zero is the API's default (1 h), `sandbox.NoTimeout` none |
| files | `sb.WriteFile`, `sb.ReadFile` (up to 16 MiB) |
| the main process's output | `sb.Logs(ctx, stdout, stderr)` |
| egress | `sb.EgressRules`, `sb.SetEgress` (replace all), `sb.Allow` (add), `sb.RemoveEgress`; `sbx.EgressLog` |
| secrets | the sandbox's own: `CreateRequest.Secrets`, `sb.SetSecret` mid-run; an agent's: `sbx.SetAgentSecret`; the bucket's: `sbx.SetSecret`; `sbx.ListSecrets`, `sbx.DeleteSecret` |
| end it | `sb.Terminate(ctx, sandbox.TerminateOptions{Retain: [...], Reason: ...})`; retained files stay readable 24 h |
| usage | `sb.CurrentUsage`, `sb.Usage` (last reported, final after the end), `sbx.UsageRecords`, `sbx.UsageByLabel(ctx, "user", q)`, `sandbox.BillingState` |

**Resources.** `CPU` (vCPU) and `Memory` (MiB) are the request: guaranteed, counted for placement, and billed as a floor while the sandbox runs (default 0.25 vCPU and 128 MiB). `CPULimit` and `MemoryLimit` are the most it can use (default: the node's sandbox CPU up to 8, and 1024 MiB or the request if more); use above the request is billed as used, and a sandbox that reaches its memory limit ends with `oom_killed`.

**Egress.** A sandbox reaches only the hosts its `Allow` rules list. A rule may name a credential in one of two ways: `Credential` (the egress proxy sets that secret's header on every request the rule allows) or `Placeholders` (the sandbox writes `sandbox.Placeholder("slack-a")`, `cere:slack-a`, in a header, and the proxy swaps in that secret; this tells two accounts on one host apart). The sandbox never sees a value. A name resolves to the sandbox's own secret, then its agent's, then the bucket's, and the secret must be bound to the rule's host.

**Labels** select sandboxes and their usage records. An application that runs sandboxes for its own users labels each one (`user=<id>`) and splits the bucket's usage with `UsageByLabel`. The `cere.` prefix is reserved.

## Errors and retries

An answer outside 2xx is a `*client.Error` with the status and the API's stable code; match on those, never on the message:

```go
if client.IsCode(err, sandbox.CodeNotRunning) { ... }  // 409: the sandbox has ended
if client.IsCode(err, sandbox.CodeFileGone) { ... }    // 410: not retained, or kept more than 24 h
if client.StatusOf(err) == http.StatusForbidden { ... } // not this bucket's token, or no balance
```

A call is retried (3 tries, backoff with jitter, `Retry-After` honoured) only when that cannot make it take effect twice: reads and replacements (GET, PUT, DELETE) on 429, 502, 503, 504 and transport errors; any call on 429, 503 `AUTH_UNAVAILABLE`, or a connection that was never made; a create also on 503 `no_capacity`, which creates nothing. An exec is never repeated. `client.WithRetry(client.NoRetry)` turns retries off.

## Tests

`go test ./...` runs against an API in memory. `sandbox/live_test.go` runs every call against a real cluster:

```
CERE_TOKEN_FILE=<file> go test -tags live -run Live -v ./sandbox/
CERE_GRANT_FILE=<grant> CERE_KEY_FILE=<seed hex> go test -tags live -run LiveDelegation -v ./sandbox/
```

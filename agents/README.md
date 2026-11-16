# agents

```
go get github.com/cerebellum-network/cere-ddc-sdk-go/agents
```

Run agents on a DDC cluster, and read their logs and resource use. Calls go to the cluster’s agent endpoint (its agent gateway), never to a runtime directly.

An agent is a JavaScript bundle stored in DDC. `Invoke` runs one of its handlers on an agent runtime the gateway picks, in a sandboxed V8 isolate. Every call carries a DDC access token (see the `auth` module) for the bucket that pays.

## Run a handler

```go
client := agents.NewClient("https://agents.example.com", 30,
	agents.WithToken(encodedToken), // auth.EncodeToken; or WithTokenSource
	agents.WithInvokeTimeout(10*time.Minute))

resp, err := client.Invoke(ctx, agents.InvokeRequest{
	Bundle:  agents.BundleRef{BucketID: "1337", CID: bundleCID},
	Handler: "onMessage",
	Event:   json.RawMessage(`{"text":"hello"}`),
	Limits:  &agents.InvokeLimits{TimeoutMs: 60_000},
})
```

A service that runs agents for many buckets shares one client and passes each run's own token: `client.Invoke(ctx, req, agents.CallToken(runToken))`. A call with no token returns `ErrNoToken` without reaching the gateway.

- **`err` is not nil** for a transport failure, or for a non-200 answer as a `*agents.StatusError` (401 no or bad token, 403 not the bucket owner, 408 timeout, 503 no runtime available).
- **`resp.Status == "error"`** means the handler ran and failed. `resp.Error.Code` is one of the `Code*` constants and `resp.Error.Retryable` says whether retrying can help.
- **Timeouts:** `Invoke` is bounded by `ctx`, not by the per-request timeout, so give `ctx` a deadline longer than `Limits.TimeoutMs`.

## Logs, metrics, health

```go
logs, err := client.QueryLogs(ctx, agents.LogQuery{
	BucketID: "1337", CID: bundleCID, Since: from, Until: to,
	Labels: map[string]string{"runId": runID}, // labels you invoked with
})
usage, err := client.QueryMetrics(ctx, agents.MetricsQuery{
	Bucket: "1337", CID: bundleCID, Metric: "agent_invocations_total",
})
err = client.Health(ctx)
```

# Changelog of the client module

Releases are tagged `client/vX.Y.Z`. Nothing has shipped widely yet, so a release may rename or remove what an earlier
one had, as the sandbox API does.

## client/v0.1.0-alpha.6

Fixes from review: streams that are cut, stall or keep ending; tokens that print; the API's codes and fields.

### Fixed

- A stream cut in the middle of a line drops the incomplete event and is a transport error, so `Follow`,
  `StreamEvents` and `Exec.Follow` reconnect from the last complete event; it was handed over as partial JSON.
- A stream's idle limit covers the wait for the answer's headers: a stalled API no longer blocks `StreamEvents`,
  `Logs` or `ExecStream` for ever under `context.Background()`.
- A stream that has started is never reopened, whatever `Request.Accept` the caller set.
- `Follow` waits at least the retry policy's `MinBackoff` between connections, also after one that delivered events
  (a server that sent an event and ended, or an error event, was reopened in a tight loop).
- A context that ends during a retry's backoff returns an error that wraps `ctx.Err()` and the last try's answer.
- `client.List` and `UsageSummary` return an error when the API repeats a cursor, instead of looping.
- A refused proxy connection counts as never sent (a `POST` is retried); a failure on a redirect's next hop does
  not (the first server had the request).
- `Exec` reads any answer the API may send: its cap allows 16 MiB of each stream at six bytes per byte of JSON.
- `VerifyWebhook` reads signatures split over several `webhook-signature` header lines.
- An `ExecRequest.Stdin` that is not valid UTF-8 is an error before any call; it was rewritten to U+FFFD.
- A `Sandbox` or `Exec` decoded from JSON returns `sandbox.ErrNoService` from its calls instead of panicking.

### Changed

- `Follow` reconnects for as long as `ctx` lives, with backoff capped at `MaxBackoff`; it gave up after five
  failed connections (about 15 s of API outage). `StreamEvents` and `Exec.Follow` say so; bound them with `ctx`.
  An answer that is not an event stream at all is not reconnected to.
- `Config.Token` is a `StaticToken`, not a `string` (`string(cfg.Token)` is the token). `StaticToken` never prints:
  `fmt` with any verb, `slog` and `encoding/json` show `StaticTokenRedacted`.
- `Wait`, `WaitRunning` and `Exec.Wait` still poll through answers of 500 and above until `ctx` ends, now documented;
  the error at `ctx`'s end wraps the last such answer.
- `UsageSums.CPUAvg` (the summary's totals carry it too); `SandboxUsage.CPUAvg` moved there.

### Added

- `client.CodeInsufficientBalance` (403 `insufficient_balance`: the bucket's owner must top up); `CodeForbidden` is
  only a token for another bucket.
- `ExitStatus.Error`: why a command failed to run (exit code -1). A command at its timeout is exit code 124.
- `UsageQuery.DACStatus` (the records' `dac_status` filter); JSON names on `LabelUsage`.

## client/v0.1.0-alpha.5

### Fixed

- `sandbox.ParseAllow` reads a credential part of placeholders, `host:POST:/openai/v1:cere:ddc-inference` or
  `host:*::cere:a,cere:b`, as `Allow.Placeholders`, as the `cere` CLI does. It was read as `Credential`
  `"cere:ddc-inference"`, which the API refuses (`credential "cere:ddc-inference" is not a valid secret name`).

### Changed

- Label filters are sent as the API now takes them, `label=key=value` (`ListOptions.Labels`, `EventQuery.Labels`,
  usage records and summary). `WebhookRequest.Label` and `Webhook.Label` are `"key=value"`; `"key:value"` is refused.

## client/v0.1.0-alpha.4

The sandbox API's events, detached exec, retained files, private images, summaries and limits.

### Added

- `client`: `Request.Header` and `Request.IdempotencyKey` (sent as `Idempotency-Key`; a POST with a key is retried as
  a PUT is); `DoResponse` returns the answer's status and headers, and `Response.Replayed` reads
  `Idempotent-Replayed`; `Event.ID` carries the stream's last event id; `Follow` reads a resumable stream, opening it
  again whenever a connection ends, drops or goes silent.
- `sandbox`: idempotent creates (`CreateRequest.IdempotencyKey`, `Sandbox.Replayed`, `CodeIdempotencyConflict`).
- `sandbox`: files kept whatever ends a sandbox (`CreateRequest.Retain` with `Paths`, `On`, `TTLS`; `RetainOnAny`,
  `RetainOnFailure`); after the end `Sandbox.Retained`, `NotRetained` and `RetainedUntil`.
- `sandbox`: private images: `Secret.Type` and `Username`, `RegistrySecret`, `CreateRequest.RegistryCredential`;
  listings show a registry secret's username.
- `sandbox`: detached exec: `Sandbox.ExecDetach` (`ExecRequest.IdempotencyKey`), `Execs`, `GetExec`; `Exec.Refresh`,
  `Wait`, `Output` (from byte offsets), `Follow` and `Stream` (from byte offsets, resumed after a dropped
  connection); `CodeExecNotFound`, `CodeExecGone`.
- `sandbox`: lifecycle events: `Service.Events` (long-poll, with the next cursor) and `Service.StreamEvents` (SSE,
  reconnecting with `Last-Event-ID`); `Event`, `EventQuery`, `FromOldest`, `CodeCursorExpired`.
- `sandbox`: webhooks: `CreateWebhook`, `ListWebhooks`, `DeleteWebhook`, `WebhookDeliveries`; `VerifyWebhook` checks
  a delivery's signature and timestamp; `CodeLimitWebhooks`.
- `sandbox`: `Service.UsageSummary` (usage summed per sandbox and in all, by label), `Service.EgressSummary`,
  `Service.Limits`.
- `sandbox`: `Sandbox.ReadyAt`, `Sandbox.ExitKind` (`ExitKind*`), `Sandbox.RegistryCredential`,
  `Sandbox.IdempotencyKey`, `Usage.CPUAvg`; `UsageRecord.Labels`.
- Examples: `examples/detached-exec` (resumes from saved offsets), `examples/events` (a cursor kept in a file).

### Changed

- `Sandbox.Image` is an `Image` (`Ref`, `Digest`, `SizeBytes`, `Cached`, `PullMS`), not a string.
- `Sandbox.Retained` is `[]RetainedFile` (`Path`, `Size`), not `[]string`.
- `Create` and `WaitRunning` return a sandbox that ran and has already ended without a `StartError`; only one that
  never started is one.
- `UsageByLabel` groups the records by their own labels instead of listing the sandboxes first.

## client/v0.1.0-alpha.3

- `Resources.Disk` (MiB, default 512); the default sandbox timeout is 5 minutes; the API's names that never shipped
  are gone. A create's token only has to be valid at create.

## client/v0.1.0-alpha.2

- `sandbox.NoTimeout` for a sandbox or an exec with no timeout; a static token is sent as it is; exec and log streams
  give up after 2 minutes of silence, not after the command's timeout.

## client/v0.1.0-alpha.1

- The module: one client (base URL, access tokens and delegations, errors, retries, paged listings, event streams,
  the `cere login` config) and `client/sandbox` (create, exec, files, logs, egress, secrets in three scopes,
  terminate keeping files, usage, the egress log).

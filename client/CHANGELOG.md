# Changelog of the client module

Releases are tagged `client/vX.Y.Z`. Nothing has shipped widely yet, so a release may rename or remove what an earlier
one had, as the sandbox API does.

## client/v0.1.0-alpha.4 (unreleased)

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

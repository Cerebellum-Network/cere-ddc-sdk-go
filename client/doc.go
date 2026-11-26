// Package client is the Go client of Cere's developer platform: one client for
// every service the platform serves, with sandboxes as the first.
//
// A Client holds what every service shares: the API's base URL, the access
// token (a static token, or tokens minted per call from a delegation), the API's
// errors, timeouts and retries. Each service is a package beside this one that
// takes a Client:
//
//	c, err := client.New(client.WithToken(os.Getenv("CERE_TOKEN")))
//	if err != nil { ... }
//	sbx := sandbox.New(c)
//	sb, err := sbx.Create(ctx, sandbox.CreateRequest{Image: "python:3.12-slim"})
//
// Services:
//
//   - client/sandbox: gVisor sandboxes (create, exec, files, logs, egress,
//     secrets, usage). Available.
//   - client/storage, client/cdn, client/functions (the V8 engine),
//     client/inference: later, as packages beside sandbox on the same Client.
//
// # Access tokens
//
// Every call carries a DDC access token (see the auth module) for the bucket
// that pays, as "Authorization: Bearer <token>". A TokenSource gives it:
//
//   - StaticToken: one token, for example from the developer console's Access
//     tokens page or what `cere login` saved (LoadConfig, NewFromEnv).
//   - Delegation: the bucket owner signs a grant to the service's own key once
//     (NewGrant); the service then mints a short token beneath it for each call,
//     and one for each sandbox that lives exactly as long as the sandbox needs.
//
// A sandbox create needs a token that is valid when the node checks it; the
// owner's balance, not the token, bounds the sandbox's life.
//
// # Errors and retries
//
// An answer outside 2xx is an *Error with the HTTP status and the API's stable
// code ("not_running", "no_capacity", ...): match on those with IsCode and
// StatusOf, never on the message. Calls that cannot have taken effect twice are
// retried with backoff (see RetryPolicy): reads and replacements (GET, PUT,
// DELETE) and calls with an Idempotency-Key (Request.IdempotencyKey) on 429,
// 502, 503 and 504 and on transport errors, and any call the API refused
// before doing anything (429, 503 AUTH_UNAVAILABLE, a connection that was
// never made). DoResponse also returns the answer's headers; Response.Replayed
// says the API answered a repeated key with the first call's result.
//
// # Streams
//
// Stream reads one event stream. Follow reads one that can be resumed (an
// event feed with Last-Event-ID, a command's output from byte offsets): it
// opens it again, from where its caller got to, whenever a connection ends,
// drops or goes silent.
package client

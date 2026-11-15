// Package agentgateway is the client and wire contract of the DDC agent
// gateway, the front door for running agents on a DDC cluster.
//
// An agent is a JavaScript bundle stored in DDC. POST /v1/invoke runs one of its
// handlers on an agent runtime in the cluster, in a sandboxed V8 isolate, and
// returns the result. The gateway picks the runtime; callers never address a
// runtime directly. Every call carries a DDC access token for the bucket that
// pays for it (see the auth module).
//
// The gateway also serves an agent bundle's logs (GET /v1/logs) and approximate
// resource use (GET /v1/metrics), both scoped to the bucket the caller owns.
package agentgateway

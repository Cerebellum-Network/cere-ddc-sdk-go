# cere-ddc-sdk-go

The Cere DDC SDK for Go. Each folder below is its own Go module, released with tags `<folder>/vX.Y.Z`.

| Module | What |
|---|---|
| [`client`](client) | The client of Cere's developer platform: one client (base URL, access tokens, errors, retries) with a package per service. Sandboxes first (`client/sandbox`) |
| [`auth`](auth) | Mint, encode and verify DDC access tokens |
| [`indexer`](indexer) | On-chain DDC customer state without a chain RPC connection: bucket owners, balance, delegates |
| [`storage`](storage) | Content on a DDC storage node over its gRPC API: pieces, DAG nodes, CNS names |
| [`agents`](agents) | Run functions on the V8 agent runtime through a cluster's agent gateway, and read their logs and metrics |
| `blockchain`, `core`, `contract` | The older chain and contract clients |

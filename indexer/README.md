# indexer

```
go get github.com/cerebellum-network/cere-ddc-sdk-go/indexer
```

Read on-chain DDC customer state without a chain RPC connection: who owns a bucket, whether the owner has balance, whether the bucket was removed, which proxy delegates the owner has granted, and which buckets an account owns.

The client speaks the blockchain index contract, a small versioned read-only HTTP API (`/v1/…`), which DDC storage nodes serve today. You configure one base URL.

`*indexer.Client` implements `auth.BucketOwnerIndex`, so it is what an `auth.Authorizer` checks token chains against.

## Authorize requests against the chain

```go
index, err := indexer.New("https://storage.example.com")
if err != nil {
	return err // a bad URL or option fails here, at startup
}
authorizer := auth.New(index)
verified, err := authorizer.Authorize(ctx, encodedToken, pb.Operation_GET, bucketID)
```

## Read state directly

```go
info, err := index.LookupBucket(ctx, bucketID)             // owner, balance, removed, delegates
delegates, err := index.LookupDelegates(ctx, ownerPubKey)  // the owner's honored proxies
page, err := index.ListBuckets(ctx, ownerPubKey, indexer.BucketsQuery{After: 0, Limit: 100})
```

## Behaviour

- **Fails closed.** When the index cannot be reached and no recent enough answer is cached, lookups return an error, never a permissive default. An outage is never reported as "no delegates" or "no buckets".
- **Caches** each bucket and delegates answer in a bounded LRU for 15 seconds (the balance TTL). When a refresh fails, it serves the cached answer for up to 2 more minutes (the grace window), never once the answer is an hour old (the ownership TTL). Concurrent lookups of the same key share one request. Owner listings are not cached.
- **Tunable** with `WithTTLs`, `WithGraceWindow`, `WithRequestTimeout`, `WithCacheSize`, `WithHTTPClient` and `WithClock`.
- **Server side:** `contract.go` holds the wire types (`BucketResponse`, `BucketsResponse`, `DelegatesResponse`, `ErrorResponse`) and paths, for implementations of the contract.

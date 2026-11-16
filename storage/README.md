# storage

```
go get github.com/cerebellum-network/cere-ddc-sdk-go/storage
```

Store and read content on a DDC storage node over its gRPC API: raw pieces, DAG nodes, and CNS names (signed, mutable pointers from a name in a bucket to a CID).

| Package | Contents |
|---|---|
| `storage` | `Client`: `PutPiece`, `PutNode`, `GetNode`, `PutName`, `ResolveName`; `CIDBytes` / `CIDString` |
| `storage/fileapi`, `dagapi`, `cnsapi` | The storage node's generated gRPC clients, servers and messages |
| `storage/cid` | DDC CIDs (v2): decode, encode, validate, compute (`Sum`) |
| `storage/activity` | DDC activity messages, including `ActivityAcknowledgment` used by the file API's read stream |
| `storage/storagetest` | An in-process storage node for tests, which authorizes requests like a real one |

## Publish content under a name

```go
conn, err := grpc.NewClient("storage.example.com:443", grpc.WithTransportCredentials(credentials.NewTLS(nil)))
client := storage.NewClient(conn,
	storage.WithToken(encodedToken),                    // auth.EncodeToken: PUT and GET on the bucket
	storage.WithRecordSigner(auth.NewEd25519Signer(key)), // signs CNS records
)

pieceCID, err := client.PutPiece(ctx, bucketID, file, size) // up to 128 MiB, size must be exact
pieceBytes, err := storage.CIDBytes(pieceCID)
nodeCID, err := client.PutNode(ctx, bucketID, &dagapi.Node{
	Links: []*dagapi.Link{{Cid: pieceBytes, Name: "index.html", Size: uint64(size)}},
})
err = client.PutName(ctx, bucketID, "site", nodeCID)

// Later, anywhere:
cid, err := client.ResolveName(ctx, bucketID, "site") // errors.Is(err, storage.ErrNameNotFound) if unset
node, err := client.GetNode(ctx, bucketID, cid)
```

CIDs cross `Client` as base32 strings, the form DDC CDN URLs use. For a token that expires, use `WithTokenSource` to supply a fresh one per call. A service that writes to many buckets shares one client and passes each bucket's token per call: `client.PutNode(ctx, bucketID, node, storage.CallToken(bucketToken))`. A call with no token returns `ErrNoToken` without reaching the node.

## Protobuf

`proto/` holds verbatim copies of the storage node's API protos and of `activity/record.proto` (sources in `proto/SOURCE`). Proto packages, file paths and descriptors are unchanged, so the wire format and gRPC method names are the storage node's. A binary must not link these packages together with another package that registers the same proto files. `generate.sh` regenerates them.

Minimum runtime versions: `google.golang.org/grpc` v1.64 (generated with protoc-gen-go-grpc v1.5.1, `SupportPackageIsVersion9`) and `google.golang.org/protobuf` v1.36.6. The module requires grpc v1.67.1.

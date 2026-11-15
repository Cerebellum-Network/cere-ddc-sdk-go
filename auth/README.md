# auth

```
go get github.com/cerebellum-network/cere-ddc-sdk-go/auth
```

Mint, encode and verify DDC access tokens: the credential every DDC storage node, gateway and compute runtime accepts in the `X-Auth-Token` header.

A token is a chain of signed levels. The root level is signed by a bucket owner and grants operations (`GET`, `PUT`, `DELETE`, `COMPUTE`) on that bucket. Each further level delegates the grant to the subject its parent names. `Authorizer` accepts a token only if its chain roots in the bucket's on-chain owner (or one of the owner's proxy delegates) and the owner has balance.

| Package | Contents |
|---|---|
| `auth` | `SignMessage` and signers (ed25519, sr25519, or any key via `NewSigner`); `EncodeToken` / `DecodeToken`; `VerifySignature`; `Authorizer` with `BucketOwnerIndex`; `BucketID`, `ExpiresAtSeconds`; `AuthTokenHeader` |
| `auth/pb` | The `AuthToken`, `Payload`, `Operation` and `Signature` messages, generated from `proto/` |
| `auth/authtest` | Test fixtures: keys, signed chains, forgeries and canned owner indexes |

## Mint a token

```go
owner := auth.NewEd25519Signer(ownerKey) // or auth.NewSr25519Signer(seed)
bucket := uint64(1337)
expires := time.Now().Add(time.Hour).Unix()

token := &pb.AuthToken{Payload: &pb.Payload{
	BucketId:   &bucket,
	Operations: []pb.Operation{pb.Operation_GET},
	ExpiresAt:  &expires,
}}
if _, err := auth.SignMessage(owner, token); err != nil {
	return err
}
encoded, err := auth.EncodeToken(token)
req.Header.Set(auth.AuthTokenHeader, encoded)
```

## Verify a token

```go
authorizer := auth.New(index) // index is a BucketOwnerIndex, e.g. from the indexer module
verified, err := authorizer.Authorize(ctx, encoded, pb.Operation_GET, bucket)
if err != nil {
	http.Error(w, err.Error(), auth.KindOf(err).HTTPStatus()) // 401, 403 or 503
	return
}
// verified.RootIssuer is the bucket owner (or delegate); verified.Subject is the leaf's subject.
```

## Rules a minter must follow

- Set a future `ExpiresAt` on every level. A missing expiry reads as expired. Seconds and milliseconds are both accepted.
- Keep the operation set identical at every level. The attenuation check is inverted for compatibility with storage nodes (see the package doc).
- A child level may not outlive its parent.

## Protobuf

`proto/` is a verbatim copy of `auth/token.proto` and `signature/signature.proto` from the DDC protocol definitions (commit in `proto/SOURCE`). The proto packages are `auth` and `signature`. A binary must not link this package together with another package that registers the same proto files. `generate.sh` regenerates `pb/`.

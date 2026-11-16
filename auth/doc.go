// Package auth mints, encodes and verifies DDC access tokens.
//
// A DDC access token (pb.AuthToken) is a chain of signed levels. The root level
// is signed by the owner of a DDC bucket and grants operations on that bucket;
// each further level delegates the grant to the subject its parent names. The
// presented token is the leaf. Storage nodes, the gateways and the compute
// runtimes all accept the same token, and all apply one rule:
//
//	the chain must root in the OWNER of the bucket it bills, or in one of that
//	owner's honored proxy delegates, and that owner must have balance.
//
// Checking signatures alone is not enough: anyone can mint a self-signed token
// naming an arbitrary bucket. Authorizer binds the chain's root issuer to the
// bucket owner through a BucketOwnerIndex (the indexer module provides one),
// which is what turns a well-formed token into an authorization.
//
// # Minting
//
//	owner := auth.NewEd25519Signer(ownerKey)
//	expires := time.Now().Add(time.Hour).Unix()
//	bucket := uint64(1337)
//	token := &pb.AuthToken{Payload: &pb.Payload{
//		BucketId:   &bucket,
//		Operations: []pb.Operation{pb.Operation_GET, pb.Operation_PUT},
//		ExpiresAt:  &expires,
//	}}
//	if _, err := auth.SignMessage(owner, token); err != nil { ... }
//	encoded, err := auth.EncodeToken(token) // send as the X-Auth-Token header
//
// # Verifying
//
//	authorizer := auth.New(index)
//	verified, err := authorizer.Authorize(ctx, encoded, pb.Operation_GET, bucket)
//	if err != nil {
//		http.Error(w, err.Error(), auth.KindOf(err).HTTPStatus())
//		return
//	}
//
// # Differences from the storage node's check
//
// Authorize walks the chain exactly as DDC storage nodes do, and rejects more:
//
//   - A request naming no bucket (bucketID 0) is rejected: there is nothing to
//     bill and no owner to bind.
//   - A chain that names no bucket is rejected.
//   - A piece-scoped token is rejected: this entry point carries no piece to
//     match it against.
//   - Chain depth is bounded (DefaultMaxChainDepth), and so is the encoded
//     length (DefaultMaxEncodedTokenLength): base58 decoding is quadratic, so an
//     unbounded credential is a pre-authentication CPU amplifier.
//   - Every failure is a typed *Error carrying a Kind, so a transport maps
//     401 / 403 / 503 without matching sentinels.
//
// The public-bucket GET bypass and the customer and bucket blacklists belong to
// the storage data path and are not part of this package. Bucket removal
// arrives through LookupResult.Removed.
//
// # Preserved quirks
//
// Four behaviours of the walk are surprising and are kept, because changing them
// changes live storage semantics. Each is pinned by a test. None lets a caller
// reach authority it does not already hold; each is a case where an intent
// expressed by a token author is not enforced.
//
//  1. The attenuation check is inverted: it enforces parent.ops ⊆ child.ops.
//     Keep operation sets identical across levels. See permissionsValid.
//  2. A nil or zero ExpiresAt reads as EXPIRED, not as "no expiry".
//  3. A level with an EMPTY operation set inherits instead of granting
//     nothing.
//  4. CanDelegate is only enforced at depth > 1: a grant minted with
//     canDelegate=false can still have exactly one level signed beneath it.
//     Tokens are bearer credentials with no proof of possession of Subject, so
//     whoever could present the parent could already use it directly.
//
// One deviation from the storage node is a strict rejection: a middle level
// with an empty operation set does not reset the set the walk has accumulated. On the
// node it does, so root {GET, PUT}, then {}, then leaf {GET} authorizes PUT
// there; here that chain is checked as if the empty level were absent.
//
// DecodeToken accepts base58 (DDC-native) and standard base64.
package auth

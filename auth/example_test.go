package auth_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb"
)

// ownerIndex is a one-bucket BucketOwnerIndex. In production, use the indexer
// module, which reads bucket owners and balances from the chain.
type ownerIndex struct {
	bucket uint64
	owner  ed25519.PublicKey
}

func (i ownerIndex) Lookup(_ context.Context, bucket uint64) (auth.LookupResult, error) {
	if bucket != i.bucket {
		return auth.LookupResult{}, nil
	}
	return auth.LookupResult{Owner: i.owner, HasBalance: true}, nil
}

func Example() {
	ownerPub, ownerKey, _ := ed25519.GenerateKey(rand.Reader)
	bucket := uint64(1337)

	// The bucket owner mints a token granting GET on the bucket for an hour.
	expires := time.Now().Add(time.Hour).Unix()
	token := &pb.AuthToken{Payload: &pb.Payload{
		BucketId:   &bucket,
		Operations: []pb.Operation{pb.Operation_GET},
		ExpiresAt:  &expires,
	}}
	if _, err := auth.SignMessage(auth.NewEd25519Signer(ownerKey), token); err != nil {
		panic(err)
	}
	encoded, _ := auth.EncodeToken(token) // sent as the X-Auth-Token header

	// A service checks it against the bucket's on-chain owner.
	authorizer := auth.New(ownerIndex{bucket: bucket, owner: ownerPub})
	if _, err := authorizer.Authorize(context.Background(), encoded, pb.Operation_GET, bucket); err != nil {
		fmt.Println("denied:", err)
	} else {
		fmt.Println("allowed")
	}

	// The token grants GET only.
	_, err := authorizer.Authorize(context.Background(), encoded, pb.Operation_PUT, bucket)
	fmt.Println(auth.KindOf(err).HTTPStatus(), err)

	// Output:
	// allowed
	// 403 forbidden: token permissions are invalid
}

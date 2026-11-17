package indexer_test

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb"
	"github.com/cerebellum-network/cere-ddc-sdk-go/indexer"
)

// The client is the BucketOwnerIndex an auth.Authorizer checks token chains
// against.
func Example() {
	index, err := indexer.New("https://storage.example.com",
		indexer.WithTTLs(time.Hour, 15*time.Second))
	if err != nil {
		panic(err)
	}
	authorizer := auth.New(index)

	var encodedToken string // from the X-Auth-Token header
	if _, err := authorizer.Authorize(context.Background(), encodedToken, pb.Operation_GET, 1337); err != nil {
		fmt.Println(auth.KindOf(err).HTTPStatus())
	}
}

// List the buckets an account owns, a page at a time.
func ExampleClient_ListBuckets() {
	index, _ := indexer.New("https://storage.example.com")
	var owner ed25519.PublicKey // the account's 32-byte public key

	query := indexer.BucketsQuery{}
	for {
		page, err := index.ListBuckets(context.Background(), owner, query)
		if err != nil {
			panic(err)
		}
		for _, bucket := range page.Buckets {
			fmt.Println(bucket.BucketID, bucket.HasBalance)
		}
		if !page.HasMore() {
			break
		}
		query.After = page.NextAfter
	}
}

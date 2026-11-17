package agents_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/agents"
)

func ExampleHTTPClient_Invoke() {
	// A COMPUTE grant on bucket 1337, minted and encoded with the auth module
	// (auth.EncodeToken).
	var token string

	client := agents.NewClient("https://agents.example.com", 30, agents.WithToken(token))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	resp, err := client.Invoke(ctx, agents.InvokeRequest{
		Bundle:  agents.BundleRef{BucketID: "1337", CID: "bafk..."},
		Handler: "onMessage",
		Event:   json.RawMessage(`{"text":"hello"}`),
		Limits:  &agents.InvokeLimits{TimeoutMs: 60_000},
	})
	if err != nil {
		panic(err) // transport failure or a non-200 *agents.StatusError
	}
	if resp.Status == "error" {
		fmt.Println(resp.Error.Code, resp.Error.Message) // the handler ran and failed
		return
	}
	fmt.Println(string(resp.Result))
}

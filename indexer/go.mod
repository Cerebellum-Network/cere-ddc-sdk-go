module github.com/cerebellum-network/cere-ddc-sdk-go/indexer

go 1.24.0

require (
	github.com/cerebellum-network/cere-ddc-sdk-go/auth v0.0.0-20261116143455-8bd376b0b8c6
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/ChainSafe/go-schnorrkel v1.1.0 // indirect
	github.com/cosmos/go-bip39 v0.0.0-20180819234021-555e2067c45d // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gtank/merlin v0.1.1-0.20191105220539-8318aed1a79f // indirect
	github.com/gtank/ristretto255 v0.1.2 // indirect
	github.com/mimoo/StrobeGo v0.0.0-20181016162300-f8f6d4d2b643 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Builds against the auth module in this repository. Consumers ignore this
// replace and use the version required above.
replace github.com/cerebellum-network/cere-ddc-sdk-go/auth => ../auth

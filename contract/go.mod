module github.com/cerebellum-network/cere-ddc-sdk-go/contract

require (
	github.com/centrifuge/go-substrate-rpc-client/v2 v2.1.1-0.20210302023953-904cb0b931a9
	github.com/patractlabs/go-patract v0.2.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
)

require (
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/ethereum/go-ethereum v1.10.1 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/pierrec/xxHash v0.1.5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rs/cors v1.7.0 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	go.uber.org/atomic v1.6.0 // indirect
	go.uber.org/multierr v1.5.0 // indirect
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292 // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/sys v0.0.0-20220227234510-4e6760a101f9 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	honnef.co/go/tools v0.0.1-2020.1.4 // indirect
)

go 1.18

replace github.com/centrifuge/go-substrate-rpc-client/v2 v2.1.1-0.20210302023953-904cb0b931a9 => github.com/Snowfork/go-substrate-rpc-client/v2 v2.0.2-0.20210115165558-f6ad0aceb9bc

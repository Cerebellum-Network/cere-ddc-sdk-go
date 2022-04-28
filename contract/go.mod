module github.com/cerebellum-network/cere-ddc-sdk-go/contract

require (
	github.com/centrifuge/go-substrate-rpc-client/v2 v2.1.1-0.20210302023953-904cb0b931a9
	github.com/patractlabs/go-patract v0.2.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.2.0
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110
)

require (
	github.com/btcsuite/btcutil v1.0.2 // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/ethereum/go-ethereum v1.10.1 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.1 // indirect
	github.com/pierrec/xxHash v0.1.5 // indirect
	github.com/rs/cors v1.7.0 // indirect
	go.uber.org/atomic v1.6.0 // indirect
	go.uber.org/multierr v1.5.0 // indirect
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/crypto v0.0.0-20210317152858-513c2a44f670 // indirect
	golang.org/x/sys v0.0.0-20210317225723-c4fcb01b228e // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
)

go 1.18

replace github.com/centrifuge/go-substrate-rpc-client/v2 v2.1.1-0.20210302023953-904cb0b931a9 => github.com/Snowfork/go-substrate-rpc-client/v2 v2.0.2-0.20210115165558-f6ad0aceb9bc

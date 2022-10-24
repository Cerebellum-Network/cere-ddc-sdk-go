package actcapture

import "github.com/centrifuge/go-substrate-rpc-client/v2/types"

type Commit struct {
	Hash types.Hash
	Gas  types.U128
	From types.U64 //nanoseconds
	To   types.U64 //nanoseconds
}

type EraConfig struct {
	Start    types.U64 // milliseconds
	Interval types.U64 // milliseconds
}

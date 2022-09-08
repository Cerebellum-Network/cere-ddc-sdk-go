package actcapture

import "github.com/centrifuge/go-substrate-rpc-client/v2/types"

type Commit struct {
	Hash      types.Hash
	Resources types.U128
}

type EraConfig struct {
	Start    types.U64 // milliseconds
	Interval types.U64 // milliseconds
}

package pallets

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

const (
	storedHost             = "178.251.228.236"
	storedHttpPort         = uint16(8080)
	storedGrpcPort         = uint16(9090)
	storedP2pPort          = uint16(9070)
	storageNodeModeStorage = byte(2)
)

// storedStorageNodeProps is ddc-nodes' StorageNodeProps as the chain stores
// it once domain and ssl are removed: host, http_port, grpc_port, p2p_port,
// mode.
func storedStorageNodeProps(t *testing.T) []byte {
	t.Helper()
	host, err := codec.Encode(types.NewBytes([]byte(storedHost)))
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	b.Write(host)
	for _, port := range []uint16{storedHttpPort, storedGrpcPort, storedP2pPort} {
		_ = binary.Write(&b, binary.LittleEndian, port)
	}
	b.WriteByte(storageNodeModeStorage)
	return b.Bytes()
}

func TestStorageNodePropsDecodeTheStoredLayout(t *testing.T) {
	var props StorageNodeProps
	if err := codec.Decode(storedStorageNodeProps(t), &props); err != nil {
		t.Fatalf("decode: %v", err)
	}

	host := make([]byte, len(props.Host))
	for i, u := range props.Host {
		host[i] = byte(u)
	}
	if string(host) != storedHost {
		t.Errorf("host %q, want %q", host, storedHost)
	}
	if props.HttpPort != types.U16(storedHttpPort) || props.GrpcPort != types.U16(storedGrpcPort) || props.P2pPort != types.U16(storedP2pPort) {
		t.Errorf("ports %d/%d/%d, want %d/%d/%d", props.HttpPort, props.GrpcPort, props.P2pPort, storedHttpPort, storedGrpcPort, storedP2pPort)
	}
	if !props.Mode.IsStorage {
		t.Errorf("mode %+v, want Storage", props.Mode)
	}
}

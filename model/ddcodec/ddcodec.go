package ddcodec

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func MarshalTyped(m proto.Message, ddtype uint64) ([]byte, error) {
	buf := []byte{0xD0, 0x0C}
	buf = protowire.AppendVarint(buf, uint64(ddtype))

	return proto.MarshalOptions{}.MarshalAppend(buf, m)
}

func UnmarshalTyped(b []byte, m proto.Message, ddtype uint64) error {
	if len(b) >= 3 && b[0] == 0xD0 && b[1] == 0x0C {
		gotType, errCode := protowire.ConsumeVarint(b[2:])
		if errCode < 0 || gotType != ddtype {
			return fmt.Errorf("invalid message type (%d)", gotType)
		}
	}

	return proto.UnmarshalOptions{}.Unmarshal(b, m)
}

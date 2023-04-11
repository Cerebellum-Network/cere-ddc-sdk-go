package topology

import "fmt"

type VNode struct {
	nodeId uint32
	token  uint64
}

func (v VNode) NodeId() uint32 {
	return v.nodeId
}

func (v VNode) Token() uint64 {
	return v.token
}

func (v VNode) String() string {
	return fmt.Sprintf("{\"nodeId\":%d,\"token\":%d}", v.nodeId, v.token)
}

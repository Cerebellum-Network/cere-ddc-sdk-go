package topology

type VNode struct {
	nodeId uint32
	token  uint64
}

func (V *VNode) NodeId() uint32 {
	return V.nodeId
}

func (V *VNode) Token() uint64 {
	return V.token
}

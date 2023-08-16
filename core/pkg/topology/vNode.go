package topology

import "fmt"

type VNode struct {
	nodeKey string
	token   uint64
}

func (v VNode) NodeKey() string {
	return v.nodeKey
}

func (v VNode) Token() uint64 {
	return v.token
}

func (v VNode) String() string {
	return fmt.Sprintf("{\"nodeKey\":%v,\"token\":%d}", v.nodeKey, v.token)
}

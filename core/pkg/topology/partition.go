package topology

import "fmt"

type Partition struct {
	From uint64
	To   uint64
	//NodeId with main replica
	NodeId uint32
}

func (p Partition) Belongs(token uint64) bool {
	isHigherOrEqualFrom := p.From <= token
	isLowerOrEqualTo := token <= p.To

	if p.From <= p.To {
		return isHigherOrEqualFrom && isLowerOrEqualTo
	} else {
		return isHigherOrEqualFrom || isLowerOrEqualTo
	}
}

func (p Partition) String() string {
	return fmt.Sprintf("{\"from\":%d,\"to\":%d,\"nodeId\":%d}", p.From, p.To, p.NodeId)
}

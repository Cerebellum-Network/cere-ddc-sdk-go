package topology

import (
	"sort"
)

type (
	Ring interface {
		Tokens(nodeId uint32) []uint64
		Neighbours(token uint64) (VNode, VNode)
		Replicas(token uint64) []VNode

		Partitions(nodeId uint32) []Partition
		ExcessPartitions(nodeId uint32) []Partition

		RemoveVNode(token uint64) bool

		VNodes() []VNode
		ReplicationFactor() uint
	}

	ring struct {
		vNodes            []VNode
		replicationFactor uint
	}
)

func NewTopology(nodeIds []uint32, vNodes [][]uint64, replicaFactor uint) Ring {
	if replicaFactor == 0 {
		replicaFactor = 1
	}

	topologyVNodes := make([]VNode, 0)
	for i, nodeId := range nodeIds {
		for _, token := range vNodes[i] {
			topologyVNode := VNode{
				nodeId: nodeId,
				token:  token,
			}

			topologyVNodes = append(topologyVNodes, topologyVNode)
		}
	}

	sort.Slice(topologyVNodes, func(i, j int) bool {
		return topologyVNodes[i].token < topologyVNodes[j].token
	})

	return &ring{
		vNodes:            topologyVNodes,
		replicationFactor: replicaFactor,
	}
}

func (r *ring) Tokens(nodeId uint32) []uint64 {
	result := make([]uint64, 0)
	r.vNodeDo(nodeId, func(i int, vNode VNode) {
		result = append(result, vNode.token)
	})

	return result
}

func (r *ring) Replicas(token uint64) []VNode {
	searchIndex := r.search(token)
	if len(r.vNodes) == searchIndex || r.vNodes[searchIndex].token != token {
		searchIndex = r.prevIndex(searchIndex)
	}

	nodes := make([]VNode, 0, r.replicationFactor)
	for i := searchIndex; uint(len(nodes)) < r.replicationFactor; i = r.nextIndex(i) {
		nodes = append(nodes, r.vNodes[i])
	}

	return nodes
}

func (r *ring) Neighbours(token uint64) (prev VNode, next VNode) {
	searchIndex := r.search(token)
	prev = r.vNodes[r.prevIndex(searchIndex)]

	if searchIndex == len(r.vNodes) || r.vNodes[searchIndex].token == token {
		next = r.vNodes[r.nextIndex(searchIndex)]
	} else {
		next = r.vNodes[searchIndex]
	}

	return
}

func (r *ring) Partitions(nodeId uint32) []Partition {
	result := make([]Partition, 0)
	r.vNodeDo(nodeId, func(i int, vNode VNode) {
		for j := uint(1); j < r.replicationFactor; j++ {
			i = r.prevIndex(i)
		}

		for j := uint(0); j < r.replicationFactor; j++ {
			vNode := r.vNodes[i]
			from := vNode.Token()
			i = r.nextIndex(i)
			partition := Partition{From: from, To: r.vNodes[i].Token() - 1, NodeId: vNode.NodeId()}
			result = append(result, partition)
		}
	})

	return result
}

func (r *ring) ExcessPartitions(nodeId uint32) []Partition {
	partitions := r.Partitions(nodeId)

	result := make([]Partition, 0)
	for i := 0; i < len(partitions); i++ {
		j := nextIndex(i, len(partitions))
		if partitions[i].To == partitions[j].From-1 {
			continue
		}

		result = append(result, Partition{From: partitions[i].To + 1, To: partitions[j].From - 1, NodeId: nodeId})
	}

	return result
}

func (r *ring) RemoveVNode(token uint64) bool {
	vNodeId := r.search(token)
	if vNodeId >= len(r.vNodes) || r.vNodes[vNodeId].Token() != token {
		return false
	}

	copy(r.vNodes[vNodeId:], r.vNodes[vNodeId+1:])
	r.vNodes[len(r.vNodes)-1] = VNode{}
	r.vNodes = r.vNodes[:len(r.vNodes)-1]

	return true
}

func (r *ring) VNodes() []VNode {
	return r.vNodes
}

func (r *ring) ReplicationFactor() uint {
	return r.replicationFactor
}

func (r *ring) search(token uint64) int {
	return sort.Search(len(r.vNodes), func(i int) bool { return r.vNodes[i].token >= token })
}

func (r *ring) prevIndex(i int) int {
	return prevIndex(i, len(r.vNodes))
}

func (r *ring) nextIndex(i int) int {
	return nextIndex(i, len(r.vNodes))
}

func (r *ring) vNodeDo(nodeId uint32, do func(int, VNode)) {
	for i, vNode := range r.vNodes {
		if nodeId == vNode.nodeId {
			do(i, vNode)
		}
	}
}

func nextIndex(i int, length int) int {
	i++
	if i >= length {
		return 0
	}

	return i
}

func prevIndex(i int, length int) int {
	i--
	if i < 0 {
		return length - 1
	}

	return i
}

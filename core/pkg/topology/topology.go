package topology

import (
	"sort"
)

type (
	Ring interface {
		Tokens(nodeId uint32) []uint64
		Neighbours(token uint64) (*VNode, *VNode)
		Replicas(token uint64) []*VNode
	}

	ring struct {
		vNodes        []*VNode
		replicaFactor uint
	}
)

func NewTopology(nodeIds []uint32, vNodes [][]uint64, replicaFactor uint) Ring {
	topologyVNodes := make([]*VNode, 0)
	for i, nodeId := range nodeIds {
		for _, token := range vNodes[i] {
			topologyVNode := &VNode{
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
		vNodes:        topologyVNodes,
		replicaFactor: replicaFactor,
	}
}

func (t *ring) Tokens(nodeId uint32) []uint64 {
	result := make([]uint64, 0)
	for _, vNode := range t.vNodes {
		if nodeId == vNode.nodeId {
			result = append(result, vNode.token)
		}
	}

	return result
}

func (t *ring) Replicas(token uint64) []*VNode {
	searchIndex := t.search(token)
	if len(t.vNodes) == searchIndex || t.vNodes[searchIndex].token != token {
		searchIndex = t.prevIndex(searchIndex)
	}

	nodes := make([]*VNode, 0, t.replicaFactor)
	for i := searchIndex; uint(len(nodes)) < t.replicaFactor; i = t.nextIndex(i) {
		nodes = append(nodes, t.vNodes[i])
	}

	return nodes
}

func (t *ring) Neighbours(token uint64) (prev *VNode, next *VNode) {
	searchIndex := t.search(token)
	prev = t.vNodes[t.prevIndex(searchIndex)]

	if searchIndex == len(t.vNodes) || t.vNodes[searchIndex].token == token {
		next = t.vNodes[t.nextIndex(searchIndex)]
	} else {
		next = t.vNodes[searchIndex]
	}

	return
}

func (t *ring) search(token uint64) int {
	return sort.Search(len(t.vNodes), func(i int) bool { return t.vNodes[i].token >= token })
}

func (t *ring) prevIndex(i int) int {
	i--
	if i < 0 {
		return len(t.vNodes) - 1
	}

	return i
}

func (t *ring) nextIndex(i int) int {
	i++
	if i >= len(t.vNodes) {
		return 0
	}

	return i
}

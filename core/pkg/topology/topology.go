package topology

import (
	"sort"
)

type (
	Ring interface {
		Tokens(nodeId uint32) []uint64
		Neighbours(token uint64) (*VNode, *VNode)
		Replicas(token uint64) []*VNode
		VNodes() []*VNode
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

func (r *ring) Tokens(nodeId uint32) []uint64 {
	result := make([]uint64, 0)
	for _, vNode := range r.vNodes {
		if nodeId == vNode.nodeId {
			result = append(result, vNode.token)
		}
	}

	return result
}

func (r *ring) Replicas(token uint64) []*VNode {
	searchIndex := r.search(token)
	if len(r.vNodes) == searchIndex || r.vNodes[searchIndex].token != token {
		searchIndex = r.prevIndex(searchIndex)
	}

	nodes := make([]*VNode, 0, r.replicaFactor)
	for i := searchIndex; uint(len(nodes)) < r.replicaFactor; i = r.nextIndex(i) {
		nodes = append(nodes, r.vNodes[i])
	}

	return nodes
}

func (r *ring) Neighbours(token uint64) (prev *VNode, next *VNode) {
	searchIndex := r.search(token)
	prev = r.vNodes[r.prevIndex(searchIndex)]

	if searchIndex == len(r.vNodes) || r.vNodes[searchIndex].token == token {
		next = r.vNodes[r.nextIndex(searchIndex)]
	} else {
		next = r.vNodes[searchIndex]
	}

	return
}

func (r *ring) VNodes() []*VNode {
	return r.vNodes
}

func (r *ring) search(token uint64) int {
	return sort.Search(len(r.vNodes), func(i int) bool { return r.vNodes[i].token >= token })
}

func (r *ring) prevIndex(i int) int {
	i--
	if i < 0 {
		return len(r.vNodes) - 1
	}

	return i
}

func (r *ring) nextIndex(i int) int {
	i++
	if i >= len(r.vNodes) {
		return 0
	}

	return i
}

package sync

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/core/pkg/topology"
	"sync"
)

type ring struct {
	ring  topology.Ring
	mutex sync.RWMutex
}

func NewTopology(nodeIds []uint32, vNodes [][]uint64, replicaFactor uint) topology.Ring {
	return &ring{
		ring: topology.NewTopology(nodeIds, vNodes, replicaFactor),
	}
}

func (r *ring) Tokens(nodeId uint32) []uint64 {
	r.mutex.RLock()
	result := r.ring.Tokens(nodeId)
	r.mutex.RUnlock()

	return result
}

func (r *ring) Neighbours(token uint64) (topology.VNode, topology.VNode) {
	r.mutex.RLock()
	prev, next := r.ring.Neighbours(token)
	r.mutex.RUnlock()

	return prev, next
}

func (r *ring) Replicas(token uint64) []topology.VNode {
	r.mutex.RLock()
	result := r.ring.Replicas(token)
	r.mutex.RUnlock()

	return result
}

func (r *ring) Partitions(nodeId uint32) []topology.Partition {
	r.mutex.RLock()
	result := r.ring.Partitions(nodeId)
	r.mutex.RUnlock()

	return result
}

func (r *ring) ExcessPartitions(nodeId uint32) []topology.Partition {
	r.mutex.RLock()
	result := r.ring.ExcessPartitions(nodeId)
	r.mutex.RUnlock()

	return result
}

func (r *ring) RemoveVNode(token uint64) bool {
	r.mutex.Lock()
	result := r.ring.RemoveVNode(token)
	r.mutex.Unlock()

	return result
}

func (r *ring) VNodes() []topology.VNode {
	r.mutex.RLock()
	result := r.ring.VNodes()
	r.mutex.RUnlock()

	return result
}

func (r *ring) ReplicationFactor() uint {
	return r.ring.ReplicationFactor()
}

package sync

import (
	"sync"

	"github.com/cerebellum-network/cere-ddc-sdk-go/core/pkg/topology"
)

type ring struct {
	ring  topology.Ring
	mutex sync.RWMutex
}

func NewTopology(nodes topology.NodesVNodes, replicaFactor uint) topology.Ring {
	return &ring{
		ring: topology.NewTopology(nodes, replicaFactor),
	}
}

func (r *ring) Tokens(nodeKey string) []uint64 {
	r.mutex.RLock()
	result := r.ring.Tokens(nodeKey)
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

func (r *ring) Partitions(nodeKey string) []topology.Partition {
	r.mutex.RLock()
	result := r.ring.Partitions(nodeKey)
	r.mutex.RUnlock()

	return result
}

func (r *ring) ExcessPartitions(nodeKey string) []topology.Partition {
	r.mutex.RLock()
	result := r.ring.ExcessPartitions(nodeKey)
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

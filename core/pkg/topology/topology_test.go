package topology

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

var clusters = []struct {
	name          string
	nodeIds       []uint32
	vNodes        [][]uint64
	replicaFactor uint
}{
	{"size 2 replication 3", []uint32{1, 2}, [][]uint64{{9223372036854775806, 3074457345618258602, 15372286728091293010}, {12297829382473034408, 6148914691236517204, 18446744073709551612}}, 2},
	{"size 3 replication 3", []uint32{1, 2, 3}, [][]uint64{{12297829382473034408, 3074457345618258602}, {6148914691236517204, 15372286728091293010}, {18446744073709551612, 9223372036854775806}}, 3},
	{"size 4 replication 3", []uint32{1, 2, 3, 4}, [][]uint64{{12297829382473034408, 3074457345618258602}, {6148914691236517204, 15372286728091293010}, {18446744073709551612, 9223372036854775806}, {4611686018427387903}}, 3},
	{"size 2 replication 1", []uint32{1, 2}, [][]uint64{{9223372036854775806, 3074457345618258602, 15372286728091293010}, {12297829382473034408, 6148914691236517204, 18446744073709551612}}, 1},
	{"size 3 replication 1", []uint32{1, 2, 3}, [][]uint64{{12297829382473034408, 3074457345618258602}, {6148914691236517204, 15372286728091293010}, {18446744073709551612, 9223372036854775806}}, 1},
}

func TestTokens(t *testing.T) {
	for _, test := range clusters {
		t.Run(test.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(test.nodeIds, test.vNodes, test.replicaFactor)

			for i, nodeId := range test.nodeIds {
				//when
				tokens := testSubject.Tokens(nodeId)

				//then
				assert.True(t, sort.SliceIsSorted(tokens, func(i, j int) bool { return tokens[i] < tokens[j] }))

				expected := test.vNodes[i]
				assert.Len(t, tokens, len(expected))
				for _, value := range expected {
					assert.Contains(t, tokens, value)
				}
			}
		})
	}
}

func TestNeighbours(t *testing.T) {
	tests := []struct {
		clusterId int
		token     uint64
		prev      VNode
		next      VNode
	}{
		{0, 3074457345618258602, VNode{2, 18446744073709551612}, VNode{2, 6148914691236517204}},
		{0, 3074457345618258602 + 100, VNode{1, 3074457345618258602}, VNode{2, 6148914691236517204}},
		{0, 18446744073709551612, VNode{1, 15372286728091293010}, VNode{1, 3074457345618258602}},
		{0, 18446744073709551612 + 1, VNode{2, 18446744073709551612}, VNode{1, 3074457345618258602}},
		{1, 6148914691236517204, VNode{1, 3074457345618258602}, VNode{3, 9223372036854775806}},
		{1, 6148914691236517204 + 100, VNode{2, 6148914691236517204}, VNode{3, 9223372036854775806}},
	}

	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodeIds, cluster.vNodes, cluster.replicaFactor)

			//when
			prev, next := testSubject.Neighbours(test.token)

			//then
			assert.Equal(t, test.next, next)
			assert.Equal(t, test.prev, prev)

		})
	}
}

func TestReplicas(t *testing.T) {
	tests := []struct {
		clusterId int
		token     uint64
		replicas  []VNode
	}{
		{0, 3074457345618258602, []VNode{{1, 3074457345618258602}, {2, 6148914691236517204}}},
		{0, 3074457345618258602 + 100, []VNode{{1, 3074457345618258602}, {2, 6148914691236517204}}},
		{0, 18446744073709551612, []VNode{{2, 18446744073709551612}, {1, 3074457345618258602}}},
		{0, 18446744073709551612 + 1, []VNode{{2, 18446744073709551612}, {1, 3074457345618258602}}},
		{1, 6148914691236517204, []VNode{{2, 6148914691236517204}, {3, 9223372036854775806}, {1, 12297829382473034408}}},
		{1, 6148914691236517204 + 10, []VNode{{2, 6148914691236517204}, {3, 9223372036854775806}, {1, 12297829382473034408}}},
		{2, 6980919141067302587, []VNode{{2, 6148914691236517204}, {3, 9223372036854775806}, {1, 12297829382473034408}}},
		{2, 7577601381952616217, []VNode{{2, 6148914691236517204}, {3, 9223372036854775806}, {1, 12297829382473034408}}},
	}

	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodeIds, cluster.vNodes, cluster.replicaFactor)

			//when
			replicas := testSubject.Replicas(test.token)

			//then
			assert.Len(t, replicas, int(cluster.replicaFactor))
			for i, vNode := range replicas {
				assert.Equal(t, test.replicas[i], vNode)
			}
		})
	}
}

func TestPartitions(t *testing.T) {
	tests := []struct {
		clusterId  int
		nodeId     uint32
		partitions []Partition
	}{
		{3, 1, []Partition{{From: 3074457345618258602, To: 6148914691236517204 - 1, NodeId: 1}, {From: 9223372036854775806, To: 12297829382473034408 - 1, NodeId: 1}, {From: 15372286728091293010, To: 18446744073709551612 - 1, NodeId: 1}}},
		{3, 2, []Partition{{From: 6148914691236517204, To: 9223372036854775806 - 1, NodeId: 2}, {From: 12297829382473034408, To: 15372286728091293010 - 1, NodeId: 2}, {From: 18446744073709551612, To: 3074457345618258602 - 1, NodeId: 2}}},
		{4, 1, []Partition{{From: 3074457345618258602, To: 6148914691236517204 - 1, NodeId: 1}, {From: 12297829382473034408, To: 15372286728091293010 - 1, NodeId: 1}}},
		{4, 2, []Partition{{From: 6148914691236517204, To: 9223372036854775806 - 1, NodeId: 2}, {From: 15372286728091293010, To: 18446744073709551612 - 1, NodeId: 2}}},
		{4, 3, []Partition{{From: 9223372036854775806, To: 12297829382473034408 - 1, NodeId: 3}, {From: 18446744073709551612, To: 3074457345618258602 - 1, NodeId: 3}}},
		{0, 1, []Partition{{From: 18446744073709551612, To: 3074457345618258602 - 1, NodeId: 2}, {From: 3074457345618258602, To: 6148914691236517204 - 1, NodeId: 1}, {From: 6148914691236517204, To: 9223372036854775806 - 1, NodeId: 2}, {From: 9223372036854775806, To: 12297829382473034408 - 1, NodeId: 1}, {From: 12297829382473034408, To: 15372286728091293010 - 1, NodeId: 2}, {From: 15372286728091293010, To: 18446744073709551612 - 1, NodeId: 1}}},
		{0, 2, []Partition{{From: 3074457345618258602, To: 6148914691236517204 - 1, NodeId: 1}, {From: 6148914691236517204, To: 9223372036854775806 - 1, NodeId: 2}, {From: 9223372036854775806, To: 12297829382473034408 - 1, NodeId: 1}, {From: 12297829382473034408, To: 15372286728091293010 - 1, NodeId: 2}, {From: 15372286728091293010, To: 18446744073709551612 - 1, NodeId: 1}, {From: 18446744073709551612, To: 3074457345618258602 - 1, NodeId: 2}}},
		{2, 4, []Partition{{From: 18446744073709551612, To: 3074457345618258602 - 1, NodeId: 3}, {From: 3074457345618258602, To: 4611686018427387903 - 1, NodeId: 1}, {From: 4611686018427387903, To: 6148914691236517204 - 1, NodeId: 4}}},
	}

	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodeIds, cluster.vNodes, cluster.replicaFactor)

			//when
			partitions := testSubject.Partitions(test.nodeId)

			//then
			assert.Len(t, partitions, len(test.partitions))
			for i, partition := range partitions {
				assert.Equal(t, test.partitions[i], partition)
			}
		})

	}
}

func TestExcessPartitions(t *testing.T) {
	tests := []struct {
		clusterId        int
		nodeId           uint32
		excessPartitions []Partition
	}{
		{3, 1, []Partition{{From: 6148914691236517204, To: 9223372036854775806 - 1}, {From: 12297829382473034408, To: 15372286728091293010 - 1}, {From: 18446744073709551612, To: 3074457345618258602 - 1}}},
		{3, 2, []Partition{{From: 9223372036854775806, To: 12297829382473034408 - 1}, {From: 15372286728091293010, To: 18446744073709551612 - 1}, {From: 3074457345618258602, To: 6148914691236517204 - 1}}},
		{4, 1, []Partition{{From: 6148914691236517204, To: 12297829382473034408 - 1}, {From: 15372286728091293010, To: 3074457345618258602 - 1}}},
		{4, 2, []Partition{{From: 9223372036854775806, To: 15372286728091293010 - 1}, {From: 18446744073709551612, To: 6148914691236517204 - 1}}},
		{4, 3, []Partition{{From: 12297829382473034408, To: 18446744073709551612 - 1}, {From: 3074457345618258602, To: 9223372036854775806 - 1}}},
		{0, 1, []Partition{}},
		{0, 2, []Partition{}},
		{0, 3, []Partition{}},
		{2, 4, []Partition{{From: 6148914691236517204, To: 18446744073709551612 - 1}}},
	}

	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodeIds, cluster.vNodes, cluster.replicaFactor)

			//when
			partitions := testSubject.ExcessPartitions(test.nodeId)

			//then
			assert.Len(t, partitions, len(test.excessPartitions))
			for i, partition := range partitions {
				test.excessPartitions[i].NodeId = test.nodeId
				assert.Equal(t, test.excessPartitions[i], partition)
			}
		})

	}
}

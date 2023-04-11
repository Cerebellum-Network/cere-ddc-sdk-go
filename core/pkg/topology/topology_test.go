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
		{3, 1,
			[]Partition{
				{From: 3074457345618258602, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeId: 1, token: 3074457345618258602}}},
				{From: 9223372036854775806, To: 12297829382473034408 - 1, VNodes: []VNode{{nodeId: 1, token: 9223372036854775806}}},
				{From: 15372286728091293010, To: 18446744073709551612 - 1, VNodes: []VNode{{nodeId: 1, token: 15372286728091293010}}},
			},
		},
		{3, 2,
			[]Partition{
				{From: 6148914691236517204, To: 9223372036854775806 - 1, VNodes: []VNode{{nodeId: 2, token: 6148914691236517204}}},
				{From: 12297829382473034408, To: 15372286728091293010 - 1, VNodes: []VNode{{nodeId: 2, token: 12297829382473034408}}},
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeId: 2, token: 18446744073709551612}}},
			},
		},
		{4, 1,
			[]Partition{
				{From: 3074457345618258602, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeId: 1, token: 3074457345618258602}}},
				{From: 12297829382473034408, To: 15372286728091293010 - 1, VNodes: []VNode{{nodeId: 1, token: 12297829382473034408}}},
			},
		},
		{4, 2,
			[]Partition{
				{From: 6148914691236517204, To: 9223372036854775806 - 1, VNodes: []VNode{{nodeId: 2, token: 6148914691236517204}}},
				{From: 15372286728091293010, To: 18446744073709551612 - 1, VNodes: []VNode{{nodeId: 2, token: 15372286728091293010}}},
			},
		},
		{4, 3,
			[]Partition{
				{From: 9223372036854775806, To: 12297829382473034408 - 1, VNodes: []VNode{{nodeId: 3, token: 9223372036854775806}}},
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeId: 3, token: 18446744073709551612}}},
			},
		},
		{0, 1,
			[]Partition{
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeId: 2, token: 18446744073709551612}, {nodeId: 1, token: 3074457345618258602}}},
				{From: 3074457345618258602, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeId: 1, token: 3074457345618258602}, {nodeId: 2, token: 6148914691236517204}}},
				{From: 6148914691236517204, To: 9223372036854775806 - 1, VNodes: []VNode{{nodeId: 2, token: 6148914691236517204}, {nodeId: 1, token: 9223372036854775806}}},
				{From: 9223372036854775806, To: 12297829382473034408 - 1, VNodes: []VNode{{nodeId: 1, token: 9223372036854775806}, {nodeId: 2, token: 12297829382473034408}}},
				{From: 12297829382473034408, To: 15372286728091293010 - 1, VNodes: []VNode{{nodeId: 2, token: 12297829382473034408}, {nodeId: 1, token: 15372286728091293010}}},
				{From: 15372286728091293010, To: 18446744073709551612 - 1, VNodes: []VNode{{nodeId: 1, token: 15372286728091293010}, {nodeId: 2, token: 18446744073709551612}}},
			},
		},
		{0, 2,
			[]Partition{
				{From: 3074457345618258602, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeId: 1, token: 3074457345618258602}, {nodeId: 2, token: 6148914691236517204}}},
				{From: 6148914691236517204, To: 9223372036854775806 - 1, VNodes: []VNode{{nodeId: 2, token: 6148914691236517204}, {nodeId: 1, token: 9223372036854775806}}},
				{From: 9223372036854775806, To: 12297829382473034408 - 1, VNodes: []VNode{{nodeId: 1, token: 9223372036854775806}, {nodeId: 2, token: 12297829382473034408}}},
				{From: 12297829382473034408, To: 15372286728091293010 - 1, VNodes: []VNode{{nodeId: 2, token: 12297829382473034408}, {nodeId: 1, token: 15372286728091293010}}},
				{From: 15372286728091293010, To: 18446744073709551612 - 1, VNodes: []VNode{{nodeId: 1, token: 15372286728091293010}, {nodeId: 2, token: 18446744073709551612}}},
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeId: 2, token: 18446744073709551612}, {nodeId: 1, token: 3074457345618258602}}},
			},
		},
		{2, 4,
			[]Partition{
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeId: 3, token: 18446744073709551612}, {nodeId: 1, token: 3074457345618258602}, {nodeId: 4, token: 4611686018427387903}}},
				{From: 3074457345618258602, To: 4611686018427387903 - 1, VNodes: []VNode{{nodeId: 1, token: 3074457345618258602}, {nodeId: 4, token: 4611686018427387903}, {nodeId: 2, token: 6148914691236517204}}},
				{From: 4611686018427387903, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeId: 4, token: 4611686018427387903}, {nodeId: 2, token: 6148914691236517204}, {nodeId: 3, token: 9223372036854775806}}},
			},
		},
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
				assert.Equal(t, test.excessPartitions[i], partition)
			}
		})

	}
}

func TestRemoveVNode(t *testing.T) {
	tests := []struct {
		clusterId       int
		token           uint64
		expectIsRemoved bool
		expected        []VNode
	}{
		{0, 100, false, []VNode{{token: 3074457345618258602, nodeId: 1}, {token: 6148914691236517204, nodeId: 2}, {token: 9223372036854775806, nodeId: 1}, {token: 12297829382473034408, nodeId: 2}, {token: 15372286728091293010, nodeId: 1}, {token: 18446744073709551612, nodeId: 2}}},
		{0, 9223372036854775806, true, []VNode{{token: 3074457345618258602, nodeId: 1}, {token: 6148914691236517204, nodeId: 2}, {token: 12297829382473034408, nodeId: 2}, {token: 15372286728091293010, nodeId: 1}, {token: 18446744073709551612, nodeId: 2}}},
		{0, 12297829382473034408, true, []VNode{{token: 3074457345618258602, nodeId: 1}, {token: 6148914691236517204, nodeId: 2}, {token: 9223372036854775806, nodeId: 1}, {token: 15372286728091293010, nodeId: 1}, {token: 18446744073709551612, nodeId: 2}}},
	}
	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodeIds, cluster.vNodes, cluster.replicaFactor)

			//when
			ok := testSubject.RemoveVNode(test.token)

			//then
			assert.Equal(t, test.expectIsRemoved, ok)
			for i, vnode := range testSubject.VNodes() {
				assert.Equal(t, test.expected[i], vnode)
			}
		})
	}
}

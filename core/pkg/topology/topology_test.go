package topology

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

var nodeKey1 = "e5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a"
var nodeKey2 = "8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48"
var nodeKey3 = "90b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22"
var nodeKey4 = "306721211d5404bd9da88e0204360a1a9ab8b87c66c1bc2fcdd37f3c2222cc20"

var clusters = []struct {
	name          string
	nodeKeys      []string
	vNodes        [][]uint64
	replicaFactor uint
}{
	{"size 2 replication 3", []string{nodeKey1, nodeKey2}, [][]uint64{{9223372036854775806, 3074457345618258602, 15372286728091293010}, {12297829382473034408, 6148914691236517204, 18446744073709551612}}, 2},
	{"size 3 replication 3", []string{nodeKey1, nodeKey2, nodeKey3}, [][]uint64{{12297829382473034408, 3074457345618258602}, {6148914691236517204, 15372286728091293010}, {18446744073709551612, 9223372036854775806}}, 3},
	{"size 4 replication 3", []string{nodeKey1, nodeKey2, nodeKey3, nodeKey4}, [][]uint64{{12297829382473034408, 3074457345618258602}, {6148914691236517204, 15372286728091293010}, {18446744073709551612, 9223372036854775806}, {4611686018427387903}}, 3},
	{"size 2 replication 1", []string{nodeKey1, nodeKey2}, [][]uint64{{9223372036854775806, 3074457345618258602, 15372286728091293010}, {12297829382473034408, 6148914691236517204, 18446744073709551612}}, 1},
	{"size 3 replication 1", []string{nodeKey1, nodeKey2, nodeKey3}, [][]uint64{{12297829382473034408, 3074457345618258602}, {6148914691236517204, 15372286728091293010}, {18446744073709551612, 9223372036854775806}}, 1},
}

func TestTokens(t *testing.T) {
	for _, test := range clusters {
		t.Run(test.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(test.nodeKeys, test.vNodes, test.replicaFactor)

			for i, nodeKey := range test.nodeKeys {
				//when
				tokens := testSubject.Tokens(nodeKey)

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
		{0, 3074457345618258602, VNode{nodeKey2, 18446744073709551612}, VNode{nodeKey2, 6148914691236517204}},
		{0, 3074457345618258602 + 100, VNode{nodeKey1, 3074457345618258602}, VNode{nodeKey2, 6148914691236517204}},
		{0, 18446744073709551612, VNode{nodeKey1, 15372286728091293010}, VNode{nodeKey1, 3074457345618258602}},
		{0, 18446744073709551612 + 1, VNode{nodeKey2, 18446744073709551612}, VNode{nodeKey1, 3074457345618258602}},
		{1, 6148914691236517204, VNode{nodeKey1, 3074457345618258602}, VNode{nodeKey3, 9223372036854775806}},
		{1, 6148914691236517204 + 100, VNode{nodeKey2, 6148914691236517204}, VNode{nodeKey3, 9223372036854775806}},
	}

	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodeKeys, cluster.vNodes, cluster.replicaFactor)

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
		{0, 3074457345618258602, []VNode{{nodeKey1, 3074457345618258602}, {nodeKey2, 6148914691236517204}}},
		{0, 3074457345618258602 + 100, []VNode{{nodeKey1, 3074457345618258602}, {nodeKey2, 6148914691236517204}}},
		{0, 18446744073709551612, []VNode{{nodeKey2, 18446744073709551612}, {nodeKey1, 3074457345618258602}}},
		{0, 18446744073709551612 + 1, []VNode{{nodeKey2, 18446744073709551612}, {nodeKey1, 3074457345618258602}}},
		{1, 6148914691236517204, []VNode{{nodeKey2, 6148914691236517204}, {nodeKey3, 9223372036854775806}, {nodeKey1, 12297829382473034408}}},
		{1, 6148914691236517204 + 10, []VNode{{nodeKey2, 6148914691236517204}, {nodeKey3, 9223372036854775806}, {nodeKey1, 12297829382473034408}}},
		{2, 6980919141067302587, []VNode{{nodeKey2, 6148914691236517204}, {nodeKey3, 9223372036854775806}, {nodeKey1, 12297829382473034408}}},
		{2, 7577601381952616217, []VNode{{nodeKey2, 6148914691236517204}, {nodeKey3, 9223372036854775806}, {nodeKey1, 12297829382473034408}}},
	}

	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodeKeys, cluster.vNodes, cluster.replicaFactor)

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
		nodeKey    string
		partitions []Partition
	}{
		{3, nodeKey1,
			[]Partition{
				{From: 3074457345618258602, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeKey: nodeKey1, token: 3074457345618258602}}},
				{From: 9223372036854775806, To: 12297829382473034408 - 1, VNodes: []VNode{{nodeKey: nodeKey1, token: 9223372036854775806}}},
				{From: 15372286728091293010, To: 18446744073709551612 - 1, VNodes: []VNode{{nodeKey: nodeKey1, token: 15372286728091293010}}},
			},
		},
		{3, nodeKey2,
			[]Partition{
				{From: 6148914691236517204, To: 9223372036854775806 - 1, VNodes: []VNode{{nodeKey: nodeKey2, token: 6148914691236517204}}},
				{From: 12297829382473034408, To: 15372286728091293010 - 1, VNodes: []VNode{{nodeKey: nodeKey2, token: 12297829382473034408}}},
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeKey: nodeKey2, token: 18446744073709551612}}},
			},
		},
		{4, nodeKey1,
			[]Partition{
				{From: 3074457345618258602, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeKey: nodeKey1, token: 3074457345618258602}}},
				{From: 12297829382473034408, To: 15372286728091293010 - 1, VNodes: []VNode{{nodeKey: nodeKey1, token: 12297829382473034408}}},
			},
		},
		{4, nodeKey2,
			[]Partition{
				{From: 6148914691236517204, To: 9223372036854775806 - 1, VNodes: []VNode{{nodeKey: nodeKey2, token: 6148914691236517204}}},
				{From: 15372286728091293010, To: 18446744073709551612 - 1, VNodes: []VNode{{nodeKey: nodeKey2, token: 15372286728091293010}}},
			},
		},
		{4, nodeKey3,
			[]Partition{
				{From: 9223372036854775806, To: 12297829382473034408 - 1, VNodes: []VNode{{nodeKey: nodeKey3, token: 9223372036854775806}}},
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeKey: nodeKey3, token: 18446744073709551612}}},
			},
		},
		{0, nodeKey1,
			[]Partition{
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeKey: nodeKey2, token: 18446744073709551612}, {nodeKey: nodeKey1, token: 3074457345618258602}}},
				{From: 3074457345618258602, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeKey: nodeKey1, token: 3074457345618258602}, {nodeKey: nodeKey2, token: 6148914691236517204}}},
				{From: 6148914691236517204, To: 9223372036854775806 - 1, VNodes: []VNode{{nodeKey: nodeKey2, token: 6148914691236517204}, {nodeKey: nodeKey1, token: 9223372036854775806}}},
				{From: 9223372036854775806, To: 12297829382473034408 - 1, VNodes: []VNode{{nodeKey: nodeKey1, token: 9223372036854775806}, {nodeKey: nodeKey2, token: 12297829382473034408}}},
				{From: 12297829382473034408, To: 15372286728091293010 - 1, VNodes: []VNode{{nodeKey: nodeKey2, token: 12297829382473034408}, {nodeKey: nodeKey1, token: 15372286728091293010}}},
				{From: 15372286728091293010, To: 18446744073709551612 - 1, VNodes: []VNode{{nodeKey: nodeKey1, token: 15372286728091293010}, {nodeKey: nodeKey2, token: 18446744073709551612}}},
			},
		},
		{0, nodeKey2,
			[]Partition{
				{From: 3074457345618258602, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeKey: nodeKey1, token: 3074457345618258602}, {nodeKey: nodeKey2, token: 6148914691236517204}}},
				{From: 6148914691236517204, To: 9223372036854775806 - 1, VNodes: []VNode{{nodeKey: nodeKey2, token: 6148914691236517204}, {nodeKey: nodeKey1, token: 9223372036854775806}}},
				{From: 9223372036854775806, To: 12297829382473034408 - 1, VNodes: []VNode{{nodeKey: nodeKey1, token: 9223372036854775806}, {nodeKey: nodeKey2, token: 12297829382473034408}}},
				{From: 12297829382473034408, To: 15372286728091293010 - 1, VNodes: []VNode{{nodeKey: nodeKey2, token: 12297829382473034408}, {nodeKey: nodeKey1, token: 15372286728091293010}}},
				{From: 15372286728091293010, To: 18446744073709551612 - 1, VNodes: []VNode{{nodeKey: nodeKey1, token: 15372286728091293010}, {nodeKey: nodeKey2, token: 18446744073709551612}}},
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeKey: nodeKey2, token: 18446744073709551612}, {nodeKey: nodeKey1, token: 3074457345618258602}}},
			},
		},
		{2, nodeKey4,
			[]Partition{
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeKey: nodeKey3, token: 18446744073709551612}, {nodeKey: nodeKey1, token: 3074457345618258602}, {nodeKey: nodeKey4, token: 4611686018427387903}}},
				{From: 3074457345618258602, To: 4611686018427387903 - 1, VNodes: []VNode{{nodeKey: nodeKey1, token: 3074457345618258602}, {nodeKey: nodeKey4, token: 4611686018427387903}, {nodeKey: nodeKey2, token: 6148914691236517204}}},
				{From: 4611686018427387903, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeKey: nodeKey4, token: 4611686018427387903}, {nodeKey: nodeKey2, token: 6148914691236517204}, {nodeKey: nodeKey3, token: 9223372036854775806}}},
			},
		},
	}

	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodeKeys, cluster.vNodes, cluster.replicaFactor)

			//when
			partitions := testSubject.Partitions(test.nodeKey)

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
		nodeKey          string
		excessPartitions []Partition
	}{
		{3, nodeKey1, []Partition{{From: 6148914691236517204, To: 9223372036854775806 - 1}, {From: 12297829382473034408, To: 15372286728091293010 - 1}, {From: 18446744073709551612, To: 3074457345618258602 - 1}}},
		{3, nodeKey2, []Partition{{From: 9223372036854775806, To: 12297829382473034408 - 1}, {From: 15372286728091293010, To: 18446744073709551612 - 1}, {From: 3074457345618258602, To: 6148914691236517204 - 1}}},
		{4, nodeKey1, []Partition{{From: 6148914691236517204, To: 12297829382473034408 - 1}, {From: 15372286728091293010, To: 3074457345618258602 - 1}}},
		{4, nodeKey2, []Partition{{From: 9223372036854775806, To: 15372286728091293010 - 1}, {From: 18446744073709551612, To: 6148914691236517204 - 1}}},
		{4, nodeKey3, []Partition{{From: 12297829382473034408, To: 18446744073709551612 - 1}, {From: 3074457345618258602, To: 9223372036854775806 - 1}}},
		{0, nodeKey1, []Partition{}},
		{0, nodeKey2, []Partition{}},
		{0, nodeKey3, []Partition{}},
		{2, nodeKey4, []Partition{{From: 6148914691236517204, To: 18446744073709551612 - 1}}},
	}

	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodeKeys, cluster.vNodes, cluster.replicaFactor)

			//when
			partitions := testSubject.ExcessPartitions(test.nodeKey)

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
		{0, 100, false, []VNode{{token: 3074457345618258602, nodeKey: nodeKey1}, {token: 6148914691236517204, nodeKey: nodeKey2}, {token: 9223372036854775806, nodeKey: nodeKey1}, {token: 12297829382473034408, nodeKey: nodeKey2}, {token: 15372286728091293010, nodeKey: nodeKey1}, {token: 18446744073709551612, nodeKey: nodeKey2}}},
		{0, 9223372036854775806, true, []VNode{{token: 3074457345618258602, nodeKey: nodeKey1}, {token: 6148914691236517204, nodeKey: nodeKey2}, {token: 12297829382473034408, nodeKey: nodeKey2}, {token: 15372286728091293010, nodeKey: nodeKey1}, {token: 18446744073709551612, nodeKey: nodeKey2}}},
		{0, 12297829382473034408, true, []VNode{{token: 3074457345618258602, nodeKey: nodeKey1}, {token: 6148914691236517204, nodeKey: nodeKey2}, {token: 9223372036854775806, nodeKey: nodeKey1}, {token: 15372286728091293010, nodeKey: nodeKey1}, {token: 18446744073709551612, nodeKey: nodeKey2}}},
	}
	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodeKeys, cluster.vNodes, cluster.replicaFactor)

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

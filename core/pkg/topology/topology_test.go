package topology

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

var NodeKey1 = "e5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a"
var NodeKey2 = "8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48"
var NodeKey3 = "90b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22"
var NodeKey4 = "306721211d5404bd9da88e0204360a1a9ab8b87c66c1bc2fcdd37f3c2222cc20"
var NodeKey5 = "e659a7a1628cdd93febc04a4e0646ea20e9f5f0ce097d9a05290d4a9e054df4e"
var NodeKey6 = "1cbd2d43530a44705ad088af313e18f80b53ef16b36177cd4b77b846f2a5f07c"
var NodeKey7 = "be5ddb1579b72e84524fc29e78609e3caf42e85aa118ebfe0b0ad404b5bdd25f"

var clusters = []struct {
	name          string
	nodesVNodes   NodesVNodes
	replicaFactor uint
}{
	{
		"size 2 replication 3",
		NodesVNodes{
			NodeVNodes{
				NodeKey: NodeKey1,
				VNodes:  []uint64{9223372036854775806, 3074457345618258602, 15372286728091293010},
			},
			NodeVNodes{
				NodeKey: NodeKey2,
				VNodes:  []uint64{12297829382473034408, 6148914691236517204, 18446744073709551612},
			},
		},
		2,
	},
	{
		"size 3 replication 3",
		NodesVNodes{
			NodeVNodes{
				NodeKey: NodeKey1,
				VNodes:  []uint64{12297829382473034408, 3074457345618258602},
			},
			NodeVNodes{
				NodeKey: NodeKey2,
				VNodes:  []uint64{6148914691236517204, 15372286728091293010},
			},
			NodeVNodes{
				NodeKey: NodeKey3,
				VNodes:  []uint64{18446744073709551612, 9223372036854775806},
			},
		},
		3,
	},
	{
		"size 4 replication 3",
		NodesVNodes{
			NodeVNodes{
				NodeKey: NodeKey1,
				VNodes:  []uint64{12297829382473034408, 3074457345618258602},
			},
			NodeVNodes{
				NodeKey: NodeKey2,
				VNodes:  []uint64{6148914691236517204, 15372286728091293010},
			},
			NodeVNodes{
				NodeKey: NodeKey3,
				VNodes:  []uint64{18446744073709551612, 9223372036854775806},
			},
			NodeVNodes{
				NodeKey: NodeKey4,
				VNodes:  []uint64{4611686018427387903},
			},
		},
		3,
	},
	{
		"size 2 replication 1",
		NodesVNodes{
			NodeVNodes{
				NodeKey: NodeKey1,
				VNodes:  []uint64{9223372036854775806, 3074457345618258602, 15372286728091293010},
			},
			NodeVNodes{
				NodeKey: NodeKey2,
				VNodes:  []uint64{12297829382473034408, 6148914691236517204, 18446744073709551612},
			},
		},
		1,
	},
	{
		"size 3 replication 1",
		NodesVNodes{
			NodeVNodes{
				NodeKey: NodeKey1,
				VNodes:  []uint64{12297829382473034408, 3074457345618258602},
			},
			NodeVNodes{
				NodeKey: NodeKey2,
				VNodes:  []uint64{6148914691236517204, 15372286728091293010},
			},
			NodeVNodes{
				NodeKey: NodeKey3,
				VNodes:  []uint64{18446744073709551612, 9223372036854775806},
			},
		},
		1,
	},
}

func TestTokens(t *testing.T) {
	for _, test := range clusters {
		t.Run(test.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(test.nodesVNodes, test.replicaFactor)

			for _, node := range test.nodesVNodes {
				//when
				tokens := testSubject.Tokens(node.NodeKey)

				//then
				assert.True(t, sort.SliceIsSorted(tokens, func(i, j int) bool { return tokens[i] < tokens[j] }))

				expected := node.VNodes
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
		{0, 3074457345618258602, VNode{NodeKey2, 18446744073709551612}, VNode{NodeKey2, 6148914691236517204}},
		{0, 3074457345618258602 + 100, VNode{NodeKey1, 3074457345618258602}, VNode{NodeKey2, 6148914691236517204}},
		{0, 18446744073709551612, VNode{NodeKey1, 15372286728091293010}, VNode{NodeKey1, 3074457345618258602}},
		{0, 18446744073709551612 + 1, VNode{NodeKey2, 18446744073709551612}, VNode{NodeKey1, 3074457345618258602}},
		{1, 6148914691236517204, VNode{NodeKey1, 3074457345618258602}, VNode{NodeKey3, 9223372036854775806}},
		{1, 6148914691236517204 + 100, VNode{NodeKey2, 6148914691236517204}, VNode{NodeKey3, 9223372036854775806}},
	}

	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodesVNodes, cluster.replicaFactor)

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
		{0, 3074457345618258602, []VNode{{NodeKey1, 3074457345618258602}, {NodeKey2, 6148914691236517204}}},
		{0, 3074457345618258602 + 100, []VNode{{NodeKey1, 3074457345618258602}, {NodeKey2, 6148914691236517204}}},
		{0, 18446744073709551612, []VNode{{NodeKey2, 18446744073709551612}, {NodeKey1, 3074457345618258602}}},
		{0, 18446744073709551612 + 1, []VNode{{NodeKey2, 18446744073709551612}, {NodeKey1, 3074457345618258602}}},
		{1, 6148914691236517204, []VNode{{NodeKey2, 6148914691236517204}, {NodeKey3, 9223372036854775806}, {NodeKey1, 12297829382473034408}}},
		{1, 6148914691236517204 + 10, []VNode{{NodeKey2, 6148914691236517204}, {NodeKey3, 9223372036854775806}, {NodeKey1, 12297829382473034408}}},
		{2, 6980919141067302587, []VNode{{NodeKey2, 6148914691236517204}, {NodeKey3, 9223372036854775806}, {NodeKey1, 12297829382473034408}}},
		{2, 7577601381952616217, []VNode{{NodeKey2, 6148914691236517204}, {NodeKey3, 9223372036854775806}, {NodeKey1, 12297829382473034408}}},
	}

	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodesVNodes, cluster.replicaFactor)

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
		{3, NodeKey1,
			[]Partition{
				{From: 3074457345618258602, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeKey: NodeKey1, token: 3074457345618258602}}},
				{From: 9223372036854775806, To: 12297829382473034408 - 1, VNodes: []VNode{{nodeKey: NodeKey1, token: 9223372036854775806}}},
				{From: 15372286728091293010, To: 18446744073709551612 - 1, VNodes: []VNode{{nodeKey: NodeKey1, token: 15372286728091293010}}},
			},
		},
		{3, NodeKey2,
			[]Partition{
				{From: 6148914691236517204, To: 9223372036854775806 - 1, VNodes: []VNode{{nodeKey: NodeKey2, token: 6148914691236517204}}},
				{From: 12297829382473034408, To: 15372286728091293010 - 1, VNodes: []VNode{{nodeKey: NodeKey2, token: 12297829382473034408}}},
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeKey: NodeKey2, token: 18446744073709551612}}},
			},
		},
		{4, NodeKey1,
			[]Partition{
				{From: 3074457345618258602, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeKey: NodeKey1, token: 3074457345618258602}}},
				{From: 12297829382473034408, To: 15372286728091293010 - 1, VNodes: []VNode{{nodeKey: NodeKey1, token: 12297829382473034408}}},
			},
		},
		{4, NodeKey2,
			[]Partition{
				{From: 6148914691236517204, To: 9223372036854775806 - 1, VNodes: []VNode{{nodeKey: NodeKey2, token: 6148914691236517204}}},
				{From: 15372286728091293010, To: 18446744073709551612 - 1, VNodes: []VNode{{nodeKey: NodeKey2, token: 15372286728091293010}}},
			},
		},
		{4, NodeKey3,
			[]Partition{
				{From: 9223372036854775806, To: 12297829382473034408 - 1, VNodes: []VNode{{nodeKey: NodeKey3, token: 9223372036854775806}}},
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeKey: NodeKey3, token: 18446744073709551612}}},
			},
		},
		{0, NodeKey1,
			[]Partition{
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeKey: NodeKey2, token: 18446744073709551612}, {nodeKey: NodeKey1, token: 3074457345618258602}}},
				{From: 3074457345618258602, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeKey: NodeKey1, token: 3074457345618258602}, {nodeKey: NodeKey2, token: 6148914691236517204}}},
				{From: 6148914691236517204, To: 9223372036854775806 - 1, VNodes: []VNode{{nodeKey: NodeKey2, token: 6148914691236517204}, {nodeKey: NodeKey1, token: 9223372036854775806}}},
				{From: 9223372036854775806, To: 12297829382473034408 - 1, VNodes: []VNode{{nodeKey: NodeKey1, token: 9223372036854775806}, {nodeKey: NodeKey2, token: 12297829382473034408}}},
				{From: 12297829382473034408, To: 15372286728091293010 - 1, VNodes: []VNode{{nodeKey: NodeKey2, token: 12297829382473034408}, {nodeKey: NodeKey1, token: 15372286728091293010}}},
				{From: 15372286728091293010, To: 18446744073709551612 - 1, VNodes: []VNode{{nodeKey: NodeKey1, token: 15372286728091293010}, {nodeKey: NodeKey2, token: 18446744073709551612}}},
			},
		},
		{0, NodeKey2,
			[]Partition{
				{From: 3074457345618258602, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeKey: NodeKey1, token: 3074457345618258602}, {nodeKey: NodeKey2, token: 6148914691236517204}}},
				{From: 6148914691236517204, To: 9223372036854775806 - 1, VNodes: []VNode{{nodeKey: NodeKey2, token: 6148914691236517204}, {nodeKey: NodeKey1, token: 9223372036854775806}}},
				{From: 9223372036854775806, To: 12297829382473034408 - 1, VNodes: []VNode{{nodeKey: NodeKey1, token: 9223372036854775806}, {nodeKey: NodeKey2, token: 12297829382473034408}}},
				{From: 12297829382473034408, To: 15372286728091293010 - 1, VNodes: []VNode{{nodeKey: NodeKey2, token: 12297829382473034408}, {nodeKey: NodeKey1, token: 15372286728091293010}}},
				{From: 15372286728091293010, To: 18446744073709551612 - 1, VNodes: []VNode{{nodeKey: NodeKey1, token: 15372286728091293010}, {nodeKey: NodeKey2, token: 18446744073709551612}}},
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeKey: NodeKey2, token: 18446744073709551612}, {nodeKey: NodeKey1, token: 3074457345618258602}}},
			},
		},
		{2, NodeKey4,
			[]Partition{
				{From: 18446744073709551612, To: 3074457345618258602 - 1, VNodes: []VNode{{nodeKey: NodeKey3, token: 18446744073709551612}, {nodeKey: NodeKey1, token: 3074457345618258602}, {nodeKey: NodeKey4, token: 4611686018427387903}}},
				{From: 3074457345618258602, To: 4611686018427387903 - 1, VNodes: []VNode{{nodeKey: NodeKey1, token: 3074457345618258602}, {nodeKey: NodeKey4, token: 4611686018427387903}, {nodeKey: NodeKey2, token: 6148914691236517204}}},
				{From: 4611686018427387903, To: 6148914691236517204 - 1, VNodes: []VNode{{nodeKey: NodeKey4, token: 4611686018427387903}, {nodeKey: NodeKey2, token: 6148914691236517204}, {nodeKey: NodeKey3, token: 9223372036854775806}}},
			},
		},
	}

	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodesVNodes, cluster.replicaFactor)

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
		{3, NodeKey1, []Partition{{From: 6148914691236517204, To: 9223372036854775806 - 1}, {From: 12297829382473034408, To: 15372286728091293010 - 1}, {From: 18446744073709551612, To: 3074457345618258602 - 1}}},
		{3, NodeKey2, []Partition{{From: 9223372036854775806, To: 12297829382473034408 - 1}, {From: 15372286728091293010, To: 18446744073709551612 - 1}, {From: 3074457345618258602, To: 6148914691236517204 - 1}}},
		{4, NodeKey1, []Partition{{From: 6148914691236517204, To: 12297829382473034408 - 1}, {From: 15372286728091293010, To: 3074457345618258602 - 1}}},
		{4, NodeKey2, []Partition{{From: 9223372036854775806, To: 15372286728091293010 - 1}, {From: 18446744073709551612, To: 6148914691236517204 - 1}}},
		{4, NodeKey3, []Partition{{From: 12297829382473034408, To: 18446744073709551612 - 1}, {From: 3074457345618258602, To: 9223372036854775806 - 1}}},
		{0, NodeKey1, []Partition{}},
		{0, NodeKey2, []Partition{}},
		{0, NodeKey3, []Partition{}},
		{2, NodeKey4, []Partition{{From: 6148914691236517204, To: 18446744073709551612 - 1}}},
	}

	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodesVNodes, cluster.replicaFactor)

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
		{0, 100, false, []VNode{{token: 3074457345618258602, nodeKey: NodeKey1}, {token: 6148914691236517204, nodeKey: NodeKey2}, {token: 9223372036854775806, nodeKey: NodeKey1}, {token: 12297829382473034408, nodeKey: NodeKey2}, {token: 15372286728091293010, nodeKey: NodeKey1}, {token: 18446744073709551612, nodeKey: NodeKey2}}},
		{0, 9223372036854775806, true, []VNode{{token: 3074457345618258602, nodeKey: NodeKey1}, {token: 6148914691236517204, nodeKey: NodeKey2}, {token: 12297829382473034408, nodeKey: NodeKey2}, {token: 15372286728091293010, nodeKey: NodeKey1}, {token: 18446744073709551612, nodeKey: NodeKey2}}},
		{0, 12297829382473034408, true, []VNode{{token: 3074457345618258602, nodeKey: NodeKey1}, {token: 6148914691236517204, nodeKey: NodeKey2}, {token: 9223372036854775806, nodeKey: NodeKey1}, {token: 15372286728091293010, nodeKey: NodeKey1}, {token: 18446744073709551612, nodeKey: NodeKey2}}},
	}
	for _, test := range tests {
		cluster := clusters[test.clusterId]
		t.Run(cluster.name, func(t *testing.T) {
			//given
			testSubject := NewTopology(cluster.nodesVNodes, cluster.replicaFactor)

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

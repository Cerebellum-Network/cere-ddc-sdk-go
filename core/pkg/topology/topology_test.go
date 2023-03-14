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
	{"2", []uint32{1, 2}, [][]uint64{{9223372036854775806, 3074457345618258602, 15372286728091293010}, {12297829382473034408, 6148914691236517204, 18446744073709551612}}, 2},
	{"3", []uint32{1, 2, 3}, [][]uint64{{12297829382473034408, 3074457345618258602}, {6148914691236517204, 15372286728091293010}, {18446744073709551612, 9223372036854775806}}, 3},
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
			assert.Equal(t, test.next, *next)
			assert.Equal(t, test.prev, *prev)

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
				assert.Equal(t, test.replicas[i], *vNode)
			}
		})
	}
}

package topology

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

var partition = Partition{From: 0, To: 10000}
var reversePartition = Partition{From: partition.To, To: partition.From}

func TestBelongs(t *testing.T) {
	tests := []struct {
		partition Partition
		token     uint64
		expected  bool
	}{
		{partition, 0, true},
		{partition, 10000, true},
		{partition, 100, true},
		{partition, uint64(math.MaxUint64), false},
		{partition, 10000000, false},

		{reversePartition, 0, true},
		{reversePartition, 10000, true},
		{reversePartition, 10000000, true},
		{reversePartition, uint64(math.MaxUint64), true},
		{reversePartition, 100, false},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("partition %s token %d", test.partition.String(), test.token), func(t *testing.T) {
			//when
			result := test.partition.Belongs(test.token)

			//then
			assert.Equal(t, test.expected, result)
		})
	}
}

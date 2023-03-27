package utils

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRemoveSorted(t *testing.T) {
	tests := []struct {
		i        int
		expected []byte
	}{
		{-1, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{10, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{2, []byte{0, 1, 3, 4, 5, 6, 7, 8, 9}},
		{5, []byte{0, 1, 2, 3, 4, 6, 7, 8, 9}},
		{7, []byte{0, 1, 2, 3, 4, 5, 6, 8, 9}},
		{9, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8}},
		{0, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}},
	}
	for _, test := range tests {
		slice := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		t.Run(fmt.Sprintf("index=%d", test.i), func(t *testing.T) {})
		{
			//when
			result := RemoveSorted(slice, test.i)

			//then
			assert.Equal(t, cap(test.expected), cap(result))
			assert.Equal(t, len(test.expected), len(result))
			assert.True(t, bytes.Equal(test.expected, result))
		}
	}
}

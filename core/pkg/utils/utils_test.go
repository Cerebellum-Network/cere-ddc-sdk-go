package utils

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strconv"
	"testing"
)

func TestIsSuccessHttpStatus(t *testing.T) {
	tests := []struct {
		code   int
		expect bool
	}{
		{http.StatusOK, true},
		{http.StatusCreated, true},
		{http.StatusAccepted, true},
		{299, true},
		{300, false},
		{199, false},
		{http.StatusBadRequest, false},
		{http.StatusBadGateway, false},
		{http.StatusInternalServerError, false},
	}
	for _, test := range tests {
		t.Run(strconv.FormatInt(int64(test.code), 10), func(t *testing.T) {
			//when
			result := IsSuccessHttpStatus(test.code)

			//then
			assert.Equal(t, test.expect, result)
		})
	}
}

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

func TestRandomInt64(t *testing.T) {
	tests := []struct {
		max int64
	}{
		{1},
		{100},
		{500000},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%d", test.max), func(t *testing.T) {
			//when
			result, err := RandomInt64(test.max)

			//then
			assert.NoError(t, err)
			assert.Less(t, result, test.max)
		})
	}
}

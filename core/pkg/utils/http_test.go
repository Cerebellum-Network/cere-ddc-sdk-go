package utils

import (
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

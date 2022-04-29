//nolint
package mock

import (
	"github.com/patractlabs/go-patract/api"
	"github.com/stretchr/testify/mock"
)

type contractMock struct {
	mock.Mock
	encodedResult string
}

func (m *contractMock) CallToRead(api.Context, interface{}, []string, ...interface{}) error {
	return nil
}
func (m *contractMock) CallToReadEncoded(api.Context, []string, ...interface{}) (string, error) {
	return m.encodedResult, nil
}
func (m *contractMock) GetAccountIDSS58() string {
	return "id"
}

func (m *contractMock) MockEncodedResult(result string) {
	m.encodedResult = result
}

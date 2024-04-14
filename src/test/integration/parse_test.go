package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseTestCases_SingleRequest(t *testing.T) {
	data := []byte(`
name: my testcase
uri: /ping
expected-status: 200
`)

	tc := &TestCase{}
	require.NoError(t, yaml.Unmarshal(data, tc))
	assert.Equal(t, "my testcase", tc.Name)
	require.Len(t, tc.Requests, 1)
	assert.Equal(t, "/ping", tc.Requests[0].URI)
	assert.Equal(t, 200, tc.Requests[0].ExpectedStatus)
}

func TestParseTestCases_MultipleRequests(t *testing.T) {
	data := []byte(`
name: my testcase
requests:
  - uri: /ping
    expected-status: 200
  - uri: /ping
    expected-status: 500
`)

	tc := &TestCase{}
	require.NoError(t, yaml.Unmarshal(data, tc))
	assert.Equal(t, "my testcase", tc.Name)
	require.Len(t, tc.Requests, 2)
	assert.Equal(t, "/ping", tc.Requests[0].URI)
	assert.Equal(t, 200, tc.Requests[0].ExpectedStatus)
	assert.Equal(t, "/ping", tc.Requests[1].URI)
	assert.Equal(t, 500, tc.Requests[1].ExpectedStatus)
}

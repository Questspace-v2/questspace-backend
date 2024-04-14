package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestRunner_SetVariable(t *testing.T) {
	var testCases = []struct {
		name     string
		input    string
		vars     map[string]interface{}
		expected string
	}{
		{
			name:  "basic var",
			input: "some$DATAdata",
			vars: map[string]interface{}{
				"DATA": "123",
			},
			expected: "some123data",
		},
		{
			name:  "var whole string",
			input: "$SOME_VAR",
			vars: map[string]interface{}{
				"SOME_VAR": "123",
			},
			expected: "123",
		},
		{
			name:     "no override any",
			input:    "$ANY$",
			vars:     nil,
			expected: "$ANY$",
		},
		{
			name:     "no override set",
			input:    "$SET$",
			vars:     nil,
			expected: "$SET$",
		},
		{
			name:     "not found",
			input:    "$SOME_DATA",
			vars:     nil,
			expected: varNotFound,
		},
		{
			name:  "two overrides",
			input: "$SOME_VAR and $VAR",
			vars: map[string]interface{}{
				"SOME_VAR": "123",
				"VAR":      "456",
			},
			expected: "123 and 456",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runner := TestRunner{
				variables: tc.vars,
			}
			got := runner.setVariable(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestTestRunner_SetVariables(t *testing.T) {
	uri := "/quest/$ID"
	json := `{"quest": {"id": "$ID", "another_val": 123, "some": "$ANY$"}}`

	runner := TestRunner{
		variables: map[string]interface{}{
			"ID": "123-456-789-123",
		},
	}
	newURI, newJSON := runner.setVariables(t, uri, json)
	t.Log(newURI, newJSON)
	assert.Contains(t, newURI, runner.variables["ID"].(string))
	assert.Contains(t, newJSON, runner.variables["ID"].(string))
	assert.Contains(t, newJSON, anySign)
}

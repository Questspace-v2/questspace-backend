package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

const (
	varStart    = '$'
	anySign     = "$ANY$"
	anyVal      = "<any-value>"
	setPrefix   = "$SET$:"
	varNotFound = "<variable-not-found>"
)

type TestRunner struct {
	serverURL string
	client    *http.Client

	variables map[string]interface{}
}

func NewTestRunner(serverURL string) *TestRunner {
	return &TestRunner{
		serverURL: serverURL,
		client:    http.DefaultClient,
		variables: make(map[string]interface{}),
	}
}

func (r *TestRunner) Fetch(t *testing.T, method, uri string, authorization string, JSONData string) (code int, data string) {
	t.Helper()

	var body io.Reader = http.NoBody

	uri, authorization, JSONData = r.setVariables(t, uri, authorization, JSONData)

	if JSONData != "" {
		body = bytes.NewBuffer([]byte(JSONData))
	}
	req, err := http.NewRequestWithContext(context.Background(), method, r.serverURL+uri, body)
	require.NoError(t, err)
	if authorization != "" {
		req.Header.Set("Authorization", "Bearer "+authorization)
	}

	resp, err := r.client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	status := resp.StatusCode
	jsonData, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return status, string(jsonData)
}

func (r *TestRunner) setVariables(t *testing.T, uri string, auth string, JSONData string) (newURI, newAuth, newJSONData string) {
	t.Helper()

	newURI = r.setVariable(uri)
	newAuth = r.setVariable(auth)
	var data []byte
	if JSONData != "" {
		var err error
		holder := new(interface{})
		require.NoError(t, json.Unmarshal([]byte(JSONData), holder))
		*holder = r.setJSONVariables(*holder)
		data, err = json.Marshal(*holder)
		require.NoError(t, err)
	}

	return newURI, newAuth, string(data)
}

func (r *TestRunner) setJSONVariables(json interface{}) interface{} {
	switch data := json.(type) {
	case map[string]interface{}:
		for k, v := range data {
			switch v := v.(type) {
			case map[string]interface{}:
				data[k] = r.setJSONVariables(v)

			case []interface{}:
				for i, item := range v {
					v[i] = r.setJSONVariables(item)
				}

			case string:
				data[k] = r.setVariable(v)
			}
		}

	case []interface{}:
		for i, item := range data {
			data[i] = r.setJSONVariables(item)
		}
	}

	return json
}

func (r *TestRunner) setVariable(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	isVar := false
	var start, end int
	for i, rn := range s {
		if !isVar && rn != varStart {
			_, _ = b.WriteRune(rn)
			continue
		}
		if !isVar {
			start = i + 1
			isVar = true
			continue
		}

		if ('A' <= rn && rn <= 'Z') || ('0' <= rn && rn <= '9') || rn == '_' {
			continue
		} else {
			end = i
			varname := s[start:end]
			if varname == "ANY" {
				_, _ = b.WriteString("$ANY")
			} else if varname == "SET" {
				_, _ = b.WriteString("$SET")
			} else if varname == "" {
				_, _ = b.WriteRune(varStart)
			} else if val, ok := r.variables[varname]; ok {
				_, _ = b.WriteString(fmt.Sprint(val))
			} else {
				_, _ = b.WriteString(varNotFound)
			}
			_, _ = b.WriteRune(rn)
			isVar = false
		}
	}
	if isVar {
		if start == len(s)-1 {
			_, _ = b.WriteRune(varStart)
		} else if val, ok := r.variables[s[start:]]; ok {
			_, _ = b.WriteString(fmt.Sprint(val))
		} else {
			_, _ = b.WriteString(varNotFound)
		}
	}

	return b.String()
}

type expectations struct {
	Set map[string]string
	Any map[string]struct{}
}

func newExpectations() expectations {
	return expectations{
		Set: make(map[string]string),
		Any: make(map[string]struct{}),
	}
}

func (r *TestRunner) preprocessJSON(data interface{}, prevKey string, expect *expectations) interface{} {
	switch data := data.(type) {
	case map[string]interface{}:
		for k, v := range data {
			key := k
			if prevKey != "" {
				key = prevKey + "." + key
			}

			switch v := v.(type) {
			case map[string]interface{}:
				data[k] = r.preprocessJSON(v, key, expect)

			case []interface{}:
				for i, item := range v {
					v[i] = r.preprocessJSON(item, key+"."+strconv.Itoa(i), expect)
				}

			case string:
				switch {
				case v == anySign:
					expect.Any[key] = struct{}{}
					data[k] = anyVal

				case strings.HasPrefix(v, setPrefix):
					varName := strings.TrimPrefix(v, setPrefix)
					expect.Set[key] = varName
					data[k] = anyVal

				default:
					data[k] = r.setVariable(v)
				}
			}
		}

	case []interface{}:
		for i, item := range data {
			key := strconv.Itoa(i)
			if prevKey != "" {
				key = prevKey + "." + key
			}

			data[i] = r.preprocessJSON(item, key, expect)
		}
	}

	return data
}

func (r *TestRunner) digestJSON(data interface{}, prevKey string, expect *expectations) interface{} {
	switch data := data.(type) {
	case map[string]interface{}:
		for k, v := range data {
			key := k
			if prevKey != "" {
				key = prevKey + "." + key
			}

			if setName, ok := expect.Set[key]; ok {
				r.variables[setName] = v
				data[k] = anyVal
			} else if _, isAny := expect.Any[key]; isAny {
				data[k] = anyVal
			} else {
				data[k] = r.digestJSON(v, key, expect)
			}
		}

	case []interface{}:
		for i, item := range data {
			key := strconv.Itoa(i)
			if prevKey != "" {
				key = prevKey + "." + key
			}

			if setName, ok := expect.Set[key]; ok {
				r.variables[setName] = item
				data[i] = anyVal
			} else if _, isAny := expect.Any[key]; isAny {
				data[i] = anyVal
			} else {
				data[i] = r.digestJSON(item, key, expect)
			}
		}
	}

	return data
}

func (r *TestRunner) VerifyData(t *testing.T, expected, actual string) {
	t.Helper()

	expectedData := new(interface{})
	require.NoError(t, json.Unmarshal([]byte(expected), expectedData))
	exp := newExpectations()
	*expectedData = r.preprocessJSON(*expectedData, "", &exp)

	gotData := new(interface{})
	require.NoError(t, json.Unmarshal([]byte(actual), gotData))
	*gotData = r.digestJSON(*gotData, "", &exp)

	needExp, err := json.Marshal(*expectedData)
	require.NoError(t, err)
	needGot, err := json.Marshal(*gotData)
	require.NoError(t, err)

	assert.JSONEq(t, string(needExp), string(needGot))
}

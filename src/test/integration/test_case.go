package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type RequestCase struct {
	Method        string `yaml:"method"`
	URI           string `yaml:"uri"`
	Authorization string `yaml:"authorization,omitempty"`
	JSONInput     string `yaml:"json-input,omitempty"`

	ExpectedStatus int    `yaml:"expected-status"`
	ExpectedJSON   string `yaml:"expected-json,omitempty"`
}

type TestCase struct {
	Name     string        `yaml:"name"`
	Requests []RequestCase `yaml:"requests"`
}

func (t *TestCase) UnmarshalYAML(value *yaml.Node) error {
	//TODO(svayp11): come up with more intelligent way to use two methods of parsing
	type dummy struct {
		Name     string        `yaml:"name"`
		Requests []RequestCase `yaml:"requests"`
		URI      bool          `yaml:"uri"` // dummy placeholder to fail on embedded request
	}
	tc := new(dummy)
	err := value.Decode(tc)
	if err == nil {
		t.Name = tc.Name
		t.Requests = tc.Requests
		if len(tc.Requests) == 0 {
			return fmt.Errorf("expected at least one request, but got none")
		}
		return nil
	}

	type singleReqTc struct {
		Name        string `yaml:"name"`
		RequestCase `yaml:",inline"`
	}

	stc := new(singleReqTc)
	err = value.Decode(stc)
	if err != nil {
		return err
	}

	t.Name = stc.Name
	t.Requests = []RequestCase{stc.RequestCase}
	return nil
}

func ListTestDirs(t *testing.T, path string) []string {
	t.Helper()

	files, err := os.ReadDir(path)
	require.NoError(t, err)

	names := make([]string, 0, len(files))
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		names = append(names, f.Name())
	}

	toInt := func(name string) int {
		i, err := strconv.Atoi(name)
		require.NoError(t, err)
		return i
	}

	sort.Slice(names, func(i, j int) bool {
		return toInt(names[i]) < toInt(names[j])
	})

	return names
}

func ReadTestCase(t *testing.T, path string) *TestCase {
	t.Helper()

	tcData, err := os.ReadFile(filepath.Join(path, "tc.yaml"))
	require.NoError(t, err)

	tc := new(TestCase)
	require.NoError(t, yaml.Unmarshal(tcData, tc))

	return tc
}

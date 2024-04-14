package integration

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"questspace/test/testutils"
)

func TestMain(m *testing.M) {
	os.Exit(testutils.InitApplication(m))
}

func TestQuestspace(t *testing.T) {
	casesDir := path.Join("./testdata", "cases")
	testDirs := ListTestDirs(t, casesDir)

	for _, dir := range testDirs {
		tc := ReadTestCase(t, filepath.Join(casesDir, dir))

		t.Run(dir+"/"+tc.Name, func(t *testing.T) {
			testutils.StartServer(t)
			runner := NewTestRunner(testutils.ServerURL)

			for _, req := range tc.Requests {
				code, body := runner.Fetch(t, req.Method, req.URI, req.Authorization, req.JSONInput)
				if assert.Equal(t, req.ExpectedStatus, code) {
					if len(req.ExpectedJSON) > 0 {
						runner.VerifyData(t, req.ExpectedJSON, body)
					}
				} else {
					t.Fatalf(`expected status %d but got %d. Stopping test case...`, req.ExpectedStatus, code)
				}
			}

		})
	}
}

package integration

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"questspace/test/testutils"
)

const (
	RunTCkey = "RUN_TC"
)

func TestMain(m *testing.M) {
	os.Exit(testutils.InitApplication(m))
}

func TestQuestspace(t *testing.T) {
	casesDir := path.Join("./testdata", "cases")
	testDirs := ListTestDirs(t, casesDir)

	runTC := os.Getenv(RunTCkey)

	for _, dir := range testDirs {
		tc := ReadTestCase(t, filepath.Join(casesDir, dir))
		tcFullName := dir + "/" + tc.Name

		t.Run(tcFullName, func(t *testing.T) {
			if tc.Ignore {
				t.Skipf("WARNING: Test case %q is ignored. Skipping...", tcFullName)
			}
			if len(runTC) != 0 && !strings.HasPrefix(tcFullName, runTC+"/") {
				t.Skipf("WARNING: Test case %q is ignored by %s. Skipping...", tcFullName, RunTCkey)
			}
			testutils.StartServer(t)
			runner := NewTestRunner(testutils.ServerURL)

			for _, req := range tc.Requests {
				code, body := runner.Fetch(t, req.Method, req.URI, req.Authorization, req.JSONInput)
				if assert.Equal(t, req.ExpectedStatus, code) {
					if len(req.ExpectedJSON) > 0 {
						runner.VerifyData(t, req.ExpectedJSON, body)
					}
				} else {
					t.Fatalf("expected status %d but got %d. Stopping test case...\nBody: \n%s", req.ExpectedStatus, code, body)
				}
			}
		})
	}
}

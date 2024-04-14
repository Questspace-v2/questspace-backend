package testutils

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

type Database interface {
	Host() string
	Port() string
	Clean(t *testing.T)
}

const (
	ServerURL            = "http://localhost:8080"
	QuestspaceServerPath = "questspace/cmd/questspace"
)

var (
	PG         Database
	ConfigPath string

	binCache *BinaryCache
	once     sync.Once
)

func InitApplication(m *testing.M) (code int) {
	postgresContainer := StartDockerPG()
	PG = postgresContainer
	var closer CloserFunc
	ConfigPath, closer = CreateTestingConfig()

	code = m.Run()

	closer()
	postgresContainer.Close()
	return code
}

func StartServer(t *testing.T) {
	t.Helper()

	once.Do(func() {
		binCache = NewBinaryCache()
	})

	binary := binCache.LoadBinary(QuestspaceServerPath)
	cmd := exec.Command(binary, "--config", ConfigPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Errorf("Error starting server: %v", err)
		t.FailNow()
	}
	numRetries := 0
	var err error
	var resp *http.Response
	for resp, err = http.Get(ServerURL + "/ping"); err != nil; resp, err = http.Get(ServerURL + "/ping") { //nolint:bodyclose,noctx
		t.Logf("Retry #%d: %v", numRetries, err)
		numRetries++
		time.Sleep(1 * time.Second)
		if numRetries > 10 {
			break
		}
	}
	if err != nil {
		t.Fatalf("Error pinging server: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	t.Cleanup(func() {
		PG.Clean(t)
	})
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
	})
}

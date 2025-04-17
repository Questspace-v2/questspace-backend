package migrations

import (
	"embed"
	"strings"

	"github.com/yandex/perforator/library/go/core/xerrors"
)

//go:embed *.sql
var migrationFiles embed.FS

func QuestspaceAsText() (string, error) {
	data, err := migrationFiles.ReadDir(".")
	if err != nil {
		return "", xerrors.Errorf("read current migrations dir: %w", err)
	}
	var b strings.Builder
	for _, entry := range data {
		if entry.IsDir() {
			continue
		}
		migrationData, err := migrationFiles.ReadFile(entry.Name())
		if err != nil {
			return "", xerrors.Errorf("read %q: %w", entry.Name(), err)
		}
		if _, err := b.Write(migrationData); err != nil {
			return "", xerrors.Errorf("write data to builder: %w", err)
		}
	}
	return b.String(), nil
}

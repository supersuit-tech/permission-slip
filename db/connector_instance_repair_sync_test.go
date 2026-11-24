package db

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDanglingRepairSQLMatchesSQLiteMigration(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(thisFile), "migrations_sqlite", "00016_drop_dangling_connector_instances.sql")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	norm := func(s string) string {
		return strings.Join(strings.Fields(s), " ")
	}
	fileNorm := norm(string(body))
	for _, stmt := range danglingConnectorRepairSQL {
		if !strings.Contains(fileNorm, norm(stmt)) {
			t.Errorf("sqlite migration is missing repair statement:\n%s", stmt)
		}
	}
}

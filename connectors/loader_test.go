package connectors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeValidConnectorDir creates a temp subdirectory with a valid manifest and
// executable so LoadExternalConnectors picks it up.
func writeValidConnectorDir(t *testing.T, parent, id, name string) string {
	t.Helper()
	dir := filepath.Join(parent, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating connector dir: %v", err)
	}

	manifest := ConnectorManifest{
		ID:   id,
		Name: name,
		Actions: []ManifestAction{
			{ActionType: id + ".do", Name: "Do"},
		},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshaling manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "connector.json"), data, 0o644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}

	script := "#!/bin/sh\necho '{\"success\":true}'\n"
	if err := os.WriteFile(filepath.Join(dir, "connector"), []byte(script), 0o755); err != nil {
		t.Fatalf("writing connector binary: %v", err)
	}

	return dir
}

func TestDefaultConnectorsDir_EnvVarOverride(t *testing.T) {
	t.Setenv("CONNECTORS_DIR", "/app/connectors")
	got := DefaultConnectorsDir()
	if got != "/app/connectors" {
		t.Errorf("DefaultConnectorsDir() = %q, want %q", got, "/app/connectors")
	}
}

func TestDefaultConnectorsDir_FallbackWithoutEnvVar(t *testing.T) {
	t.Setenv("CONNECTORS_DIR", "")
	got := DefaultConnectorsDir()
	if got == "" {
		t.Fatal("DefaultConnectorsDir() returned empty string without CONNECTORS_DIR set")
	}
	if !filepath.IsAbs(got) {
		t.Errorf("DefaultConnectorsDir() = %q, expected an absolute path", got)
	}
	if !strings.HasSuffix(got, filepath.Join(".permission_slip", "connectors")) {
		t.Errorf("DefaultConnectorsDir() = %q, expected to end with .permission_slip/connectors", got)
	}
}

func TestLoadExternalConnectors_ValidDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeValidConnectorDir(t, root, "jira", "Jira")

	loaded, err := LoadExternalConnectors(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(loaded))
	}
	if loaded[0].ID() != "jira" {
		t.Errorf("ID = %q, want %q", loaded[0].ID(), "jira")
	}
}

func TestLoadExternalConnectors_MultipleConnectors(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeValidConnectorDir(t, root, "jira", "Jira")
	writeValidConnectorDir(t, root, "pagerduty", "PagerDuty")

	loaded, err := LoadExternalConnectors(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 connectors, got %d", len(loaded))
	}
}

func TestLoadExternalConnectors_NonexistentDir(t *testing.T) {
	t.Parallel()
	loaded, err := LoadExternalConnectors("/nonexistent/path")
	if err != nil {
		t.Fatalf("expected nil error for nonexistent dir, got: %v", err)
	}
	if loaded != nil {
		t.Fatalf("expected nil for nonexistent dir, got %d connectors", len(loaded))
	}
}

func TestLoadExternalConnectors_EmptyDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	loaded, err := LoadExternalConnectors(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected 0 connectors, got %d", len(loaded))
	}
}

func TestLoadExternalConnectors_SkipsInvalidManifest(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// One valid, one with a bad manifest.
	writeValidConnectorDir(t, root, "good", "Good")

	badDir := filepath.Join(root, "bad")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "connector.json"), []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "connector"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadExternalConnectors(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 connector (skipping bad), got %d", len(loaded))
	}
	if loaded[0].ID() != "good" {
		t.Errorf("expected 'good', got %q", loaded[0].ID())
	}
}

func TestLoadExternalConnectors_SkipsMissingBinary(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	dir := filepath.Join(root, "nobinary")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := ConnectorManifest{
		ID:      "nobinary",
		Name:    "No Binary",
		Actions: []ManifestAction{{ActionType: "nobinary.do", Name: "Do"}},
	}
	data, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(dir, "connector.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	// No connector binary.

	loaded, err := LoadExternalConnectors(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected 0 connectors (binary missing), got %d", len(loaded))
	}
}

func TestLoadExternalConnectors_SkipsNonExecutableBinary(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	dir := filepath.Join(root, "noexec")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := ConnectorManifest{
		ID:      "noexec",
		Name:    "No Exec",
		Actions: []ManifestAction{{ActionType: "noexec.do", Name: "Do"}},
	}
	data, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(dir, "connector.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	// Write binary without execute permission.
	if err := os.WriteFile(filepath.Join(dir, "connector"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadExternalConnectors(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected 0 connectors (not executable), got %d", len(loaded))
	}
}

func TestLoadExternalConnectors_SkipsRegularFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// Place a regular file in the connectors directory (not a subdirectory).
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeValidConnectorDir(t, root, "valid", "Valid")

	loaded, err := LoadExternalConnectors(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 connector (skipping file), got %d", len(loaded))
	}
}

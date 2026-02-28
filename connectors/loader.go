package connectors

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// DefaultConnectorsDir returns the directory for custom connector installations.
// If the CONNECTORS_DIR environment variable is set, its value is returned.
// Otherwise it falls back to ~/.permission_slip/connectors/.
func DefaultConnectorsDir() string {
	if dir := os.Getenv("CONNECTORS_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".permission_slip", "connectors")
}

// LoadExternalConnectors scans a directory for external connector installations.
// Each subdirectory must contain a connector.json manifest and a "connector"
// executable. Returns the loaded connectors. Non-fatal issues (invalid manifests,
// missing binaries, etc.) are logged and the corresponding connectors are skipped.
// Fatal errors (e.g., unreadable directory) are returned as the error return value.
func LoadExternalConnectors(dir string) ([]*ExternalConnector, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no connectors directory — that's fine
		}
		return nil, fmt.Errorf("reading connectors directory %s: %w", dir, err)
	}

	var loaded []*ExternalConnector
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		connDir := filepath.Join(dir, entry.Name())
		verified, err := VerifyConnectorDir(connDir)
		if err != nil {
			log.Printf("Warning: skipping connector in %s: %v", connDir, err)
			continue
		}

		loaded = append(loaded, NewExternalConnector(verified.Manifest, verified.BinPath))
		log.Printf("Loaded external connector: %s (%s) from %s", verified.Manifest.ID, verified.Manifest.Name, connDir)
	}

	return loaded, nil
}

package connectors

import (
	"fmt"
	"os"
	"path/filepath"
)

// VerifiedConnector holds the results of validating a connector directory.
type VerifiedConnector struct {
	Manifest *ConnectorManifest
	BinPath  string
}

// VerifyConnectorDir checks that a directory contains a valid connector.json
// manifest and an executable "connector" binary. It returns the parsed manifest
// and binary path on success. Both LoadExternalConnectors and the
// install-connectors command use this to avoid duplicating validation logic.
func VerifyConnectorDir(dir string) (*VerifiedConnector, error) {
	manifestPath := filepath.Join(dir, "connector.json")
	binPath := filepath.Join(dir, "connector")

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	info, err := os.Stat(binPath)
	if err != nil {
		return nil, fmt.Errorf("executable not found at %s (run 'make install-connectors')", binPath)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("'connector' at %s is a directory, not an executable", binPath)
	}
	if info.Mode()&0o111 == 0 {
		return nil, fmt.Errorf("'connector' at %s is not executable", binPath)
	}

	return &VerifiedConnector{Manifest: manifest, BinPath: binPath}, nil
}

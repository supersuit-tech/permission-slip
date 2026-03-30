// Command install-connectors reads connector configuration (from the
// CUSTOM_CONNECTORS_JSON env var or the custom-connectors.json file), clones
// each connector's Git repository into the connectors directory (set via
// CONNECTORS_DIR or defaulting to ~/.permission_slip/connectors/<name>/),
// and builds the connector binary by running "make build" in the repo.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
	_ "github.com/supersuit-tech/permission-slip/connectors/providers"
)

// configFile is the name of the configuration file in the repo root.
const configFile = "custom-connectors.json"

// config represents the custom-connectors.json file.
type config struct {
	Connectors []connectorEntry `json:"connectors"`
}

// connectorEntry is a single connector repo reference in the config file.
type connectorEntry struct {
	Repo string `json:"repo"` // Git clone URL
	Ref  string `json:"ref"`  // Git ref to checkout (branch, tag, commit)
}

func main() {
	log.SetFlags(0)

	data := loadConfig()
	if data == nil {
		return
	}

	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Error parsing connector config: %v", err)
	}

	if len(cfg.Connectors) == 0 {
		log.Println("No connectors defined — nothing to install.")
		return
	}

	destDir := connectors.DefaultConnectorsDir()
	if destDir == "" {
		log.Fatal("Could not determine connectors directory (set CONNECTORS_DIR or ensure a home directory exists)")
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		log.Fatalf("Creating connectors directory %s: %v", destDir, err)
	}

	var failures []string
	for _, entry := range cfg.Connectors {
		name := repoName(entry.Repo)
		if name == "" {
			log.Printf("Error: could not extract name from repo URL %q — skipping", entry.Repo)
			failures = append(failures, entry.Repo)
			continue
		}

		connDir := filepath.Join(destDir, name)
		log.Printf("--- %s ---", name)

		if err := cloneOrUpdate(connDir, entry); err != nil {
			log.Printf("Error: %v — skipping", err)
			failures = append(failures, name)
			continue
		}

		if err := buildConnector(connDir); err != nil {
			log.Printf("Error: %v — skipping", err)
			failures = append(failures, name)
			continue
		}

		if err := verifyConnector(connDir); err != nil {
			log.Printf("Error: %v", err)
			failures = append(failures, name)
			continue
		}

		log.Printf("OK: %s installed successfully", name)
	}

	fmt.Println()
	if len(failures) > 0 {
		log.Fatalf("Failed to install %d connector(s): %s", len(failures), strings.Join(failures, ", "))
	}
	log.Printf("All %d connector(s) installed successfully to %s", len(cfg.Connectors), destDir)
}

// repoName extracts the repo name from a Git URL.
// e.g., "https://github.com/acme/ps-jira-connector" → "ps-jira-connector"
// e.g., "git@github.com:acme/ps-jira-connector.git" → "ps-jira-connector"
func repoName(repoURL string) string {
	// Strip trailing slash first, then .git (handles "repo.git/" correctly).
	name := strings.TrimRight(repoURL, "/")
	name = strings.TrimSuffix(name, ".git")
	// Get the last path segment
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	// Handle SSH-style URLs: git@host:org/repo
	if idx := strings.LastIndex(name, ":"); idx >= 0 {
		name = name[idx+1:]
	}
	return name
}

// cloneOrUpdate clones a repo or updates an existing clone.
func cloneOrUpdate(dir string, entry connectorEntry) error {
	ref := entry.Ref
	if ref == "" {
		ref = "main"
	}

	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		// Repo already exists — fetch and checkout the desired ref.
		log.Printf("  Updating existing clone in %s", dir)
		if err := run(dir, "git", "fetch", "origin"); err != nil {
			return fmt.Errorf("git fetch: %w", err)
		}
		if err := run(dir, "git", "checkout", ref); err != nil {
			return fmt.Errorf("git checkout %s: %w", ref, err)
		}
		if err := run(dir, "git", "pull", "origin", ref); err != nil {
			// pull may fail on detached HEAD (tag/commit) — that's OK after fetch+checkout
			log.Printf("  Warning: git pull failed (may be on a tag/commit): %v", err)
		}
		return nil
	}

	// Fresh clone.
	log.Printf("  Cloning %s → %s (ref: %s)", entry.Repo, dir, ref)
	if err := run(".", "git", "clone", "--branch", ref, "--single-branch", entry.Repo, dir); err != nil {
		// --branch may fail for commit SHAs; fall back to full clone + checkout.
		log.Printf("  Shallow clone failed, trying full clone + checkout...")
		if err := run(".", "git", "clone", entry.Repo, dir); err != nil {
			return fmt.Errorf("git clone: %w", err)
		}
		if err := run(dir, "git", "checkout", ref); err != nil {
			return fmt.Errorf("git checkout %s: %w", ref, err)
		}
	}
	return nil
}

// buildConnector runs "make build" in the connector directory.
func buildConnector(dir string) error {
	log.Printf("  Building connector...")
	if err := run(dir, "make", "build"); err != nil {
		return fmt.Errorf("make build in %s: %w", dir, err)
	}
	return nil
}

// verifyConnector checks that the connector has a valid manifest and executable.
func verifyConnector(dir string) error {
	_, err := connectors.VerifyConnectorDir(dir)
	return err
}

// loadConfig returns the raw JSON config bytes. It checks the
// CUSTOM_CONNECTORS_JSON env var first, then falls back to reading the
// custom-connectors.json file on disk. Returns nil if neither source exists.
func loadConfig() []byte {
	if raw := os.Getenv("CUSTOM_CONNECTORS_JSON"); raw != "" {
		log.Println("Reading connector config from CUSTOM_CONNECTORS_JSON env var")
		return []byte(raw)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("No %s found and CUSTOM_CONNECTORS_JSON not set — nothing to install.", configFile)
			return nil
		}
		log.Fatalf("Error reading %s: %v", configFile, err)
	}
	return data
}

// run executes a command in the given directory, forwarding stdout/stderr.
func run(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

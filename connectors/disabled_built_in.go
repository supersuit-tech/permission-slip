package connectors

import (
	"embed"
	"path"
	"sort"
	"strings"
	"sync"
)

// Embedded marker files for built-in connectors that should not register,
// seed the database, or appear in connector/OAuth listings. Add an empty file
// at connectors/<connector_id>/disabled (this path is next to this .go file).
//
//go:embed kroger/disabled quickbooks/disabled salesforce/disabled
var disabledBuiltInMarkerFS embed.FS

const disabledMarkerFile = "disabled"

var (
	disabledBuiltInMu     sync.RWMutex
	disabledBuiltInIDs    map[string]bool
	disabledBuiltInReason map[string]string
)

func init() {
	ids := make(map[string]bool)
	reasons := make(map[string]string)
	entries, err := disabledBuiltInMarkerFS.ReadDir(".")
	if err != nil {
		disabledBuiltInIDs = ids
		disabledBuiltInReason = reasons
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		data, err := disabledBuiltInMarkerFS.ReadFile(path.Join(id, disabledMarkerFile))
		if err != nil {
			continue
		}
		ids[id] = true
		if s := strings.TrimSpace(string(data)); s != "" {
			reasons[id] = s
		}
	}
	disabledBuiltInIDs = ids
	disabledBuiltInReason = reasons
}

// IsBuiltInConnectorDisabled reports whether the built-in connector with the
// given id is turned off via connectors/<id>/disabled.
func IsBuiltInConnectorDisabled(id string) bool {
	disabledBuiltInMu.RLock()
	defer disabledBuiltInMu.RUnlock()
	return disabledBuiltInIDs[id]
}

// DisabledBuiltInConnectorReason returns optional explanation text from the
// disabled marker file, or "" if the connector is not disabled or the file was empty.
func DisabledBuiltInConnectorReason(id string) string {
	disabledBuiltInMu.RLock()
	defer disabledBuiltInMu.RUnlock()
	return disabledBuiltInReason[id]
}

// DisabledBuiltInConnectorIDs returns connector ids with a disabled marker,
// sorted for stable tests and logs.
func DisabledBuiltInConnectorIDs() []string {
	disabledBuiltInMu.RLock()
	defer disabledBuiltInMu.RUnlock()
	out := make([]string, 0, len(disabledBuiltInIDs))
	for id := range disabledBuiltInIDs {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

package oauth

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	disabledBuiltInOAuthMu  sync.RWMutex
	disabledBuiltInOAuthIDs map[string]bool
)

func init() {
	ids := make(map[string]bool)
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		disabledBuiltInOAuthIDs = ids
		return
	}
	root := filepath.Dir(file)
	connRoot := filepath.Join(root, "..", "connectors")
	entries, err := os.ReadDir(connRoot)
	if err != nil {
		disabledBuiltInOAuthIDs = ids
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		p := filepath.Join(connRoot, id, "disabled")
		b, err := os.ReadFile(p)
		if err != nil {
			if errorsIsNotExist(err) {
				continue
			}
			continue
		}
		if strings.TrimSpace(string(b)) == "" {
			ids[id] = true
		}
	}
	disabledBuiltInOAuthIDs = ids
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}

// IsBuiltInOAuthProviderDisabled reports whether the built-in OAuth provider
// with this id is turned off alongside its connector (empty file at
// connectors/<id>/disabled). Defined here so oauth/providers avoids importing
// connectors (import cycle).
func IsBuiltInOAuthProviderDisabled(id string) bool {
	disabledBuiltInOAuthMu.RLock()
	defer disabledBuiltInOAuthMu.RUnlock()
	return disabledBuiltInOAuthIDs[id]
}

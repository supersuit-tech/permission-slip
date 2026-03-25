package connectors

import (
	"strings"
	"sync"
)

// Registry maps connector IDs to their implementations. It is populated at
// startup and injected into the API handler via Deps. All methods are safe
// for concurrent use.
type Registry struct {
	mu         sync.RWMutex
	connectors map[string]Connector
}

// NewRegistry creates an empty connector registry.
func NewRegistry() *Registry {
	return &Registry{connectors: make(map[string]Connector)}
}

// Register adds a connector to the registry. If a connector with the same ID
// is already registered, it is replaced.
func (r *Registry) Register(c Connector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connectors[c.ID()] = c
}

// Get returns the connector with the given ID, or false if not found.
func (r *Registry) Get(id string) (Connector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.connectors[id]
	return c, ok
}

// IDs returns the IDs of all registered connectors. Used for startup
// validation to detect mismatches between code and database.
func (r *Registry) IDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.connectors))
	for id := range r.connectors {
		ids = append(ids, id)
	}
	return ids
}

// GetAction looks up an action by its full action type string (e.g.,
// "github.create_issue"). It extracts the connector ID from the prefix
// before the first dot and then looks up the action in that connector's
// action map.
func (r *Registry) GetAction(actionType string) (Action, bool) {
	parts := strings.SplitN(actionType, ".", 2)
	if len(parts) != 2 {
		return nil, false
	}
	r.mu.RLock()
	conn, ok := r.connectors[parts[0]]
	r.mu.RUnlock()
	if !ok {
		return nil, false
	}
	action, ok := conn.Actions()[actionType]
	return action, ok
}

// GetActionWithConnector is like GetAction but also returns the owning
// Connector. Used by the approval handler to check connector-level
// interfaces (e.g., ParamValidator) after action-level checks.
func (r *Registry) GetActionWithConnector(actionType string) (Action, Connector, bool) {
	parts := strings.SplitN(actionType, ".", 2)
	if len(parts) != 2 {
		return nil, nil, false
	}
	r.mu.RLock()
	conn, ok := r.connectors[parts[0]]
	r.mu.RUnlock()
	if !ok {
		return nil, nil, false
	}
	action, ok := conn.Actions()[actionType]
	return action, conn, ok
}

// Remove deletes a connector from the registry. Returns false if id was not present.
func (r *Registry) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.connectors[id]; !ok {
		return false
	}
	delete(r.connectors, id)
	return true
}

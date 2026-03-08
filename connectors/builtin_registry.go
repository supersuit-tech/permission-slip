package connectors

import (
	"fmt"
	"sync"
)

// builtInConnectors holds connectors registered via init() in each connector
// package. They are collected here during init and then bulk-registered into
// the real Registry in main.go by iterating over BuiltInConnectors().
var (
	builtInMu         sync.Mutex
	builtInIDs        = make(map[string]bool)
	builtInConnectors []Connector
)

// RegisterBuiltIn adds a connector to the global list of built-in connectors.
// It is called from init() functions in individual connector packages, and
// main.go later registers them by looping over BuiltInConnectors().
//
// Panics if the connector's ID is empty or duplicates an already-registered
// connector, catching wiring mistakes at startup rather than silently allowing
// invalid or shadowed connectors.
func RegisterBuiltIn(c Connector) {
	id := c.ID()
	if id == "" {
		panic("connectors.RegisterBuiltIn: connector ID must not be empty")
	}
	builtInMu.Lock()
	defer builtInMu.Unlock()
	if builtInIDs[id] {
		panic(fmt.Sprintf("connectors.RegisterBuiltIn: duplicate connector ID %q", id))
	}
	builtInIDs[id] = true
	builtInConnectors = append(builtInConnectors, c)
}

// BuiltInConnectors returns all connectors registered via RegisterBuiltIn.
// The returned slice is a copy — callers may iterate freely.
func BuiltInConnectors() []Connector {
	builtInMu.Lock()
	defer builtInMu.Unlock()
	out := make([]Connector, len(builtInConnectors))
	copy(out, builtInConnectors)
	return out
}

// saveAndResetBuiltInConnectors snapshots the global built-in registry and
// resets it to empty. It returns a restore function that puts the original
// state back. This exists solely for unit tests that need isolation without
// permanently destroying the init()-populated state.
func saveAndResetBuiltInConnectors() (restore func()) {
	builtInMu.Lock()
	savedConnectors := builtInConnectors
	savedIDs := builtInIDs
	builtInConnectors = nil
	builtInIDs = make(map[string]bool)
	builtInMu.Unlock()

	return func() {
		builtInMu.Lock()
		builtInConnectors = savedConnectors
		builtInIDs = savedIDs
		builtInMu.Unlock()
	}
}

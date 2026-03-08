package connectors

import "sync"

// builtInConnectors holds connectors registered via init() in each connector
// package. They are collected here during init and then bulk-registered into
// the real Registry in main.go via RegisterAllBuiltIn.
var (
	builtInMu         sync.Mutex
	builtInConnectors []Connector
)

// RegisterBuiltIn adds a connector to the global list of built-in connectors.
// It is called from init() functions in individual connector packages.
//
// Panics if the connector's ID is empty to catch registration mistakes at
// startup rather than silently allowing an invalid connector.
func RegisterBuiltIn(c Connector) {
	if c.ID() == "" {
		panic("connectors.RegisterBuiltIn: connector ID must not be empty")
	}
	builtInMu.Lock()
	defer builtInMu.Unlock()
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

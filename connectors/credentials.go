package connectors

import "fmt"

// Credentials holds decrypted service credentials. It implements
// json.Marshaler and fmt.Stringer to prevent accidental exposure in
// logs, error messages, or serialized responses.
type Credentials struct {
	values map[string]string
}

// NewCredentials creates a Credentials value from a plain map.
// The map is copied so the caller cannot mutate credentials after construction.
func NewCredentials(m map[string]string) Credentials {
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return Credentials{values: cp}
}

// Get returns the credential value for the given key.
func (c Credentials) Get(key string) (string, bool) {
	v, ok := c.values[key]
	return v, ok
}

// Keys returns the credential key names (without values).
func (c Credentials) Keys() []string {
	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	return keys
}

// String redacts credential values to prevent accidental logging.
func (c Credentials) String() string {
	return fmt.Sprintf("Credentials{keys: %v}", c.Keys())
}

// GoString redacts credential values in %#v formatting.
func (c Credentials) GoString() string {
	return c.String()
}

// ToMap returns a copy of the underlying credential key-value pairs.
// This is intended for trusted internal use (e.g., passing credentials to an
// external connector subprocess). The returned map is a copy; mutations do not
// affect the original Credentials.
func (c Credentials) ToMap() map[string]string {
	cp := make(map[string]string, len(c.values))
	for k, v := range c.values {
		cp[k] = v
	}
	return cp
}

// MarshalJSON redacts credential values to prevent accidental serialization.
func (c Credentials) MarshalJSON() ([]byte, error) {
	return []byte(`"[REDACTED]"`), nil
}

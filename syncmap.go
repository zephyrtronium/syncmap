// Package syncmap implements a concurrent read-mostly map, much like sync.Map.
package syncmap

import "sync"

// Map is a concurrent read-mostly map, much like sync.Map with string keys.
type Map struct {
	mu sync.Mutex
	v  map[string]interface{}
}

// Store sets the value at a key.
func (m *Map) Store(k string, v interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.v == nil {
		m.v = make(map[string]interface{})
	}
	m.v[k] = v
}

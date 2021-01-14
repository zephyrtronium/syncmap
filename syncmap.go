// Package syncmap implements a concurrent read-mostly map, much like sync.Map.
package syncmap

import (
	"sync"
	"sync/atomic"
)

// Map is a concurrent read-mostly map, much like sync.Map with string keys.
type Map struct {
	v  atomic.Value // map[string]interface{}
	mu sync.Mutex
}

// Store sets the value at a key.
func (m *Map) Store(k string, v interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mv, _ := m.v.Load().(map[string]interface{})
	if mv == nil {
		mv = make(map[string]interface{})
		m.v.Store(mv)
	}
	mv[k] = v
}

// Load gets the value at a key. ok is false if the key was not in the map.
func (m *Map) Load(k string) (v interface{}, ok bool) {
	mv, _ := m.v.Load().(map[string]interface{})
	v, ok = mv[k]
	return v, ok
}

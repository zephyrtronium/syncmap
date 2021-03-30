// Package syncmap implements a concurrent read-mostly map, much like sync.Map.
package syncmap

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// Map is a concurrent read-mostly map, much like sync.Map with string keys.
type Map struct {
	v atomic.Value // map[string]*entry

	// mu must be held when using dirty or misses.
	mu     sync.Mutex
	dirty  map[string]*entry
	misses int
}

// Store sets the value at a key.
func (m *Map) Store(k string, v interface{}) {
	mv, _ := m.v.Load().(map[string]*entry)
	e := mv[k]
	if e == nil {
		m.mu.Lock()
		defer m.mu.Unlock()
		// Reload e in case another goroutine set it while we were locking.
		mv, _ = m.v.Load().(map[string]*entry)
		e = mv[k]
		if e == nil {
			e = m.dirty[k]
			if e == nil {
				m.miss() // Ensures m.dirty is non-nil.
				m.dirty[k] = newEntry(v)
				return
			}
		}
	}
	e.store(v)
}

// Load gets the value at a key. ok is false if the key was not in the map.
func (m *Map) Load(k string) (v interface{}, ok bool) {
	mv, _ := m.v.Load().(map[string]*entry)
	e, ok := mv[k]
	if !ok {
		m.mu.Lock()
		defer m.mu.Unlock()
		// Reload e in case another goroutine set it while we were locking.
		mv, _ = m.v.Load().(map[string]*entry)
		e, ok = mv[k]
		if !ok {
			e, ok = m.dirty[k]
			m.miss()
		}
	}
	return e.load()
}

// LoadOrStore gets the value at a key if it exists or stores and returns v if
// it does not. loaded is true if the value already existed.
func (m *Map) LoadOrStore(k string, v interface{}) (r interface{}, loaded bool) {
	mv, _ := m.v.Load().(map[string]*entry)
	e, ok := mv[k]
	if ok {
		return e.load()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// Reload e in case another goroutine set it while we were locking.
	mv, _ = m.v.Load().(map[string]*entry)
	e, ok = mv[k]
	if ok {
		return e.load()
	}
	e, ok = m.dirty[k]
	// Whether we load or store, this is a miss.
	m.miss()
	if ok {
		return e.load()
	}
	m.dirty[k] = newEntry(v)
	return v, false
}

// Delete deletes the value at a key.
func (m *Map) Delete(k string) {
	mv, _ := m.v.Load().(map[string]*entry)
	e := mv[k]
	if e != nil {
		e.delete()
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// Reload e in case another goroutine set it while we were locking.
	mv, _ = m.v.Load().(map[string]*entry)
	e = mv[k]
	if e != nil {
		e.delete()
		return
	}
	delete(m.dirty, k)
	m.miss()
}

// miss updates the miss counter and possibly promotes the dirty map. The
// caller must hold m.mu.
func (m *Map) miss() {
	m.misses++
	if m.misses < len(m.dirty) {
		return
	}
	mv := m.dirty
	m.v.Store(mv)
	m.dirty = make(map[string]*entry, len(mv))
	for k, v := range mv {
		if atomic.LoadPointer(&v.p) != nil {
			m.dirty[k] = v
		}
	}
	m.misses = 0
}

type entry struct {
	p unsafe.Pointer // *interface{}
}

func newEntry(v interface{}) *entry {
	return &entry{unsafe.Pointer(&v)}
}

func (e *entry) load() (interface{}, bool) {
	if e == nil {
		return nil, false
	}
	p := atomic.LoadPointer(&e.p)
	if p == nil {
		// Nil means deleted.
		return nil, false
	}
	return *(*interface{})(p), true
}

func (e *entry) store(v interface{}) {
	atomic.StorePointer(&e.p, unsafe.Pointer(&v))
}

func (e *entry) delete() {
	atomic.StorePointer(&e.p, nil)
}

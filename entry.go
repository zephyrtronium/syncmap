package syncmap

import (
	"sync/atomic"
	"unsafe"
)

// entry is a value in a Map. All uses of its fields must be atomic.
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

func (e *entry) delete() (old *interface{}) {
	return (*interface{})(atomic.SwapPointer(&e.p, nil))
}

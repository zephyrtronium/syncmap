package syncmap_test

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/zephyrtronium/syncmap"
)

func TestMap(t *testing.T) {
	var m syncmap.Map
	keys := strings.Fields("a b c d e f g h i j k l m n o p q r s t u v w x y z")
	ch := make(chan error, len(keys))
	start := make(chan bool)
	stop := make(chan bool)
	for _, k := range keys {
		go func(k string) {
			<-start
			p, ok := m.Load(k)
			if ok {
				ch <- fmt.Errorf("unexpected successful initial load of key %q", k)
			}
			if p != nil {
				ch <- fmt.Errorf("initial load of key %q produced nonzero value %#v", k, p)
			}
			m.Store(k, -1)
			for i := 0; i < 1e4; i++ {
				p, ok := m.Load(k)
				if !ok {
					ch <- fmt.Errorf("load %d of key %q failed", i, k)
				}
				if p != i-1 {
					ch <- fmt.Errorf("load %d of key %q gave wrong result: want %d, got %v", i, k, i-1, p)
				}
				m.Store(k, i)
			}
			ch <- nil
		}(k)
	}
	go func() {
		<-start
		for {
			select {
			case <-stop:
				return
			default:
				m.Load("nonexistent key to trigger frequent dirty map promotions")
			}
		}
	}()
	close(start)
	for i := 0; i < len(keys); i++ {
		err := <-ch
		if err != nil {
			t.Error(err)
			i-- // don't count errors, only nils
		}
	}
	close(stop)
}

func TestStoreOnly(t *testing.T) {
	var m syncmap.Map
	var wg sync.WaitGroup
	ch := make(chan bool)
	n := runtime.GOMAXPROCS(0)
	wg.Add(n)
	f := func(v int) {
		k := string(rune(v + '0'))
		<-ch
		for i := 0; i < 1e5; i++ {
			m.Store(k, v)
		}
		wg.Done()
	}
	for i := 0; i < n; i++ {
		go f(i)
	}
	close(ch)
	wg.Wait()
	// No correct result, except that the race detector shouldn't complain.
}

func TestMapKey(t *testing.T) {
	var m syncmap.Map
	var wg sync.WaitGroup
	n := runtime.GOMAXPROCS(0)
	wg.Add(n)
	if _, ok := m.Load("key"); ok {
		t.Fatal("key exists in zero value")
	}
	ch := make(chan bool)
	f := func(v int) {
		<-ch
		for i := 0; i < 1e5; i++ {
			m.Load("key")
			m.Store("key", v)
		}
		wg.Done()
	}
	for i := 0; i < n; i++ {
		go f(i)
	}
	close(ch)
	wg.Wait()
	// No correct result, except that the race detector shouldn't complain.
}

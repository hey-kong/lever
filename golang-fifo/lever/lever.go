package lever

import (
	"container/list"
	"sync"

	"github.com/hey-kong/lever/golang-fifo"
)

// entry holds the key and value of a cache entry.
type entry[K comparable, V any] struct {
	key     K
	value   V
	visited bool
}

type Lever[K comparable, V any] struct {
	lock  sync.RWMutex
	size  int
	items map[K]*list.Element
	ll    *list.List
	fast  *list.Element
	slow  *list.Element
}

func New[K comparable, V any](size int) fifo.Cache[K, V] {
	return &Lever[K, V]{
		size:  size,
		items: make(map[K]*list.Element),
		ll:    list.New(),
	}
}

func (s *Lever[K, V]) Set(key K, value V) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if e, ok := s.items[key]; ok {
		e.Value.(*entry[K, V]).value = value
		e.Value.(*entry[K, V]).visited = true
		return
	}

	if s.ll.Len() >= s.size {
		s.evict()
	}
	e := &entry[K, V]{key: key, value: value}
	s.items[key] = s.ll.PushFront(e)
}

func (s *Lever[K, V]) Get(key K) (value V, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if e, ok := s.items[key]; ok {
		e.Value.(*entry[K, V]).visited = true
		return e.Value.(*entry[K, V]).value, true
	}

	return
}

func (s *Lever[K, V]) Contains(key K) (ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, ok = s.items[key]
	return
}

func (s *Lever[K, V]) Peek(key K) (value V, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if e, ok := s.items[key]; ok {
		return e.Value.(*entry[K, V]).value, true
	}

	return
}

func (s *Lever[K, V]) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.ll.Len()
}

func (s *Lever[K, V]) Purge() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.items = make(map[K]*list.Element)
	s.ll = list.New()
}

func (s *Lever[K, V]) evict() {
	if s.slow == nil {
		s.slow = s.ll.Back()
	}
	if s.fast == nil {
		s.fast = s.ll.Back()
	}

	var o *list.Element
	for i := 0; i < 2; i++ {
		o, s.fast = s.fast, s.fast.Prev()
		if o.Value.(*entry[K, V]).visited {
			o.Value.(*entry[K, V]).visited = false
			s.ll.MoveAfter(o, s.slow)
		}
		if s.fast == nil {
			break
		}
	}

	o, s.slow = s.slow, s.slow.Prev()
	if o.Value.(*entry[K, V]).visited {
		o.Value.(*entry[K, V]).visited = false
		// FIFO demotion
		o = s.ll.Back()
		delete(s.items, o.Value.(*entry[K, V]).key)
		s.ll.Remove(o)
	}
	delete(s.items, o.Value.(*entry[K, V]).key)
	s.ll.Remove(o)
}

func (s *Lever[K, V]) removeElement(o *list.Element) {
	if s.slow == o {
		panic("lever: evicting illegal element")
	}
	if s.fast == o {
		s.fast = s.fast.Prev()
	}

	delete(s.items, o.Value.(*entry[K, V]).key)
	s.ll.Remove(o)
}

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
	old     bool
}

type Lever[K comparable, V any] struct {
	lock  sync.RWMutex
	size  int
	items map[K]*list.Element
	ll    *list.List
	hand  *list.Element
	left  int
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
		if !e.Value.(*entry[K, V]).old {
			if e == s.hand {
				s.hand = s.hand.Prev()
				s.left--
			}
			s.ll.MoveToFront(e)
		}
		e.Value.(*entry[K, V]).value = value
		e.Value.(*entry[K, V]).visited = true
		return
	}

	if s.ll.Len() >= s.size {
		s.evict()
	}
	e := &entry[K, V]{key: key, value: value}
	s.items[key] = s.ll.PushFront(e)
	s.left++
}

func (s *Lever[K, V]) Get(key K) (value V, ok bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if e, ok := s.items[key]; ok {
		if !e.Value.(*entry[K, V]).old {
			if e == s.hand {
				s.hand = s.hand.Prev()
				s.left--
			}
			s.ll.MoveToFront(e)
		}
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
	o := s.hand
	// if o is nil, then assign it to the tail element in the list
	if o == nil {
		o = s.ll.Back()
		s.left = s.size
	}

	for o.Value.(*entry[K, V]).visited {
		o.Value.(*entry[K, V]).visited = false
		o.Value.(*entry[K, V]).old = true
		o = o.Prev()
		s.left--
		if s.left <= s.size/25 {
			// reset
			o = s.ll.Back()
			s.left = s.size
		}
	}

	s.hand = o.Prev()
	s.left--
	delete(s.items, o.Value.(*entry[K, V]).key)
	s.ll.Remove(o)
}

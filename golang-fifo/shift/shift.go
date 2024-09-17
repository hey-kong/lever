package shift

import (
	"sync"

	"github.com/hey-kong/shift/golang-fifo"
	"github.com/hey-kong/shift/golang-fifo/shift/list"
)

// entry holds the key and value of a cache entry.
type entry[K comparable, V any] struct {
	key     K
	value   V
	visited bool
}

type Shift[K comparable, V any] struct {
	lock       sync.RWMutex
	size       int
	items      map[K]*list.Element
	eviction   *list.List
	retention  *list.List
	insertMark *list.Element
}

func New[K comparable, V any](size int) fifo.Cache[K, V] {
	return &Shift[K, V]{
		size:      size,
		items:     make(map[K]*list.Element),
		eviction:  list.New(),
		retention: list.New(),
	}
}

func (s *Shift[K, V]) Set(key K, value V) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if e, ok := s.items[key]; ok {
		if e.List() == s.retention && !e.Value.(*entry[K, V]).visited {
			s.retention.MoveToFront(e)
		}
		e.Value.(*entry[K, V]).value = value
		e.Value.(*entry[K, V]).visited = true
		return
	}

	if s.eviction.Len()+s.retention.Len() >= s.size {
		s.evict()
	}
	e := &entry[K, V]{key: key, value: value}
	if s.insertMark == nil {
		s.items[key] = s.eviction.PushFront(e)
	} else {
		s.items[key] = s.retention.InsertAfter(e, s.insertMark)
	}

}

func (s *Shift[K, V]) Get(key K) (value V, ok bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if e, ok := s.items[key]; ok {
		if e.List() == s.retention {
			if !e.Value.(*entry[K, V]).visited {
				s.retention.MoveToFront(e)
			}
		}
		e.Value.(*entry[K, V]).visited = true
		return e.Value.(*entry[K, V]).value, true
	}

	return
}

func (s *Shift[K, V]) Contains(key K) (ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, ok = s.items[key]
	return
}

func (s *Shift[K, V]) Peek(key K) (value V, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if e, ok := s.items[key]; ok {
		return e.Value.(*entry[K, V]).value, true
	}

	return
}

func (s *Shift[K, V]) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.eviction.Len() + s.retention.Len()
}

func (s *Shift[K, V]) Purge() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.items = make(map[K]*list.Element)
	s.eviction = list.New()
	s.retention = list.New()
}

func (s *Shift[K, V]) evict() {
	evicted := false
	for s.eviction.Len() > 0 && !evicted {
		o := s.eviction.Back()
		key := o.Value.(*entry[K, V]).key
		if o.Value.(*entry[K, V]).visited {
			o.Value.(*entry[K, V]).visited = false
			s.eviction.Remove(o)
			s.items[o.Value.(*entry[K, V]).key] = s.retention.PushFront(o.Value)
		} else {
			evicted = true
			delete(s.items, key)
			s.eviction.Remove(o)
		}
		if s.eviction.Len() == 0 {
			s.eviction, s.retention = s.retention, s.eviction
			s.insertMark = nil
		}
	}

	// if the eviction queue size is less than 2% of the total cache size
	// find the most recently visited entry in the retention queue and set it as insertMark
	// insert the new entry into the retention queue after the insertMark pointer
	if s.eviction.Len() < s.size/50 && s.insertMark == nil {
		s.insertMark = s.retention.Back()
		for s.insertMark != nil && !s.insertMark.Value.(*entry[K, V]).visited {
			s.insertMark = s.insertMark.Prev()
		}
	}
}

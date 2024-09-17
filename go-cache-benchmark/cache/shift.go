package cache

import (
	fifo "github.com/hey-kong/shift/golang-fifo"
	"github.com/hey-kong/shift/golang-fifo/shift"
)

type Shift struct {
	v fifo.Cache[string, any]
}

func NewShift(size int) Cache {
	return &Shift{shift.New[string, any](size)}
}

func (s *Shift) Name() string {
	return "shift"
}

func (s *Shift) Get(key string) bool {
	_, ok := s.v.Get(key)
	return ok
}

func (s *Shift) Set(key string) {
	s.v.Set(key, key)
}

func (s *Shift) Close() {

}

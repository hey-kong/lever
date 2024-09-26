package cache

import (
	fifo "github.com/hey-kong/shift/golang-fifo"
	"github.com/hey-kong/shift/golang-fifo/slru"
)

type SLRU struct {
	v fifo.Cache[string, any]
}

func NewSLRU(size int) Cache {
	return &SLRU{slru.New[string, any](size)}
}

func (s *SLRU) Name() string {
	return "slru"
}

func (s *SLRU) Get(key string) bool {
	_, ok := s.v.Get(key)
	return ok
}

func (s *SLRU) Set(key string) {
	s.v.Set(key, key)
}

func (s *SLRU) Close() {

}

package cache

import (
	fifo "github.com/hey-kong/shakesieve/golang-fifo"
	shakesieve "github.com/hey-kong/shakesieve/golang-fifo/shakesieve"
)

type ShakeSieve struct {
	v fifo.Cache[string, any]
}

func NewShiftSieve(size int) Cache {
	return &ShakeSieve{shakesieve.New[string, any](size)}
}

func (s *ShakeSieve) Name() string {
	return "shakesieve"
}

func (s *ShakeSieve) Get(key string) bool {
	_, ok := s.v.Get(key)
	return ok
}

func (s *ShakeSieve) Set(key string) {
	s.v.Set(key, key)
}

func (s *ShakeSieve) Close() {

}

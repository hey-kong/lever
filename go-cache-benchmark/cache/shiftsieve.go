package cache

import (
	fifo "github.com/hey-kong/shiftsieve/golang-fifo"
	shiftsieve "github.com/hey-kong/shiftsieve/golang-fifo/shiftsieve"
)

type ShiftSieve struct {
	v fifo.Cache[string, any]
}

func NewShiftSieve(size int) Cache {
	return &ShiftSieve{shiftsieve.New[string, any](size)}
}

func (s *ShiftSieve) Name() string {
	return "shiftsieve"
}

func (s *ShiftSieve) Get(key string) bool {
	_, ok := s.v.Get(key)
	return ok
}

func (s *ShiftSieve) Set(key string) {
	s.v.Set(key, key)
}

func (s *ShiftSieve) Close() {

}

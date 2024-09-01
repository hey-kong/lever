package cache

import (
	fifo "github.com/hey-kong/lever/golang-fifo"
	lever "github.com/hey-kong/lever/golang-fifo/lever"
)

type Lever struct {
	v fifo.Cache[string, any]
}

func NewLever(size int) Cache {
	return &Lever{lever.New[string, any](size)}
}

func (s *Lever) Name() string {
	return "lever"
}

func (s *Lever) Get(key string) bool {
	_, ok := s.v.Get(key)
	return ok
}

func (s *Lever) Set(key string) {
	s.v.Set(key, key)
}

func (s *Lever) Close() {

}

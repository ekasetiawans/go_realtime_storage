package realtimedb

import (
	"context"
	"sync"
)

type stream struct {
	lock        sync.Mutex
	subscribers map[chan *event]context.Context
}

func newStream() *stream {
	result := &stream{}
	result.subscribers = make(map[chan *event]context.Context)
	return result
}

func (s *stream) get(context context.Context) chan *event {
	s.lock.Lock()
	defer s.lock.Unlock()

	res := make(chan *event)
	s.subscribers[res] = context

	go func() {
		<-context.Done()
		s.lock.Lock()
		defer s.lock.Unlock()

		delete(s.subscribers, res)
		close(res)
	}()

	return res
}

func (s *stream) add(path string, value interface{}) {
	s.send("add", path, value)
}

func (s *stream) update(path string, value interface{}) {
	s.send("update", path, value)
}

func (s *stream) delete(path string, value interface{}) {
	s.send("delete", path, value)
}

func (s *stream) send(name, path string, value interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ev := &event{
		Name:  name,
		Path:  path,
		Value: value,
	}

	for c := range s.subscribers {
		c <- ev
	}
}

type event struct {
	Name  string      `json:"event"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

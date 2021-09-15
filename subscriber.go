package main

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type stream struct {
	lock      sync.Mutex
	ws        map[*websocket.Conn][]string
	listeners map[string]map[*websocket.Conn]bool
}

func newStream() *stream {
	result := &stream{}
	result.listeners = make(map[string]map[*websocket.Conn]bool)
	result.ws = make(map[*websocket.Conn][]string)

	return result
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

func (s *stream) send(event, path string, value interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	l := s.listeners[path]
	if l == nil {
		return
	}

	for ws := range l {
		err := ws.WriteJSON(map[string]interface{}{
			"event": event,
			"path":  path,
			"value": value,
		})

		if err != nil {
			log.Println(err)
			continue
		}
	}
}

func (s *stream) listen(path string, ws *websocket.Conn) {
	s.lock.Lock()
	defer s.lock.Unlock()

	l := s.listeners[path]
	if l == nil {
		l = make(map[*websocket.Conn]bool)
	}

	l[ws] = true
	s.listeners[path] = l

	paths := s.ws[ws]
	if paths == nil {
		paths = make([]string, 0)
	}

	paths = append(paths, path)
	s.ws[ws] = paths
}

func (s *stream) cancelAll(ws *websocket.Conn) {
	s.lock.Lock()
	defer s.lock.Unlock()

	l := s.ws[ws]
	if l == nil {
		return
	}

	for _, path := range l {
		l := s.listeners[path]
		if l == nil {
			continue
		}

		delete(l, ws)
	}
	delete(s.ws, ws)
}

func (s *stream) cancel(path string, ws *websocket.Conn) {
	s.lock.Lock()
	defer s.lock.Unlock()

	l := s.listeners[path]
	if l == nil {
		return
	}

	delete(l, ws)
}

package realtimedb

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func handleWebsocket(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.Status(500)
		return
	}

	stream := c.MustGet("stream").(*stream)
	lock := sync.Mutex{}
	subscriptions := make(map[string]chan *event)

	defer func() {
		for _, sub := range subscriptions {
			close(sub)
		}
	}()

	go func() {
		c := stream.get(c.Request.Context())
		for event := range c {
			func() {
				lock.Lock()
				defer lock.Unlock()
				if sub, ok := subscriptions[event.path]; ok {
					sub <- event
				}
			}()
		}
	}()

	for {
		data := map[string]interface{}{}
		err := ws.ReadJSON(&data)
		if err != nil {
			break
		}

		command := data["command"].(string)
		switch command {
		case "sub":
			func() {
				path := data["path"].(string)
				if _, ok := subscriptions[path]; !ok {
					lock.Lock()
					defer lock.Unlock()
					sub := make(chan *event)
					subscriptions[path] = sub

					go func() {
						for event := range sub {
							ws.WriteJSON(event.value)
						}
					}()
				}
			}()

		case "unsub":
			func() {
				path := data["path"].(string)
				if sub, ok := subscriptions[path]; !ok {
					lock.Lock()
					defer lock.Unlock()

					close(sub)
					delete(subscriptions, path)
				}
			}()
		}
	}
}

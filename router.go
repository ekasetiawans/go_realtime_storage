package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	router   = createRoute()
	client   = getClient()
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func createRoute() *gin.Engine {
	res := gin.Default()
	stream := newStream()
	res.Use(func(c *gin.Context) {
		c.Set("stream", stream)
	})

	return res
}

package realtimedb

import (
	"github.com/gin-gonic/gin"
)

var (
	router = createRoute()
)

func createRoute() *gin.Engine {
	res := gin.Default()
	stream := newStream()
	client := getClient()

	res.Use(func(c *gin.Context) {
		c.Set("stream", stream)
		c.Set("dbClient", client)
	})

	return res
}

type RealtimeStorage struct {
}

func (r *RealtimeStorage) Run(address string) error {
	return router.Run(address)
}

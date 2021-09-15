package realtimedb

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

func createRoute() *gin.Engine {
	res := gin.Default()
	return res
}

type RealtimeStorage struct {
	srv *http.Server
}

func (r *RealtimeStorage) Stop(ctx context.Context) error {
	return r.srv.Shutdown(ctx)
}

func (r *RealtimeStorage) Run(address string) error {
	stream := newStream()
	client := getClient()

	router := createRoute()

	router.Use(func(c *gin.Context) {
		c.Set("stream", stream)
		c.Set("dbClient", client)
	})

	initDatabaseRouter(router)
	initStorageRouter(router)

	r.srv = &http.Server{
		Addr:    address,
		Handler: router,
	}

	return r.srv.ListenAndServe()
}

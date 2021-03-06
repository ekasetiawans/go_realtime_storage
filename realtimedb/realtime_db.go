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
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "*")
		c.Header("Access-Control-Allow-Methods", "*")
		c.Next()
	})

	router.Use(func(c *gin.Context) {
		c.Set("stream", stream)
		c.Set("dbClient", client)
		c.Next()
	})

	router.Use(func(c *gin.Context) {
		token := c.Request.Header.Get("authorization")
		if token != "" {
			token = token[7:]
		}

		if token != "" {
			user, err := validateJwt(token)
			if err != nil {
				c.Status(401)
				c.Abort()
				return
			}

			c.Set("user", user)
		}

		c.Next()
	})

	initDatabaseRouter(router)
	initStorageRouter(router)
	initAuthRouter(router)

	r.srv = &http.Server{
		Addr:    address,
		Handler: router,
	}

	return r.srv.ListenAndServe()
}

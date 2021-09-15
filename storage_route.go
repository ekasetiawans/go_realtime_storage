package main

import (
	"io/ioutil"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	storageRoute := router.Group("/storage/:database", func(c *gin.Context) {
		databaseName := c.Param("database")
		db := client.Database(databaseName)
		c.Set("db", db)

		bucket, err := gridfs.NewBucket(db)
		if err != nil {
			c.Status(500)
			return
		}

		c.Set("bucket", bucket)
	})

	storageRoute.GET("*path", func(c *gin.Context) {
		docPath := c.Param("path")
		bucket := c.MustGet("bucket").(*gridfs.Bucket)

		downStream, err := bucket.OpenDownloadStreamByName(docPath)
		if err != nil {
			c.Status(404)
			return
		}

		defer downStream.Close()
		cType := downStream.GetFile().Metadata.Lookup("contentType")
		cTypeString := cType.String()
		c.DataFromReader(200, downStream.GetFile().Length, cTypeString, downStream, nil)
	})

	storageRoute.POST("*path", func(c *gin.Context) {
		docPath := c.Param("path")
		file, err := c.FormFile("file")
		if err != nil {
			c.Status(500)
			return
		}

		bucket := c.MustGet("bucket").(*gridfs.Bucket)
		cur, err := bucket.Find(bson.M{
			"filename": bson.M{
				"$eq": docPath,
			},
		})
		if err != nil {
			c.Status(500)
			return
		}

		defer cur.Close(c.Request.Context())
		if cur.Next(c.Request.Context()) {
			c.Status(400)
			return
		}

		upStream, err := bucket.OpenUploadStream(docPath, &options.UploadOptions{
			Metadata: map[string]interface{}{
				"originalFileName": file.Filename,
				"originalFileSize": file.Size,
				"contentType":      file.Header.Get("Content-Type"),
			},
		})

		if err != nil {
			c.Status(500)
			return
		}

		defer upStream.Close()

		mfile, err := file.Open()
		if err != nil {
			c.Status(500)
			return
		}
		defer mfile.Close()

		bytes, err := ioutil.ReadAll(mfile)
		if err != nil {
			c.Status(500)
			return
		}

		_, err = upStream.Write(bytes)
		if err != nil {
			c.Status(500)
			return
		}

		c.JSON(200, map[string]interface{}{
			"success": true,
			"id":      upStream.FileID.(primitive.ObjectID).Hex(),
		})

	})

	storageRoute.DELETE("*path", func(c *gin.Context) {
		docPath := c.Param("path")
		bucket := c.MustGet("bucket").(*gridfs.Bucket)
		cur, err := bucket.Find(bson.M{
			"filename": bson.M{
				"$eq": docPath,
			},
		})
		if err != nil {
			c.Status(500)
			return
		}

		defer cur.Close(c.Request.Context())
		if cur.Next(c.Request.Context()) {
			val := cur.Current
			id := val.Lookup("_id").ObjectID()
			err := bucket.Delete(id)
			if err != nil {
				c.Status(500)
				return
			}

			c.Status(200)
			return
		}

		c.Status(404)
	})
}

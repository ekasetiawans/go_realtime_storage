package realtimedb

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func initDatabaseRouter(router *gin.Engine) {
	databaseRoute := router.Group("/database/:database", func(c *gin.Context) {
		databaseName := c.Param("database")
		client := c.MustGet("dbClient").(*mongo.Client)

		db := client.Database(databaseName)
		c.Set("db", db)
	})

	databaseRoute.Any("*path", func(c *gin.Context) {
		db := c.MustGet("db").(*mongo.Database)
		if c.Request.Method == "GET" && c.IsWebsocket() {
			handleWebsocket(c)
			return
		}

		path := c.Param("path")
		segments := strings.Split(path, "/")
		segments = segments[1:]
		if segments[len(segments)-1] == "" {
			segments = segments[:len(segments)-1]
		}

		//ensure all docId valid
		for i := 1; i < len(segments); i += 2 {
			docIdHex := segments[i]
			_, err := primitive.ObjectIDFromHex(docIdHex)
			if err != nil {
				c.Status(404)
				return
			}
		}

		objectPath := strings.Join(segments, "/")
		c.Set("objectPath", objectPath)

		segmentLength := len(segments)
		if segmentLength%2 == 0 {
			// path is document
			collectionPath := strings.Join(segments[:len(segments)-1], "/")
			collection, err := getCollection(c.Request.Context(), collectionPath, db)
			if err != nil {
				c.Status(500)
				return
			}

			documentId, err := primitive.ObjectIDFromHex(segments[len(segments)-1])
			if err != nil {
				c.Status(404)
				return
			}

			c.Set("collectionPath", collectionPath)
			c.Set("collection", collection)
			c.Set("documentId", documentId)
			handleDocumentRequest(c)
		} else {
			// path is collection
			collectionPath := strings.Join(segments, "/")
			collection, err := getCollection(c.Request.Context(), collectionPath, db)
			if err != nil {
				c.Status(500)
				return
			}

			c.Set("collectionPath", collectionPath)
			c.Set("collection", collection)
			handleCollectionRequest(c)
		}
	})
}

func handleCollectionRequest(c *gin.Context) {
	collection := c.MustGet("collection").(*mongo.Collection)
	stream := c.MustGet("stream").(*stream)

	switch c.Request.Method {
	case "GET":
		// get all document in this collection

		query := c.Query("q")
		var filter interface{}
		if query != "" {
			jb, err := base64.StdEncoding.DecodeString(query)
			if err == nil {
				err := json.Unmarshal(jb, &filter)
				if err != nil {
					c.Status(402)
					return
				}
			}
		} else {
			filter = gin.H{}
		}

		cur, err := collection.Find(c.Request.Context(), filter)
		if err != nil {
			c.Status(500)
			return
		}

		defer cur.Close(c.Request.Context())
		results := make([]map[string]interface{}, 0)
		cur.All(c.Request.Context(), &results)

		c.JSON(200, results)

	case "POST":
		// add document to this collection
		data := make(map[string]interface{})
		err := c.Bind(&data)
		if err != nil {
			return
		}

		data["_created_at"] = time.Now()
		data["_updated_at"] = nil
		data["_deleted_at"] = nil
		res, err := collection.InsertOne(c.Request.Context(), data)
		if err != nil {
			c.Status(500)
			return
		}

		result := collection.FindOne(c.Request.Context(), bson.M{"_id": res.InsertedID})
		doc := make(map[string]interface{})
		err = result.Decode(&doc)
		if err != nil {
			c.Status(500)
			return
		}

		c.JSON(200, doc)
		path := c.MustGet("objectPath").(string)
		stream.add(path, doc)

	case "DELETE":
		// drop this collection
		err := dropCollections(c.Request.Context(), collection, c.MustGet("db").(*mongo.Database), stream)
		if err != nil {
			c.Status(500)
			return
		}

		c.Status(200)
	}
}

func handleDocumentRequest(c *gin.Context) {
	documentId := c.MustGet("documentId").(primitive.ObjectID)
	collection := c.MustGet("collection").(*mongo.Collection)
	documentPath := c.MustGet("objectPath").(string)
	stream := c.MustGet("stream").(*stream)

	switch c.Request.Method {
	case "GET":
		// get document by id
		result := collection.FindOne(c.Request.Context(), bson.M{"_id": documentId})
		doc := make(map[string]interface{})
		err := result.Decode(&doc)
		if err != nil {
			c.Status(404)
			return
		}

		c.JSON(200, doc)

	case "PUT":
		// update this document
		data := make(map[string]interface{})
		err := c.Bind(&data)
		if err != nil {
			return
		}

		data["_updated_at"] = time.Now()
		delete(data, "_id")

		update := bson.D{{Key: "$set", Value: data}}
		_, err = collection.UpdateByID(c.Request.Context(), documentId, update)
		if err != nil {
			c.Status(404)
			return
		}

		result := collection.FindOne(c.Request.Context(), bson.M{"_id": documentId})
		doc := make(map[string]interface{})
		result.Decode(&doc)
		c.JSON(200, doc)
		stream.update(documentPath, doc)

		collectionPath := c.MustGet("collectionPath").(string)
		stream.update(collectionPath, doc)
		return

	case "DELETE":
		// delete this document
		res, err := collection.DeleteOne(c.Request.Context(), bson.M{"_id": documentId})
		if err != nil {
			c.Status(500)
			return
		}

		if res.DeletedCount == 0 {
			c.Status(404)
			return
		}

		c.Status(200)
		deleted := map[string]interface{}{
			"documentId": documentId.Hex(),
		}
		stream.delete(documentPath, deleted)

		collectionPath := c.MustGet("collectionPath").(string)
		stream.delete(collectionPath, deleted)
	}
}

package realtimedb

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getClient() *mongo.Client {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	clientOptions.Auth = &options.Credential{
		Username: "root",
		Password: "123456",
	}

	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func getCollection(context context.Context, path string, db *mongo.Database) (*mongo.Collection, error) {
	metaCollections := db.Collection("__meta_collection__")

	var collectionId primitive.ObjectID
	collectionRes := metaCollections.FindOne(context, bson.M{"path": path})
	meta := make(map[string]interface{})
	err := collectionRes.Decode(&meta)

	if err != nil {
		insertResult, err := metaCollections.InsertOne(context, bson.M{
			"path": path,
		})
		if err != nil {
			return nil, err
		}

		collectionId = insertResult.InsertedID.(primitive.ObjectID)
	} else {
		collectionId = meta["_id"].(primitive.ObjectID)
	}

	collectionName := collectionId.Hex()
	realCollection := db.Collection(collectionName)
	return realCollection, nil
}

func dropCollections(context context.Context, collection *mongo.Collection, db *mongo.Database, stream *stream) error {
	metaCollections := db.Collection("__meta_collection__")

	id, err := primitive.ObjectIDFromHex(collection.Name())
	if err != nil {
		return err
	}
	res := metaCollections.FindOne(context, bson.M{"_id": id})
	meta := make(map[string]interface{})
	err = res.Decode(&meta)
	if err != nil {
		return err
	}

	path := meta["path"].(string)
	filter := bson.M{"path": bson.M{
		"$regex": primitive.Regex{
			Pattern: "^" + path + ".*",
			Options: "i",
		},
	}}

	cursor, err := metaCollections.Find(context, filter)
	if err != nil {
		return err
	}

	defer cursor.Close(context)
	for cursor.Next(context) {
		colData := make(map[string]interface{})
		cursor.Decode(&colData)
		colId := colData["_id"].(primitive.ObjectID)
		colPath := colData["path"].(string)
		if colId.Hex() == collection.Name() {
			continue
		}

		subCol := db.Collection(colId.Hex())
		err := subCol.Drop(context)
		if err != nil {
			return err
		}

		metaCollections.DeleteOne(context, bson.M{"_id": colId})
		stream.delete(colPath, nil)
	}

	err = collection.Drop(context)
	if err == nil {
		metaCollections.DeleteOne(context, bson.M{"_id": meta["_id"]})
		stream.delete(path, nil)
	}

	return err
}

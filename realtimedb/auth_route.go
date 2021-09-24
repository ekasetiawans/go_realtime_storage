package realtimedb

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func initAuthRouter(router *gin.Engine) {
	authGroup := router.Group("/auth", func(c *gin.Context) {
		databaseName := "user_data"
		client := c.MustGet("dbClient").(*mongo.Client)

		db := client.Database(databaseName)
		c.Set("db", db)

		collection := db.Collection("users")
		c.Set("collection", collection)
	})

	authGroup.POST("/refresh-token", func(c *gin.Context) {
		data := struct {
			Token string `json:"token" binding:"required"`
		}{}

		err := c.Bind(&data)
		if err != nil {
			return
		}

	})

	authGroup.POST("/sign-in/email", func(c *gin.Context) {
		data := struct {
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
		}{}

		err := c.Bind(&data)
		if err != nil {
			return
		}

		collection := c.MustGet("collection").(*mongo.Collection)
		res := collection.FindOne(c.Request.Context(), bson.M{
			"email":    data.Email,
			"password": data.Password,
		})

		user := make(map[string]interface{})
		err = res.Decode(&user)
		if err != nil {
			c.Status(403)
			return
		}

		exp := time.Now().Add(8 * time.Hour)
		token, err := makeJwt(user, exp)
		if err != nil {
			c.Status(500)
			return
		}
		c.JSON(200, gin.H{
			"success":    true,
			"token":      token,
			"expired_at": exp,
		})
	})

	authGroup.POST("/register/email", func(c *gin.Context) {
		data := struct {
			Email     string `json:"email" binding:"required"`
			Password  string `json:"password" binding:"required"`
			FirstName string `json:"first_name" binding:"required"`
			LastName  string `json:"last_name" binding:"required"`
		}{}

		err := c.Bind(&data)
		if err != nil {
			return
		}

		collection := c.MustGet("collection").(*mongo.Collection)
		res := collection.FindOne(c.Request.Context(), bson.M{
			"email": data.Email,
		})

		user := make(map[string]interface{})
		err = res.Decode(&user)
		if err == nil {

			c.JSON(402, gin.H{
				"message": "Email already registered",
			})
			return
		}

		insertResult, err := collection.InsertOne(c.Request.Context(), bson.M{
			"email":      data.Email,
			"password":   data.Password,
			"first_name": data.FirstName,
			"last_name":  data.LastName,
		})

		if err != nil {
			c.Status(500)
			return
		}

		res = collection.FindOne(c.Request.Context(), bson.M{
			"_id": insertResult.InsertedID,
		})

		user = make(map[string]interface{})
		err = res.Decode(&user)
		if err != nil {
			c.Status(500)
			return
		}

		exp := time.Now().Add(8 * time.Hour)
		token, err := makeJwt(user, exp)
		if err != nil {
			c.Status(500)
			return
		}
		c.JSON(200, gin.H{
			"success":    true,
			"token":      token,
			"expired_at": exp,
		})
	})
}

func validateJwt(token string) (map[string]interface{}, error) {
	t, e := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte("helloworld"), t.Claims.Valid()
	})

	if e != nil {
		return nil, e
	}

	user := t.Claims.(jwt.MapClaims)["user"].(map[string]interface{})
	return user, nil
}

func makeJwt(user map[string]interface{}, exp time.Time) (string, error) {
	delete(user, "password")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": user,
		"iat":  time.Now().Unix(),
		"exp":  exp.Unix(),
	})

	key := []byte("helloworld")
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

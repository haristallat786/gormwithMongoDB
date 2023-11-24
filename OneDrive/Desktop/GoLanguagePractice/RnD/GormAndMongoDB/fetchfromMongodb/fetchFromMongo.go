package main

import (
	"context"
	"log"
	"net/http"
	"time"
     "fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	mongoDBHost = "mongodb+srv://haris:haris786@cluster0.ep7v32j.mongodb.net/"
	mongoDBName = "todo"
)

// ToDo represents the ToDo model for MongoDB
type ToDoMongo struct {
	ID          uint      `bson:"_id,omitempty"`
	Description string    `bson:"description"`
	IsDone      bool      `bson:"isDone"`
	CreatedAt   time.Time `bson:"createdAt"`
	UpdatedAt   time.Time `bson:"updatedAt"`
}

func connectMongoDB() (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(fmt.Sprintf("%s/%s", mongoDBHost, mongoDBName))
	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getAllToDosFromMongo(collection *mongo.Collection) ([]ToDoMongo, error) {
	var todos []ToDoMongo
	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var todo ToDoMongo
		err := cursor.Decode(&todo)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}

	return todos, nil
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	mongoClient, err := connectMongoDB()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	todoCollection := mongoClient.Database(mongoDBName).Collection("todos")

	r.GET("/todos-mongo", func(c *gin.Context) {
		todos, err := getAllToDosFromMongo(todoCollection)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching ToDo items from MongoDB"})
			return
		}

		c.JSON(http.StatusOK, todos)
	})

	r.Run(":8086")
}

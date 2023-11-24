package main
import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	// "go.mongodb.org/mongo-driver/bson"
	// "go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	postgresHost     = "localhost"
	postgresPort     = 9920
	postgresUser     = "postgres"
	postgresPassword = "12345"
	postgresDBName   = "postgres"

	mongoDBHost = "mongodb+srv://haris:haris786@cluster0.ep7v32j.mongodb.net/"
	mongoDBPort = 27017
	mongoDBName = "todo"
)

// ToDo represents the ToDo model for PostgreSQL and MongoDB
type ToDo struct {
	ID          uint               `gorm:"primaryKey" bson:"_id,omitempty"`
	Description string             `gorm:"not null" bson:"description"`
	IsDone      bool               `gorm:"column:is_done" bson:"isDone"`
	CreatedAt   time.Time          `gorm:"column:created_at" bson:"createdAt"`
	UpdatedAt   time.Time          `gorm:"column:updated_at" bson:"updatedAt"`
}

func connectPostgresDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", postgresHost, postgresPort, postgresUser, postgresPassword, postgresDBName)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
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

func migrateDB(db *gorm.DB) {
	err := db.AutoMigrate(&ToDo{})
	if err != nil {
		log.Fatal(err)
	}
}

func getAllToDosFromPostgres(db *gorm.DB) ([]ToDo, error) {
	var todos []ToDo
	if err := db.Find(&todos).Error; err != nil {
		return nil, err
	}
	return todos, nil
}

func insertToDosIntoMongoDB(collection *mongo.Collection, todos []ToDo) error {
	var mongoTodos []interface{}
	for _, todo := range todos {
		mongoTodos = append(mongoTodos, todo)
	}

	_, err := collection.InsertMany(context.TODO(), mongoTodos)
	return err
}

func createToDoHandler(c *gin.Context, db *gorm.DB) {
	var newTodo ToDo
	if err := c.ShouldBindJSON(&newTodo); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	if err := db.Create(&newTodo).Error; err != nil {
		c.JSON(500, gin.H{"error": "Error creating ToDo item"})
		return
	}

	c.JSON(201, gin.H{"message": "ToDo item created successfully", "todo": newTodo})
}

func updateToDoHandler(c *gin.Context, db *gorm.DB) {
	id := c.Param("id")
	var todo ToDo

	if err := db.First(&todo, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "ToDo item not found"})
		return
	}

	var updateData struct {
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	if err := db.Model(&ToDo{}).Where("id = ?", id).Update("description", updateData.Description).Error; err != nil {
		c.JSON(500, gin.H{"error": "Error updating ToDo item"})
		return
	}

	c.JSON(200, gin.H{"message": "ToDo item updated successfully"})
}

func deleteToDoHandler(c *gin.Context, db *gorm.DB) {
	id := c.Param("id")

	var todo ToDo
	if err := db.First(&todo, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "ToDo item not found"})
		return
	}

	if err := db.Delete(&todo).Error; err != nil {
		c.JSON(500, gin.H{"error": "Error deleting ToDo item"})
		return
	}

	c.JSON(200, gin.H{"message": "ToDo item deleted successfully"})
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	postgresDB, err := connectPostgresDB()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		sqlDB, err := postgresDB.DB()
		if err != nil {
			log.Fatal(err)
		}
		sqlDB.Close()
	}()

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

	migrateDB(postgresDB)

	r.POST("/todos", func(c *gin.Context) {
		createToDoHandler(c, postgresDB)
	})

	r.GET("/todos", func(c *gin.Context) {
		todos, err := getAllToDosFromPostgres(postgresDB)
		if err != nil {
			c.JSON(500, gin.H{"error": "Error fetching ToDo items from PostgreSQL"})
			return
		}

		err = insertToDosIntoMongoDB(todoCollection, todos)
		if err != nil {
			c.JSON(500, gin.H{"error": "Error inserting ToDo items into MongoDB"})
			return
		}

		c.JSON(200, todos)
	})
	todos, err := getAllToDosFromPostgres(postgresDB)
    if err != nil {
        log.Fatal("Error fetching ToDo items from PostgreSQL:", err)
    }

    err = insertToDosIntoMongoDB(todoCollection, todos)
    if err != nil {
        log.Fatal("Error inserting ToDo items into MongoDB:", err)
    }
	r.PUT("/todos/:id", func(c *gin.Context) {
		updateToDoHandler(c, postgresDB)
	})

	r.DELETE("/todos/:id", func(c *gin.Context) {
		deleteToDoHandler(c, postgresDB)
	})

	r.Run(":8085")
}

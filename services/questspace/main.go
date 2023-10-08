package main

import (
	"fmt"
	"net/http"
	"questspace/pkg/application"
	"questspace/services/questspace/internal/handlers/user"
	pgdb "questspace/services/questspace/internal/pgdb/client"

	"github.com/jackc/pgx"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"
)

var config struct {
	Section struct {
		Key string `yaml:"key"`
	} `yaml:"section"`
	DB struct {
		Host     string `yaml:"host"`
		Port     uint16 `yaml:"port"`
		Database string `yaml:"database"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"db"`
}

var reqCount = 0

// HandleHello is test handler to check OK and err responses
func HandleHello(c *gin.Context) error {
	if reqCount == 2 {
		return xerrors.New("Too many requests")
	}
	reqCount++
	c.JSON(http.StatusOK, gin.H{"message": "hello"})
	return nil
}

func Init(app application.App) error {
	fmt.Printf("Got key: %s", config.Section.Key)
	app.Router().GET("/hello", application.AsGinHandler(HandleHello))

	connConfig := pgx.ConnConfig{
		Host:     config.DB.Host,
		Port:     config.DB.Port,
		Database: config.DB.Database,
		User:     config.DB.User,
		Password: config.DB.Password,
	}
	conn, err := pgx.Connect(connConfig)
	if err != nil {
		return xerrors.Errorf("failed to connect to database: %w", err)
	}
	sqlStorage := pgdb.NewClient(conn)
	client := http.Client{}
	createHandler := user.NewCreateHandler(sqlStorage, client)
	app.Router().POST("/user", application.AsGinHandler(createHandler.Handle))

	getHandler := user.NewGetHandler(sqlStorage)
	app.Router().GET("/user", application.AsGinHandler(getHandler.HandleQS))
	app.Router().GET("/user/:id", application.AsGinHandler(getHandler.HandlePath))

	return nil
}

func main() {
	application.Run(Init, &config)
}

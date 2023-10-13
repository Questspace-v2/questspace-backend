package main

import (
	"fmt"
	"net/http"
	user "questspace/internal/handlers/user"
	pgdb "questspace/internal/pgdb/client"
	"questspace/pkg/application"
	"strings"

	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx"
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
	Cors struct {
		AllowOrigin string `yaml:"allow-origin"`
	} `yaml:"cors"`
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

	corsConfig := cors.DefaultConfig()
	if config.Cors.AllowOrigin == "*" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = strings.Split(config.Cors.AllowOrigin, ",")
	}
	app.Router().Use(cors.New(corsConfig))

	app.Router().GET("/hello", application.AsGinHandler(HandleHello))

	// TODO(svayp11): Create type for database config and use env secrets
	connConfig := pgx.ConnConfig{
		Host:     config.DB.Host,
		Port:     config.DB.Port,
		Database: config.DB.Database,
		User:     config.DB.User,
		Password: application.ReadSecret(config.DB.Password),
	}
	conn, err := pgx.Connect(connConfig)
	if err != nil {
		return xerrors.Errorf("failed to connect to database: %w", err)
	}
	sqlStorage := pgdb.NewClient(conn)
	client := http.Client{}

	userGroup := app.Router().Group("/user")

	createHandler := user.NewCreateHandler(sqlStorage, client)
	userGroup.POST("", application.AsGinHandler(createHandler.Handle))

	getHandler := user.NewGetHandler(sqlStorage)
	userGroup.GET("", application.AsGinHandler(getHandler.HandleQS))
	userGroup.GET("/:id", application.AsGinHandler(getHandler.HandlePath))

	updateHandler := user.NewUpdateHandler(sqlStorage, client)
	userGroup.POST("/:id", application.AsGinHandler(updateHandler.Handle))

	return nil
}

func main() {
	application.Run(Init, &config)
}

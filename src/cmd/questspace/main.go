package main

import (
	"crypto/sha256"
	"database/sql"
	"net/http"
	"strings"

	"questspace/docs"
	"questspace/internal/dbconfig"
	"questspace/internal/handlers/user"
	pgdb "questspace/internal/pgdb/client"
	"questspace/pkg/application"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/stdlib"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
	"golang.org/x/xerrors"
)

var config struct {
	DB   dbconfig.Config `yaml:"db"`
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
	corsConfig := cors.DefaultConfig()
	if config.Cors.AllowOrigin == "*" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = strings.Split(config.Cors.AllowOrigin, ",")
	}
	app.Router().Use(cors.New(corsConfig))

	app.Router().GET("/hello", application.AsGinHandler(HandleHello))

	conn, err := sql.Open("pgx", config.DB.GetDSN())
	if err != nil {
		return xerrors.Errorf("failed to connect to database: %w", err)
	}

	sqlStorage := pgdb.NewClient(conn)
	// TODO(svayp11): configure client
	client := http.Client{}
	// TODO(svayp11): Create custom hasher interface
	hasher := sha256.New()

	docs.SwaggerInfo.BasePath = "/"

	userGroup := app.Router().Group("/user")

	createHandler := user.NewCreateHandler(sqlStorage, client, hasher)
	userGroup.POST("", application.AsGinHandler(createHandler.Handle))

	getHandler := user.NewGetHandler(sqlStorage)
	userGroup.GET("/:id", application.AsGinHandler(getHandler.Handle))

	updateHandler := user.NewUpdateHandler(sqlStorage, client, hasher)
	userGroup.POST("/:id", application.AsGinHandler(updateHandler.Handle))

	app.Router().GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))

	return nil
}

func main() {
	application.Run(Init, &config)
}

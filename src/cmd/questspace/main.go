package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/stdlib"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
	"golang.org/x/xerrors"
	"golang.yandex/hasql"
	"golang.yandex/hasql/checkers"

	"questspace/docs"
	"questspace/internal/handlers/quest"
	"questspace/internal/handlers/user"
	"questspace/internal/hasher"
	pgdb "questspace/internal/pgdb/client"
	"questspace/internal/pgdb/pgconfig"
	"questspace/pkg/application"
	"questspace/pkg/auth/jwt"
)

var config struct {
	DB       pgconfig.Config `yaml:"db"`
	HashCost int             `yaml:"hash-cost"`
	Cors     struct {
		AllowOrigin string `yaml:"allow-origin"`
	} `yaml:"cors"`
	JWT jwt.Config `yaml:"jwt"`
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

	nodes, errs := config.DB.GetNodes()
	if len(errs) > 0 {
		return xerrors.Errorf("failed to connect to db nodes: %w", errors.Join(errs...))
	}
	cl, err := hasql.NewCluster(nodes, checkers.PostgreSQL, hasql.WithNodePicker(hasql.PickNodeClosest()))
	if err != nil {
		return xerrors.Errorf("failed to create cluster: %w", err)
	}
	sqlStorage := pgdb.NewClient(cl)

	// TODO(svayp11): configure client
	client := http.Client{}

	pwHasher := hasher.NewBCryptHasher(config.HashCost)

	jwtParser := jwt.NewParser(config.JWT.GetEncryptionKey())

	docs.SwaggerInfo.BasePath = "/"

	userGroup := app.Router().Group("/user")

	createUserHandler := user.NewCreateHandler(sqlStorage, client, pwHasher)
	userGroup.POST("", application.AsGinHandler(createUserHandler.Handle))

	getUserHandler := user.NewGetHandler(sqlStorage)
	userGroup.GET("/:id", application.AsGinHandler(getUserHandler.Handle))

	updateUserHandler := user.NewUpdateHandler(sqlStorage, client, pwHasher)
	userGroup.POST("/:id", application.AsGinHandler(updateUserHandler.Handle))

	questGroup := app.Router().Group("/quest")

	createQuestHandler := quest.NewCreateHandler(sqlStorage)
	questGroup.POST("", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, createQuestHandler.Handle)))

	getQuestHandler := quest.NewGetHandler(sqlStorage)
	questGroup.GET("/:id", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, getQuestHandler.Handle)))

	updateQuestHandler := quest.NewUpdateHandler(sqlStorage)
	questGroup.POST("/:id", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, updateQuestHandler.Handle)))

	app.Router().GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))

	return nil
}

func main() {
	application.Run(Init, &config)
}

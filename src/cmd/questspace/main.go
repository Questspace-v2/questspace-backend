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
	"questspace/internal/handlers/auth"
	"questspace/internal/handlers/quest"
	"questspace/internal/handlers/user"
	"questspace/internal/hasher"
	pgdb "questspace/internal/pgdb/client"
	"questspace/internal/pgdb/pgconfig"
	"questspace/pkg/application"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/dbnode"
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
	nodePicker := dbnode.NewBasicPicker(cl)
	clientFactory := pgdb.NewQuestspaceClientFactory(nodePicker)

	// TODO(svayp11): configure client
	client := http.Client{}

	pwHasher := hasher.NewBCryptHasher(config.HashCost)

	jwtParser := jwt.NewParser(config.JWT.GetEncryptionKey())

	docs.SwaggerInfo.BasePath = "/"

	authGroup := app.Router().Group("/auth")
	authHandler := auth.NewHandler(clientFactory, client, pwHasher, jwtParser)
	authGroup.POST("/register", application.AsGinHandler(authHandler.HandleBasicSignUp))
	authGroup.POST("/sign-in", application.AsGinHandler(authHandler.HandleBasicSignIn))

	userGroup := app.Router().Group("/user")

	getUserHandler := user.NewGetHandler(clientFactory)
	userGroup.GET("/:id", application.AsGinHandler(getUserHandler.Handle))

	updateUserHandler := user.NewUpdateHandler(clientFactory, client, pwHasher, jwtParser)
	userGroup.POST("/:id", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, updateUserHandler.HandleUser)))
	userGroup.POST("/:id/password", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, updateUserHandler.HandlePassword)))

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

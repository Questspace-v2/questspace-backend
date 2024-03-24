package main

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
	"golang.org/x/xerrors"
	"golang.yandex/hasql"
	"golang.yandex/hasql/checkers"

	"questspace/docs"
	"questspace/internal/handlers/auth"
	"questspace/internal/handlers/quest"
	"questspace/internal/handlers/taskgroups"
	"questspace/internal/handlers/teams"
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
	CORS     struct {
		AllowOrigins []string `yaml:"allow-origins"`
		AllowHeaders []string `yaml:"allow-headers"`
		AllowMethods []string `yaml:"allow-methods"`
	} `yaml:"cors"`
	JWT jwt.Config `yaml:"jwt"`
}

func Init(app application.App) error {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = slices.Clone(config.CORS.AllowOrigins)
	corsConfig.AllowHeaders = slices.Clone(config.CORS.AllowHeaders)
	corsConfig.AllowMethods = slices.Clone(config.CORS.AllowMethods)
	app.Router().Use(cors.New(corsConfig))
	app.Router().OPTIONS("/*any", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	nodes, errs := config.DB.GetNodes()
	if len(errs) > 0 {
		return xerrors.Errorf("failed to connect to db nodes: %w", errors.Join(errs...))
	}
	cl, err := hasql.NewCluster(nodes, checkers.PostgreSQL, hasql.WithNodePicker(hasql.PickNodeClosest()))
	if err != nil {
		return xerrors.Errorf("failed to create cluster: %w", err)
	}
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err := cl.WaitForAlive(timeoutCtx); err != nil {
		return xerrors.Errorf("cannot connect to database cluster: %w", err)
	}
	nodePicker := dbnode.NewBasicPicker(cl)
	clientFactory := pgdb.NewQuestspaceClientFactory(nodePicker)
	client := http.Client{
		Timeout: time.Minute,
	}
	pwHasher := hasher.NewBCryptHasher(config.HashCost)
	jwtParser := jwt.NewTokenParser(config.JWT.GetEncryptionKey())

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
	userGroup.DELETE("/:id", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, updateUserHandler.HandleDelete)))

	teamsHandler := teams.NewHandler(clientFactory, "https://new.questspace.app:3000/join")

	questGroup := app.Router().Group("/quest")
	questHandler := quest.NewHandler(clientFactory, client)
	questGroup.POST("", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, questHandler.HandleCreate)))
	questGroup.GET("/:id", application.AsGinHandler(questHandler.HandleGet))
	questGroup.POST("/:id", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, questHandler.HandleUpdate)))
	questGroup.DELETE("/:id", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, questHandler.HandleDelete)))
	questGroup.POST("/:id/teams", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, teamsHandler.HandleCreate)))

	teamsGroup := app.Router().Group("/teams")
	teamsGroup.GET("/:path", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, teamsHandler.HandleJoin)))

	taskGroupHandler := taskgroups.NewHandler(clientFactory)
	questGroup.PATCH("/:id/task-groups/bulk", application.AsGinHandler(taskGroupHandler.HandleBulkUpdate))

	app.Router().GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))
	return nil
}

func main() {
	application.Run(Init, &config)
}

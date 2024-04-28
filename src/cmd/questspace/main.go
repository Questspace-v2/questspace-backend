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
	"google.golang.org/api/idtoken"

	"questspace/docs"
	"questspace/internal/handlers/auth"
	"questspace/internal/handlers/auth/google"
	"questspace/internal/handlers/play"
	"questspace/internal/handlers/quest"
	"questspace/internal/handlers/taskgroups"
	"questspace/internal/handlers/teams"
	"questspace/internal/handlers/user"
	"questspace/internal/hasher"
	"questspace/internal/pgdb"
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
	JWT    jwt.Config    `yaml:"jwt"`
	Teams  teams.Config  `yaml:"teams"`
	Google google.Config `yaml:"google-oauth"`
}

// Init godoc
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
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

	jwtSecret, err := config.JWT.Secret.Read()
	if err != nil {
		return xerrors.Errorf("load jwt secret: %w", err)
	}

	jwtParser := jwt.NewTokenParser([]byte(jwtSecret))

	docs.SwaggerInfo.BasePath = "/"

	tokenValidator, err := idtoken.NewValidator(context.Background())
	if err != nil {
		return xerrors.Errorf("create token validator: %w", err)
	}
	googleOAuthHandler := google.NewOAuthHandler(clientFactory, tokenValidator, jwtParser, config.Google)

	authGroup := app.Router().Group("/auth")
	authHandler := auth.NewHandler(clientFactory, client, pwHasher, jwtParser)
	authGroup.POST("/register", application.AsGinHandler(authHandler.HandleBasicSignUp))
	authGroup.POST("/sign-in", application.AsGinHandler(authHandler.HandleBasicSignIn))
	authGroup.POST("/google", application.AsGinHandler(googleOAuthHandler.Handle))

	userGroup := app.Router().Group("/user")

	getUserHandler := user.NewGetHandler(clientFactory)
	userGroup.GET("/:id", application.AsGinHandler(getUserHandler.Handle))

	updateUserHandler := user.NewUpdateHandler(clientFactory, client, pwHasher, jwtParser)
	userGroup.POST("/:id", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(updateUserHandler.HandleUser))
	userGroup.POST("/:id/password", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(updateUserHandler.HandlePassword))
	userGroup.DELETE("/:id", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(updateUserHandler.HandleDelete))

	teamsHandler := teams.NewHandler(clientFactory, config.Teams.InviteLinkPrefix)

	questGroup := app.Router().Group("/quest")
	questHandler := quest.NewHandler(clientFactory, client, config.Teams.InviteLinkPrefix)
	questGroup.POST("", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(questHandler.HandleCreate))
	questGroup.GET("", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(questHandler.HandleGetMany))
	questGroup.GET("/:id", jwt.AuthMiddleware(jwtParser), application.AsGinHandler(questHandler.HandleGet))
	questGroup.POST("/:id", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(questHandler.HandleUpdate))
	questGroup.DELETE("/:id", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(questHandler.HandleDelete))
	questGroup.POST("/:id/finish", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(questHandler.HandleFinish))

	teamsGroup := app.Router().Group("/teams")
	questGroup.POST("/:id/teams", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(teamsHandler.HandleCreate))
	questGroup.GET("/:id/teams", application.AsGinHandler(teamsHandler.HandleGetMany))
	teamsGroup.GET("/join/:path", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(teamsHandler.HandleJoin))
	teamsGroup.GET("/join/:path/quest", jwt.AuthMiddleware(jwtParser), application.AsGinHandler(teamsHandler.HandleGetQuestByTeamInvite))
	teamsGroup.GET("/:id", application.AsGinHandler(teamsHandler.HandleGet))
	teamsGroup.POST("/:id", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(teamsHandler.HandleUpdate))
	teamsGroup.DELETE("/:id", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(teamsHandler.HandleDelete))
	teamsGroup.POST("/:id/captain", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(teamsHandler.HandleChangeLeader))
	teamsGroup.POST("/:id/leave", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(teamsHandler.HandleLeave))
	teamsGroup.DELETE("/:id/:user_id", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(teamsHandler.HandleRemoveUser))

	taskGroupHandler := taskgroups.NewHandler(clientFactory)
	questGroup.PATCH("/:id/task-groups/bulk", application.AsGinHandler(taskGroupHandler.HandleBulkUpdate))
	questGroup.POST("/:id/task-groups", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(taskGroupHandler.HandleCreate))
	questGroup.GET("/:id/task-groups", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(taskGroupHandler.HandleGet))

	playHandler := play.NewHandler(clientFactory)
	questGroup.GET("/:id/play", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(playHandler.HandleGet))
	questGroup.POST("/:id/hint", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(playHandler.HandleTakeHint))
	questGroup.POST("/:id/answer", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(playHandler.HandleTryAnswer))
	questGroup.GET("/:id/table", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(playHandler.HandleGetTableResults))
	questGroup.GET("/:id/leaderboard", application.AsGinHandler(playHandler.HandleLeaderboard))
	questGroup.POST("/:id/penalty", jwt.AuthMiddlewareStrict(jwtParser), application.AsGinHandler(playHandler.HandleAddPenalty))

	app.Router().GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))
	return nil
}

func main() {
	application.Run(Init, &config)
}

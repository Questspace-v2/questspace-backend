package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"time"

	httpswagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"golang.org/x/xerrors"
	"google.golang.org/api/idtoken"

	"questspace/docs"
	"questspace/internal/app"
	"questspace/internal/handlers/auth"
	"questspace/internal/handlers/auth/google"
	"questspace/internal/handlers/play"
	"questspace/internal/handlers/quest"
	"questspace/internal/handlers/taskgroups"
	"questspace/internal/handlers/teams"
	"questspace/internal/handlers/user"
	"questspace/internal/hasher"
	"questspace/internal/pgdb"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/cors"
	"questspace/pkg/dbnode"
	"questspace/pkg/transport"
)

// InitApp godoc
// @securityDefinitions.apikey 	ApiKeyAuth
// @in 							header
// @name 						Authorization
func InitApp(ctx context.Context, application *app.App) error {
	cfg, err := application.GetConfig()
	if err != nil {
		return err
	}

	application.Router().Use(cors.Middleware(&cfg.CORS))
	application.Router().H().OPTIONS("/*any", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	nodes, errs := pgdb.GetNodes(&cfg.DB)
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	cl, err := pgdb.CreateCluster(ctx, nodes)
	if err != nil {
		return err
	}
	application.Cleanup(cl.Close)
	nodePicker := dbnode.NewBasicPicker(cl)
	clientFactory := pgdb.NewQuestspaceClientFactory(nodePicker)
	httpClient := http.Client{
		Timeout: time.Minute,
	}

	pwHasher := hasher.NewBCryptHasher(cfg.HashCost)
	jwtSecret, err := cfg.JWT.Secret.Read()
	if err != nil {
		return err
	}
	jwtParser := jwt.NewTokenParser([]byte(jwtSecret))

	docs.SwaggerInfo.BasePath = "/"

	tokenValidator, err := idtoken.NewValidator(ctx)
	if err != nil {
		return xerrors.Errorf("create token validator: %w", err)
	}

	r := application.Router()
	r.H().GET("/debug/pprof/", http.HandlerFunc(pprof.Index))
	r.H().GET("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	r.H().GET("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	r.H().GET("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	r.H().GET("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	r.H().GET("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	r.H().GET("/debug/pprof/heap", pprof.Handler("heap"))
	r.H().GET("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	r.H().GET("/debug/pprof/block", pprof.Handler("block"))

	r.H().GET("/swagger/*path", httpswagger.Handler())

	authHandler := auth.NewHandler(clientFactory, httpClient, pwHasher, jwtParser)
	googleOAuthHandler := google.NewOAuthHandler(clientFactory, tokenValidator, jwtParser, cfg.Google)
	r.H().POST("/auth/register", transport.WrapCtxErr(authHandler.HandleBasicSignUp))
	r.H().POST("/auth/sign-in", transport.WrapCtxErr(authHandler.HandleBasicSignIn))
	r.H().POST("/auth/google", transport.WrapCtxErr(googleOAuthHandler.Handle))

	getUserHandler := user.NewGetHandler(clientFactory)
	r.H().GET("/user/:id", transport.WrapCtxErr(getUserHandler.Handle))
	updateUserHandler := user.NewUpdateHandler(clientFactory, httpClient, pwHasher, jwtParser)
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/user/:id", transport.WrapCtxErr(updateUserHandler.HandleUser))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/user/:id/password", transport.WrapCtxErr(updateUserHandler.HandlePassword))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).DELETE("/user/:id", transport.WrapCtxErr(updateUserHandler.HandleDelete))

	questHandler := quest.NewHandler(clientFactory, httpClient, cfg.Teams.InviteLinkPrefix)
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/quest", transport.WrapCtxErr(questHandler.HandleCreate))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).GET("/quest", transport.WrapCtxErr(questHandler.HandleGetMany))
	r.H().Use(jwt.AuthMiddleware(jwtParser)).GET("/quest/:id", transport.WrapCtxErr(questHandler.HandleGet))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/quest/:id", transport.WrapCtxErr(questHandler.HandleUpdate))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).DELETE("/quest/:id", transport.WrapCtxErr(questHandler.HandleDelete))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/quest/:id/finish", transport.WrapCtxErr(questHandler.HandleFinish))

	teamsHandler := teams.NewHandler(clientFactory, cfg.Teams.InviteLinkPrefix)
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/quest/:id/teams", transport.WrapCtxErr(teamsHandler.HandleCreate))
	r.H().GET("/quest/:id/teams", transport.WrapCtxErr(teamsHandler.HandleGetMany))
	r.H().GET("/teams/all/:id", transport.WrapCtxErr(teamsHandler.HandleGet))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).DELETE("/teams/all/:id", transport.WrapCtxErr(teamsHandler.HandleDelete))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).GET("/teams/join/:path", transport.WrapCtxErr(teamsHandler.HandleJoin))
	r.H().Use(jwt.AuthMiddleware(jwtParser)).GET("/teams/join/:path/quest", transport.WrapCtxErr(teamsHandler.HandleGetQuestByTeamInvite))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/teams/all/:id", transport.WrapCtxErr(teamsHandler.HandleUpdate))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/teams/all/:id/captain", transport.WrapCtxErr(teamsHandler.HandleChangeLeader))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/teams/all/:id/leave", transport.WrapCtxErr(teamsHandler.HandleLeave))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).DELETE("/teams/all/:id/:user_id", transport.WrapCtxErr(teamsHandler.HandleRemoveUser))

	taskGroupHandler := taskgroups.NewHandler(clientFactory)
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).PATCH("/quest/:id/task-groups/bulk", transport.WrapCtxErr(taskGroupHandler.HandleBulkUpdate))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/quest/:id/task-groups", transport.WrapCtxErr(taskGroupHandler.HandleCreate))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).GET("/quest/:id/task-groups", transport.WrapCtxErr(taskGroupHandler.HandleGet))

	playHandler := play.NewHandler(clientFactory)
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).GET("/quest/:id/play", transport.WrapCtxErr(playHandler.HandleGet))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/quest/:id/hint", transport.WrapCtxErr(playHandler.HandleTakeHint))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/quest/:id/answer", transport.WrapCtxErr(playHandler.HandleTryAnswer))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).GET("/quest/:id/table", transport.WrapCtxErr(playHandler.HandleGetTableResults))
	r.H().GET("/quest/:id/leaderboard", transport.WrapCtxErr(playHandler.HandleLeaderboard))
	r.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/quest/:id/penalty", transport.WrapCtxErr(playHandler.HandleAddPenalty))
	return nil
}

func RunApp() (code int) {
	ctx, cancel := signal.NotifyContext(context.Background(), unix.SIGINT, unix.SIGTERM)
	defer cancel()

	application := app.NewApp()
	defer func() { _ = application.Close() }()

	if err := InitApp(ctx, application); err != nil {
		application.Logger().Error("application init error", zap.Error(err))
		return 1
	}

	if err := application.Run(ctx); err != nil {
		application.Logger().Error("server down", zap.Error(err))
		return 1
	}
	return 0
}

func main() {
	os.Exit(RunApp())
}

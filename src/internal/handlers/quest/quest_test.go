package quest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/spkg/ptr"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	pgdb "questspace/internal/pgdb/client"
	"questspace/pkg/application"
	"questspace/pkg/auth/jwt"
	jwtmock "questspace/pkg/auth/jwt/mocks"
	"questspace/pkg/storage"
	storagemock "questspace/pkg/storage/mocks"
)

func TestHandleCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	userStorage := storagemock.NewMockQuestSpaceStorage(ctrl)
	jwtParser := jwtmock.NewMockParser(ctrl)
	factory := pgdb.NewFakeClientFactory(userStorage)

	router := gin.Default()
	handler := NewHandler(factory, http.Client{})
	router.Use(func(c *gin.Context) {
		c.Set("app-logger", zap.NewNop())
		c.Next()
	})
	router.POST("/quest", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, handler.HandleCreate)))

	now := ptr.Time(time.Unix(time.Now().Unix(), 0))
	userCreator := storage.User{
		Username: "username",
		Password: "password",
	}
	req := storage.CreateQuestRequest{
		Name:        "questname",
		Description: "test description yea",
		Access:      storage.Public,
		StartTime:   now,
		MaxTeamCap:  ptr.Int(228),
	}
	newQuest := storage.Quest{
		ID:          "1",
		Name:        req.Name,
		Description: req.Description,
		Access:      req.Access,
		Creator:     &userCreator,
		StartTime:   req.StartTime,
		MaxTeamCap:  req.MaxTeamCap,
	}

	raw, err := json.Marshal(req)
	require.NoError(t, err)
	httpReq, err := http.NewRequest(http.MethodPost, "/quest", bytes.NewReader(raw))
	httpReq.Header.Add("Authorization", "Bearer token")
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	req.Creator = &userCreator
	jwtParser.EXPECT().ParseToken("token").Return(&userCreator, nil)
	userStorage.EXPECT().CreateQuest(gomock.Any(), &req).Return(&newQuest, nil)

	router.ServeHTTP(rr, httpReq)
	require.Equal(t, http.StatusOK, rr.Code)
}

package user

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"

	"questspace/internal/hasher"
	pgdb "questspace/internal/pgdb/client"
	"questspace/pkg/application"
	"questspace/pkg/auth/jwt"
	jwtmock "questspace/pkg/auth/jwt/mocks"
	"questspace/pkg/storage"
	storagemock "questspace/pkg/storage/mocks"
)

func TestGetHandler_CommonCases(t *testing.T) {
	testCases := []struct {
		name       string
		id         string
		getReq     *storage.GetUserRequest
		getErr     error
		statusCode int
	}{
		{
			name:       "ok",
			id:         "id",
			getReq:     &storage.GetUserRequest{ID: "id"},
			statusCode: http.StatusOK,
		},
		{
			name:       "not found",
			id:         "non_existent_id",
			getReq:     &storage.GetUserRequest{ID: "non_existent_id"},
			getErr:     storage.ErrNotFound,
			statusCode: http.StatusNotFound,
		},
		{
			name:       "internal error",
			id:         "id",
			getReq:     &storage.GetUserRequest{ID: "id"},
			getErr:     xerrors.New("oops"),
			statusCode: http.StatusInternalServerError,
		},
	}

	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	userStorage := storagemock.NewMockQuestSpaceStorage(ctrl)
	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Set("app-logger", zap.NewNop())
		c.Next()
	})
	factory := pgdb.NewFakeClientFactory(userStorage)
	handler := NewGetHandler(factory)
	router.GET("/user/:id", application.AsGinHandler(handler.Handle))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, "/user/"+tc.id, nil)
			require.NoError(t, err)

			userStorage.EXPECT().GetUser(gomock.Any(), tc.getReq).Return(nil, tc.getErr)

			router.ServeHTTP(rr, request)
			require.Equal(t, tc.statusCode, rr.Code)
		})
	}
}

func TestUpdateHandler_HandleUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	userStorage := storagemock.NewMockQuestSpaceStorage(ctrl)
	jwtParser := jwtmock.NewMockParser(ctrl)
	factory := pgdb.NewFakeClientFactory(userStorage)
	pwHasher := hasher.NewNopHasher()

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Set("app-logger", zap.NewNop())
		c.Next()
	})
	handler := NewUpdateHandler(factory, http.Client{}, pwHasher, jwtParser)
	router.POST("/user/:id", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, handler.HandleUser)))

	oldUser := storage.User{
		ID:        "1",
		Username:  "old_username",
		AvatarURL: "https://api.dicebear.com/7.x/thumbs/svg?seed=123132",
	}
	expectedUser := storage.User{
		ID:        "1",
		Username:  "another_username",
		AvatarURL: "https://api.dicebear.com/7.x/thumbs/svg?seed=123132",
	}

	req := UpdatePublicDataRequest{
		Username: expectedUser.Username,
	}
	raw, err := json.Marshal(req)
	require.NoError(t, err)
	httpReq, err := http.NewRequest(http.MethodPost, "/user/1", bytes.NewReader(raw))
	httpReq.Header.Add("Authorization", "Bearer alg.pld.key")
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	jwtParser.EXPECT().ParseToken("alg.pld.key").Return(&oldUser, nil)
	userStorage.EXPECT().UpdateUser(gomock.Any(), &storage.UpdateUserRequest{ID: oldUser.ID, Username: expectedUser.Username}).Return(&expectedUser, nil)
	jwtParser.EXPECT().CreateToken(&expectedUser).Return("alg.another_pld.new_key", nil)

	router.ServeHTTP(rr, httpReq)
	require.Equal(t, http.StatusOK, rr.Code)
	factory.ExpectCommit(t)
	require.Contains(t, rr.Header().Get("Set-Cookie"), "alg.another_pld.new_key")
}

func TestUpdateHandler_HandlePassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	userStorage := storagemock.NewMockQuestSpaceStorage(ctrl)
	jwtParser := jwtmock.NewMockParser(ctrl)
	factory := pgdb.NewFakeClientFactory(userStorage)
	pwHasher := hasher.NewNopHasher()

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Set("app-logger", zap.NewNop())
		c.Next()
	})
	handler := NewUpdateHandler(factory, http.Client{}, pwHasher, jwtParser)
	router.POST("/user/:id/password", application.AsGinHandler(jwt.WithJWTMiddleware(jwtParser, handler.HandlePassword)))

	oldUser := storage.User{
		ID:        "1",
		Username:  "username",
		AvatarURL: "https://api.dicebear.com/7.x/thumbs/svg?seed=123132",
	}
	oldPw := "1234"
	newPw := "93d14e9df9c57a379b2185ad742a4c25ff8de353"

	req := UpdatePasswordRequest{
		OldPassword: oldPw,
		NewPassword: newPw,
	}
	raw, err := json.Marshal(req)
	require.NoError(t, err)
	httpReq, err := http.NewRequest(http.MethodPost, "/user/1/password", bytes.NewReader(raw))
	httpReq.Header.Add("Authorization", "Bearer alg.pld.key")
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	jwtParser.EXPECT().ParseToken("alg.pld.key").Return(&oldUser, nil)
	userStorage.EXPECT().GetUserPasswordHash(gomock.Any(), &storage.GetUserRequest{ID: oldUser.ID}).Return(oldPw, nil)
	userStorage.EXPECT().UpdateUser(gomock.Any(), &storage.UpdateUserRequest{ID: oldUser.ID, Password: newPw}).Return(&oldUser, nil)

	router.ServeHTTP(rr, httpReq)
	require.Equal(t, http.StatusOK, rr.Code)
}

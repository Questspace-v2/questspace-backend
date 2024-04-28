package user

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/xerrors"

	"questspace/internal/hasher"
	"questspace/internal/pgdb"
	"questspace/pkg/auth/jwt"
	jwtmock "questspace/pkg/auth/jwt/mocks"
	"questspace/pkg/middleware"
	"questspace/pkg/storage"
	storagemock "questspace/pkg/storage/mocks"
	"questspace/pkg/transport"
)

var (
	existentID    = uuid.Must(uuid.NewV4())
	nonExistentID = uuid.Must(uuid.NewV4())
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
			id:         existentID.String(),
			getReq:     &storage.GetUserRequest{ID: existentID.String()},
			statusCode: http.StatusOK,
		},
		{
			name:       "not found",
			id:         nonExistentID.String(),
			getReq:     &storage.GetUserRequest{ID: nonExistentID.String()},
			getErr:     storage.ErrNotFound,
			statusCode: http.StatusNotFound,
		},
		{
			name:       "internal error",
			id:         existentID.String(),
			getReq:     &storage.GetUserRequest{ID: existentID.String()},
			getErr:     xerrors.New("oops"),
			statusCode: http.StatusInternalServerError,
		},
	}

	ctrl := gomock.NewController(t)
	userStorage := storagemock.NewMockQuestSpaceStorage(ctrl)
	router := transport.NewRouter()
	router.Use(middleware.CtxLog(zaptest.NewLogger(t)))
	factory := pgdb.NewFakeClientFactory(userStorage)
	handler := NewGetHandler(factory)
	router.H().GET("/user/{id}", transport.WrapCtxErr(handler.Handle))

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
	ctrl := gomock.NewController(t)
	userStorage := storagemock.NewMockQuestSpaceStorage(ctrl)
	jwtParser := jwtmock.NewMockParser(ctrl)
	factory := pgdb.NewFakeClientFactory(userStorage)
	pwHasher := hasher.NewNopHasher()

	router := transport.NewRouter()
	router.Use(middleware.CtxLog(zaptest.NewLogger(t)))
	handler := NewUpdateHandler(factory, http.Client{}, pwHasher, jwtParser)
	router.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/user/{id}", transport.WrapCtxErr(handler.HandleUser))

	oldUser := storage.User{
		ID:        existentID.String(),
		Username:  "old_username",
		AvatarURL: "https://api.dicebear.com/7.x/thumbs/svg?seed=123132",
	}
	expectedUser := storage.User{
		ID:        existentID.String(),
		Username:  "another_username",
		AvatarURL: "https://api.dicebear.com/7.x/thumbs/svg?seed=123132",
	}

	req := UpdatePublicDataRequest{
		Username: expectedUser.Username,
	}
	raw, err := json.Marshal(req)
	require.NoError(t, err)
	httpReq, err := http.NewRequest(http.MethodPost, "/user/"+existentID.String(), bytes.NewReader(raw))
	require.NoError(t, err)
	httpReq.Header.Add("Authorization", "Bearer alg.pld.key")
	rr := httptest.NewRecorder()

	jwtParser.EXPECT().ParseToken("alg.pld.key").Return(&oldUser, nil)
	userStorage.EXPECT().UpdateUser(gomock.Any(), &storage.UpdateUserRequest{ID: oldUser.ID, Username: expectedUser.Username}).Return(&expectedUser, nil)
	jwtParser.EXPECT().CreateToken(&expectedUser).Return("alg.another_pld.new_key", nil)

	router.ServeHTTP(rr, httpReq)
	require.Equal(t, http.StatusOK, rr.Code)
	factory.ExpectCommit(t)
}

func TestUpdateHandler_HandlePassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	userStorage := storagemock.NewMockQuestSpaceStorage(ctrl)
	jwtParser := jwtmock.NewMockParser(ctrl)
	factory := pgdb.NewFakeClientFactory(userStorage)
	pwHasher := hasher.NewNopHasher()

	router := transport.NewRouter()
	router.Use(middleware.CtxLog(zaptest.NewLogger(t)))
	handler := NewUpdateHandler(factory, http.Client{}, pwHasher, jwtParser)
	router.H().Use(jwt.AuthMiddlewareStrict(jwtParser)).POST("/user/{id}/password", transport.WrapCtxErr(handler.HandlePassword))

	oldUser := storage.User{
		ID:        existentID.String(),
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
	httpReq, err := http.NewRequest(http.MethodPost, "/user/"+existentID.String()+"/password", bytes.NewReader(raw))
	require.NoError(t, err)
	httpReq.Header.Add("Authorization", "Bearer alg.pld.key")
	rr := httptest.NewRecorder()

	jwtParser.EXPECT().ParseToken("alg.pld.key").Return(&oldUser, nil)
	userStorage.EXPECT().GetUserPasswordHash(gomock.Any(), &storage.GetUserRequest{ID: oldUser.ID}).Return(oldPw, nil)
	userStorage.EXPECT().UpdateUser(gomock.Any(), &storage.UpdateUserRequest{ID: oldUser.ID, Password: newPw}).Return(&oldUser, nil)

	router.ServeHTTP(rr, httpReq)
	require.Equal(t, http.StatusOK, rr.Code)
}

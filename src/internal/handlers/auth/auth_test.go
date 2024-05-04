package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/xerrors"

	"questspace/internal/hasher"
	"questspace/internal/pgdb"
	jwtmock "questspace/pkg/auth/jwt/mocks"
	"questspace/pkg/middleware"
	"questspace/pkg/storage"
	storagemock "questspace/pkg/storage/mocks"
	"questspace/pkg/transport"
)

type storageRegData struct {
	willStore  bool
	willCommit bool
	usr        storage.User
	err        error
}

type jwtData struct {
	willCreate bool
	token      string
	err        error
}

func TestAuth_HandleBasicSignUp(t *testing.T) {
	testCases := []struct {
		name        string
		imgType     string
		req         storage.CreateUserRequest
		storageData storageRegData
		jwtData     jwtData
		wantCommit  bool
		statusCode  int
	}{
		{
			name:    "ok",
			imgType: "image/svg",
			req: storage.CreateUserRequest{
				Username: "user",
				Password: "password",
			},
			storageData: storageRegData{
				willStore:  true,
				willCommit: true,
				usr: storage.User{
					Username: "user",
					Password: "password",
				},
			},
			jwtData: jwtData{
				willCreate: true,
				token:      "new_token",
			},
			wantCommit: true,
			statusCode: http.StatusOK,
		},
		{
			name:    "invalid image",
			imgType: "application/xml",
			req: storage.CreateUserRequest{
				Username: "user",
				Password: "password",
			},
			statusCode: http.StatusUnsupportedMediaType,
		},
		{
			name:    "already exists",
			imgType: "image/svg",
			req: storage.CreateUserRequest{
				Username: "exists",
				Password: "whatever",
			},
			storageData: storageRegData{
				willStore: true,
				err:       storage.ErrExists,
			},
			statusCode: http.StatusBadRequest,
		},
		{
			name:    "jwt token error",
			imgType: "image/svg",
			req: storage.CreateUserRequest{
				Username: "username",
				Password: "password",
			},
			storageData: storageRegData{
				willStore: true,
				usr: storage.User{
					Username: "username",
					Password: "password",
				},
			},
			jwtData: jwtData{
				willCreate: true,
				err:        xerrors.New("oops"),
			},
			statusCode: http.StatusInternalServerError,
		},
	}
	ctrl := gomock.NewController(t)
	userStorage := storagemock.NewMockQuestSpaceStorage(ctrl)
	pwHasher := hasher.NewNopHasher()
	factory := pgdb.NewFakeClientFactory(userStorage)
	jwtParser := jwtmock.NewMockParser(ctrl)

	router := transport.NewRouter()

	handler := NewHandler(factory, http.Client{}, pwHasher, jwtParser)
	router.Use(middleware.CtxLog(zaptest.NewLogger(t)))
	router.H().POST("/auth/register", transport.WrapCtxErr(handler.HandleBasicSignUp))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			imgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Content-Type", tc.imgType)
				w.WriteHeader(http.StatusOK)
			}))
			defer imgSrv.Close()
			tc.req.AvatarURL = imgSrv.URL

			raw, err := json.Marshal(tc.req)
			require.NoError(t, err)
			httpReq, err := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(raw))
			require.NoError(t, err)

			if tc.storageData.willStore {
				userStorage.EXPECT().CreateUser(gomock.Any(), &tc.req).Return(&tc.storageData.usr, tc.storageData.err)
			}
			if tc.jwtData.willCreate {
				jwtParser.EXPECT().CreateToken(&tc.storageData.usr).Return(tc.jwtData.token, tc.jwtData.err)
			}

			router.ServeHTTP(rr, httpReq)
			require.Equal(t, tc.statusCode, rr.Code)
			if tc.storageData.willCommit {
				factory.ExpectCommit(t)
			}
		})
	}
}

type storageSignInData struct {
	oldPw string
	pwErr error
	usr   storage.User
	err   error
}

func TestAuth_HandleBasicSignIn(t *testing.T) {
	testCases := []struct {
		name        string
		req         SignInRequest
		storageData storageSignInData
		jwtData     jwtData
		statusCode  int
	}{
		{
			name: "ok",
			req: SignInRequest{
				Username: "user",
				Password: "password",
			},
			storageData: storageSignInData{
				oldPw: "password",
				usr: storage.User{
					ID:       "1",
					Username: "user",
					Password: "password",
				},
			},
			jwtData: jwtData{
				willCreate: true,
				token:      "new_token",
			},
			statusCode: http.StatusOK,
		},
		{
			name: "no such user",
			req: SignInRequest{
				Username: "not exists",
				Password: "whatever",
			},
			storageData: storageSignInData{
				pwErr: storage.ErrNotFound,
			},
			statusCode: http.StatusNotFound,
		},
		{
			name: "invalid password",
			req: SignInRequest{
				Username: "username",
				Password: "ew not that",
			},
			storageData: storageSignInData{
				oldPw: "password",
			},
			statusCode: http.StatusForbidden,
		},
	}
	ctrl := gomock.NewController(t)
	userStorage := storagemock.NewMockQuestSpaceStorage(ctrl)
	pwHasher := hasher.NewNopHasher()
	factory := pgdb.NewFakeClientFactory(userStorage)
	jwtParser := jwtmock.NewMockParser(ctrl)

	router := transport.NewRouter()
	router.Use(middleware.CtxLog(zaptest.NewLogger(t)))
	handler := NewHandler(factory, http.Client{}, pwHasher, jwtParser)
	router.H().POST("/auth/sign-in", transport.WrapCtxErr(handler.HandleBasicSignIn))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			raw, err := json.Marshal(tc.req)
			require.NoError(t, err)
			httpReq, err := http.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewReader(raw))
			require.NoError(t, err)

			userStorage.EXPECT().GetUserPasswordHash(gomock.Any(), &storage.GetUserRequest{Username: tc.req.Username}).Return(tc.storageData.oldPw, tc.storageData.pwErr)
			if tc.storageData.pwErr == nil && tc.storageData.oldPw == tc.req.Password {
				userStorage.EXPECT().GetUser(gomock.Any(), &storage.GetUserRequest{Username: tc.req.Username}).Return(&tc.storageData.usr, tc.storageData.err)
			}
			if tc.jwtData.willCreate {
				jwtParser.EXPECT().CreateToken(&tc.storageData.usr).Return(tc.jwtData.token, tc.jwtData.err)
			}

			router.ServeHTTP(rr, httpReq)
			require.Equal(t, tc.statusCode, rr.Code)
		})
	}
}

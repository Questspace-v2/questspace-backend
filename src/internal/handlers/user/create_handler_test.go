package user

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"questspace/internal/hasher"
	"questspace/pkg/application"
	"questspace/pkg/storage"
	"questspace/pkg/storage/mocks"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
)

func TestCreateHandler_CommonCases(t *testing.T) {
	testCases := []struct {
		name       string
		imgType    string
		req        *storage.CreateUserRequest
		wantStore  bool
		storeErr   error
		statusCode int
	}{
		{
			name:    "ok",
			imgType: "image/svg",
			req: &storage.CreateUserRequest{
				Username: "user",
				Password: "password",
			},
			wantStore:  true,
			statusCode: http.StatusOK,
		},
		{
			name: "ok with custom avatar link",
			req: &storage.CreateUserRequest{
				Username:  "user",
				Password:  "password",
				AvatarURL: "https://some.domain.com/avatar.png",
			},
			wantStore:  true,
			statusCode: http.StatusOK,
		},
		{
			name:    "not an image",
			imgType: "application/json",
			req: &storage.CreateUserRequest{
				Username: "user",
				Password: "password",
			},
			wantStore:  false,
			statusCode: http.StatusUnprocessableEntity,
		},
		{
			name:    "already exists",
			imgType: "image/png",
			req: &storage.CreateUserRequest{
				Username: "user",
				Password: "password",
			},
			wantStore:  true,
			storeErr:   storage.ErrExists,
			statusCode: http.StatusBadRequest,
		},
		{
			name:    "internal error",
			imgType: "image/jpg",
			req: &storage.CreateUserRequest{
				Username: "user",
				Password: "password",
			},
			wantStore:  true,
			storeErr:   xerrors.New("oops"),
			statusCode: http.StatusInternalServerError,
		},
	}

	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	userStorage := mocks.NewMockUserStorage(ctrl)
	router := gin.Default()
	pwHasher := sha256.New()
	handler := NewCreateHandler(userStorage, http.Client{}, pwHasher)
	router.POST("/test", application.AsGinHandler(handler.Handle))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			if tc.req.AvatarURL == "" {
				img := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Add("Content-Type", tc.imgType)
					w.WriteHeader(http.StatusOK)
				}))
				defer img.Close()
				tc.req.AvatarURL = img.URL
			}
			raw, err := json.Marshal(tc.req)
			require.NoError(t, err)
			request, err := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(raw))
			require.NoError(t, err)
			actualReq := &storage.CreateUserRequest{
				Username:  tc.req.Username,
				Password:  hasher.HashString(pwHasher, tc.req.Password),
				AvatarURL: tc.req.AvatarURL,
			}
			pwHasher.Reset()
			if tc.wantStore {
				userStorage.EXPECT().CreateUser(gomock.Any(), actualReq).Return(nil, tc.storeErr)
			}

			router.ServeHTTP(rr, request)
			require.Equal(t, tc.statusCode, rr.Code)
		})
	}
}

func TestCreateHandler_SetsDefaultURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	userStorage := mocks.NewMockUserStorage(ctrl)
	pwHasher := sha256.New()
	rr := httptest.NewRecorder()
	router := gin.Default()
	handler := NewCreateHandler(userStorage, http.Client{}, pwHasher)
	router.POST("/test", application.AsGinHandler(handler.Handle))
	req := &storage.CreateUserRequest{
		Username: "user",
		Password: "password",
	}
	storageReq := &storage.CreateUserRequest{
		Username:  req.Username,
		Password:  hasher.HashString(pwHasher, req.Password),
		AvatarURL: defaultAvatarURL,
	}
	raw, err := json.Marshal(req)
	require.NoError(t, err)
	request, err := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(raw))
	require.NoError(t, err)

	userStorage.EXPECT().CreateUser(gomock.Any(), storageReq).Return(nil, nil)

	router.ServeHTTP(rr, request)
	require.Equal(t, http.StatusOK, rr.Code)
}

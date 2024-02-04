package user

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
				AvatarURL: "https://image.winudf.com/v2/image/Y29tLmtub3dpbmdhbmltYWxzLmFwcF9zY3JlZW5fMTNfMTUzMDQ4MzkzOV8wODY/screen-13.jpg?fakeurl=1&type=.jpg",
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
			statusCode: http.StatusUnsupportedMediaType,
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
	pwHasher := hasher.NewNopHasher()
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
			if tc.wantStore {
				userStorage.EXPECT().CreateUser(gomock.Any(), tc.req).Return(nil, tc.storeErr)
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
	pwHasher := hasher.NewNopHasher()
	rr := httptest.NewRecorder()
	router := gin.Default()
	handler := NewCreateHandler(userStorage, http.Client{}, pwHasher)
	router.POST("/test", application.AsGinHandler(handler.Handle))
	req := &storage.CreateUserRequest{
		Username: "user",
		Password: "password",
	}
	storageReq := &storage.CreateUserRequest{
		Username: req.Username,
		Password: req.Password,
	}
	raw, err := json.Marshal(req)
	require.NoError(t, err)
	request, err := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(raw))
	require.NoError(t, err)

	userStorage.EXPECT().CreateUser(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, req *storage.CreateUserRequest) (*storage.User, error) {
		require.Equal(t, storageReq.Username, req.Username)
		require.Equal(t, storageReq.Password, req.Password)
		require.True(t, strings.HasPrefix(req.AvatarURL, defaultAvatarURLTmpl))

		return nil, nil
	})

	router.ServeHTTP(rr, request)
	require.Equal(t, http.StatusOK, rr.Code)
}

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
			getReq:     &storage.GetUserRequest{Id: "id"},
			statusCode: http.StatusOK,
		},
		{
			name:       "not found",
			id:         "non_existent_id",
			getReq:     &storage.GetUserRequest{Id: "non_existent_id"},
			getErr:     storage.ErrNotFound,
			statusCode: http.StatusNotFound,
		},
		{
			name:       "internal error",
			id:         "id",
			getReq:     &storage.GetUserRequest{Id: "id"},
			getErr:     xerrors.New("oops"),
			statusCode: http.StatusInternalServerError,
		},
	}

	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	userStorage := mocks.NewMockUserStorage(ctrl)
	router := gin.Default()
	handler := NewGetHandler(userStorage)
	router.GET("/test/:id", application.AsGinHandler(handler.Handle))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, "/test/"+tc.id, nil)
			require.NoError(t, err)

			userStorage.EXPECT().GetUser(gomock.Any(), tc.getReq).Return(nil, tc.getErr)

			router.ServeHTTP(rr, request)
			require.Equal(t, tc.statusCode, rr.Code)
		})
	}
}

func TestUpdateHandler(t *testing.T) {
	testCases := []struct {
		name       string
		imgType    string
		req        *storage.UpdateUserRequest
		wantUpd    bool
		updErr     error
		statusCode int
	}{
		{
			name:    "ok",
			imgType: "image/svg",
			req: &storage.UpdateUserRequest{
				Id:          "1",
				Username:    "user",
				OldPassword: "password",
			},
			wantUpd:    true,
			statusCode: http.StatusOK,
		},
		{
			name: "ok with custom avatar link",
			req: &storage.UpdateUserRequest{
				Id:          "1",
				Username:    "user",
				OldPassword: "password",
				AvatarURL:   "https://image.winudf.com/v2/image/Y29tLmtub3dpbmdhbmltYWxzLmFwcF9zY3JlZW5fMTNfMTUzMDQ4MzkzOV8wODY/screen-13.jpg?fakeurl=1&type=.jpg",
			},
			wantUpd:    true,
			statusCode: http.StatusOK,
		},
		{
			name:    "not found",
			imgType: "image/svg",
			req: &storage.UpdateUserRequest{
				Id:          "non_existent_id",
				Username:    "user",
				OldPassword: "password",
			},
			wantUpd:    true,
			updErr:     storage.ErrNotFound,
			statusCode: http.StatusNotFound,
		},
		{
			name:    "not an image",
			imgType: "application/json",
			req: &storage.UpdateUserRequest{
				Id:          "1",
				Username:    "user",
				OldPassword: "password",
			},
			wantUpd:    false,
			statusCode: http.StatusUnsupportedMediaType,
		},
		{
			name:    "internal error",
			imgType: "image/jpg",
			req: &storage.UpdateUserRequest{
				Id:          "1",
				Username:    "user",
				OldPassword: "password",
			},
			wantUpd:    true,
			updErr:     xerrors.New("oops"),
			statusCode: http.StatusInternalServerError,
		},
	}

	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	userStorage := mocks.NewMockUserStorage(ctrl)
	router := gin.Default()
	pwHasher := hasher.NewNopHasher()
	handler := NewUpdateHandler(userStorage, http.Client{}, pwHasher)
	router.POST("/test/:id", application.AsGinHandler(handler.Handle))

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
			request, err := http.NewRequest(http.MethodPost, "/test/"+tc.req.Id, bytes.NewReader(raw))
			require.NoError(t, err)

			if tc.wantUpd {
				userStorage.EXPECT().GetUserPasswordHash(gomock.Any(), &storage.GetUserRequest{Id: tc.req.Id, Username: tc.req.Username}).Return(tc.req.OldPassword, nil)
				userStorage.EXPECT().UpdateUser(gomock.Any(), tc.req).Return(&storage.User{}, tc.updErr)
			}

			router.ServeHTTP(rr, request)
			require.Equal(t, tc.statusCode, rr.Code)
		})
	}
}

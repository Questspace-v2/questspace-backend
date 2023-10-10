package user

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"questspace/pkg/application"
	"questspace/pkg/storage"
	"questspace/pkg/storage/mocks"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
)

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
				Id:       "1",
				Username: "user",
				Password: "password",
			},
			wantUpd:    true,
			statusCode: http.StatusOK,
		},
		{
			name: "ok with custom avatar link",
			req: &storage.UpdateUserRequest{
				Id:        "1",
				Username:  "user",
				Password:  "password",
				AvatarURL: "https://some.domain.com/avatar.png",
			},
			wantUpd:    true,
			statusCode: http.StatusOK,
		},
		{
			name:    "not found",
			imgType: "image/svg",
			req: &storage.UpdateUserRequest{
				Id:       "non_existent_id",
				Username: "user",
				Password: "password",
			},
			wantUpd:    true,
			updErr:     storage.ErrNotFound,
			statusCode: http.StatusNotFound,
		},
		{
			name:    "not an image",
			imgType: "application/json",
			req: &storage.UpdateUserRequest{
				Id:       "1",
				Username: "user",
				Password: "password",
			},
			wantUpd:    false,
			statusCode: http.StatusUnprocessableEntity,
		},
		{
			name:    "internal error",
			imgType: "image/jpg",
			req: &storage.UpdateUserRequest{
				Id:       "1",
				Username: "user",
				Password: "password",
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
	handler := NewUpdateHandler(userStorage, http.Client{})
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
				userStorage.EXPECT().UpdateUser(gomock.Any(), tc.req).Return(&storage.User{}, tc.updErr)
			}

			router.ServeHTTP(rr, request)
			require.Equal(t, tc.statusCode, rr.Code)
		})
	}
}

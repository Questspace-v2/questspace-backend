package user

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"questspace/pkg/application"
	"questspace/pkg/storage"
	"questspace/pkg/storage/mocks"
	"testing"

	"golang.org/x/xerrors"

	"github.com/stretchr/testify/require"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
)

func TestGetHandler_HandleQS(t *testing.T) {
	testCases := []struct {
		name       string
		query      map[string]string
		wantGet    bool
		getReq     *storage.GetUserRequest
		getErr     error
		statusCode int
	}{
		{
			name: "ok",
			query: map[string]string{
				"id": "id",
			},
			wantGet:    true,
			getReq:     &storage.GetUserRequest{Id: "id"},
			statusCode: http.StatusOK,
		},
		{
			name: "ok with both values",
			query: map[string]string{
				"id":       "id",
				"username": "username",
			},
			wantGet:    true,
			getReq:     &storage.GetUserRequest{Id: "id", Username: "username"},
			statusCode: http.StatusOK,
		},
		{
			name:       "no args bad request",
			wantGet:    false,
			statusCode: http.StatusBadRequest,
		},
		{
			name: "not found",
			query: map[string]string{
				"id": "non_existent_id",
			},
			wantGet:    true,
			getReq:     &storage.GetUserRequest{Id: "non_existent_id"},
			getErr:     storage.ErrNotFound,
			statusCode: http.StatusNotFound,
		},
		{
			name: "internal error",
			query: map[string]string{
				"id": "id",
			},
			wantGet:    true,
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
	router.GET("/test", application.AsGinHandler(handler.HandleQS))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			qs := url.Values{}
			for k, v := range tc.query {
				qs.Set(k, v)
			}
			request, err := http.NewRequest(http.MethodGet, "/test?"+qs.Encode(), nil)
			require.NoError(t, err)

			if tc.wantGet {
				userStorage.EXPECT().GetUser(gomock.Any(), tc.getReq).Return(nil, tc.getErr)
			}

			router.ServeHTTP(rr, request)
			require.Equal(t, tc.statusCode, rr.Code)
		})
	}
}

func TestGetHandler_HandlePath(t *testing.T) {
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
	router.GET("/test/:id", application.AsGinHandler(handler.HandlePath))

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

package pgclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"questspace/internal/pgdb/pgtest"
	"questspace/pkg/storage"
)

var (
	createUserReq = storage.CreateUserRequest{
		Username:  "svayp11",
		Password:  "veryverysecure",
		AvatarURL: "https://google.com",
	}
	newUsername = "newusername"
	newAvatar   = "https://ya.ru"
	newPW       = "insecure"
)

func TestUserStorage_CreateUser(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	userReq := createUserReq
	user, err := client.CreateUser(ctx, &userReq)
	require.NoError(t, err)

	assert.Equal(t, userReq.Username, user.Username)
	assert.Equal(t, userReq.AvatarURL, user.AvatarURL)
	assert.Emptyf(t, user.Password, "Returned user must not contain password")
	assert.NotEmpty(t, user.ID)
}

func TestUserStorage_CreateUser_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	userReq := createUserReq
	user, err := client.CreateUser(ctx, &userReq)
	require.NoError(t, err)
	assert.NotNil(t, user)

	user2, err := client.CreateUser(ctx, &userReq)
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrExists)
	assert.Nil(t, user2)
}

func TestUserStorage_GetUser(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	userReq := createUserReq
	created, err := client.CreateUser(ctx, &userReq)
	require.NoError(t, err)

	gotByID, err := client.GetUser(ctx, &storage.GetUserRequest{ID: created.ID})
	require.NoError(t, err)
	assert.Equal(t, *created, *gotByID)

	gotByUsername, err := client.GetUser(ctx, &storage.GetUserRequest{Username: created.Username})
	require.NoError(t, err)
	assert.Equal(t, *created, *gotByUsername)
}

func TestUserStorage_GetUser_NotFound(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	gotByID, err := client.GetUser(ctx, &storage.GetUserRequest{ID: storage.NewID()})
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
	assert.Nil(t, gotByID)
}

func TestUserStorage_GetUser_ErrorOnEmptyRequest(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	got, err := client.GetUser(ctx, &storage.GetUserRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrValidation)
	assert.Nil(t, got)
}

func TestUserStorage_UpdateUser(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	userReq := createUserReq
	created, err := client.CreateUser(ctx, &userReq)
	require.NoError(t, err)
	createdHash, err := client.GetUserPasswordHash(ctx, &storage.GetUserRequest{ID: created.ID})
	require.NoError(t, err)
	assert.NotEmpty(t, createdHash)

	updatedName, err := client.UpdateUser(ctx, &storage.UpdateUserRequest{ID: created.ID, Username: newUsername})
	require.NoError(t, err)
	assert.Equal(t, created.ID, updatedName.ID)
	assert.Equal(t, created.AvatarURL, updatedName.AvatarURL)
	assert.Equal(t, created.Password, updatedName.Password)
	assert.NotEqual(t, created.Username, updatedName.Username)
	assert.Equal(t, newUsername, updatedName.Username)

	updatedAvatar, err := client.UpdateUser(ctx, &storage.UpdateUserRequest{ID: created.ID, AvatarURL: newAvatar})
	require.NoError(t, err)
	assert.Equal(t, created.ID, updatedAvatar.ID)
	assert.Equal(t, created.Password, updatedAvatar.Password)
	assert.NotEqual(t, created.AvatarURL, updatedAvatar.AvatarURL)
	assert.Equal(t, updatedName.Username, updatedAvatar.Username)
	assert.Equal(t, newAvatar, updatedAvatar.AvatarURL)

	updatedPW, err := client.UpdateUser(ctx, &storage.UpdateUserRequest{ID: created.ID, Password: newPW})
	require.NoError(t, err)
	assert.Equal(t, created.ID, updatedPW.ID)
	assert.Equal(t, created.Password, updatedPW.Password)
	assert.Equal(t, updatedName.Username, updatedPW.Username)
	assert.Equal(t, updatedAvatar.AvatarURL, updatedPW.AvatarURL)

	updatedHash, err := client.GetUserPasswordHash(ctx, &storage.GetUserRequest{ID: created.ID})
	require.NoError(t, err)
	assert.NotEqual(t, createdHash, updatedHash)
	assert.Equal(t, newPW, updatedHash)
}

func TestUserStorage_UpdateUser_NotFound(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	upd, err := client.UpdateUser(ctx, &storage.UpdateUserRequest{ID: storage.NewID(), Username: newUsername})
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
	assert.Nil(t, upd)
}

func TestUserStorage_UpdateUser_ErrorOnEmptyRequest(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	upd, err := client.UpdateUser(ctx, &storage.UpdateUserRequest{ID: storage.NewID()})
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrValidation)
	assert.Nil(t, upd)
}

func TestUserStorage_UpdateUser_ErrOnAlreadyExists(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	userReq := createUserReq
	user, err := client.CreateUser(ctx, &userReq)
	require.NoError(t, err)
	require.NotNil(t, user)

	createReq2 := createUserReq
	createReq2.Username = "new"
	user2, err := client.CreateUser(ctx, &createReq2)
	require.NoError(t, err)
	require.NotNil(t, user2)

	updateReq := storage.UpdateUserRequest{
		ID:       user2.ID,
		Username: user.Username,
	}
	updated, err := client.UpdateUser(ctx, &updateReq)
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrExists)
	assert.Nil(t, updated)
}

func TestUserStorage_GetUserPasswordHash(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	userReq := createUserReq
	created, err := client.CreateUser(ctx, &userReq)
	require.NoError(t, err)

	pwHashByID, err := client.GetUserPasswordHash(ctx, &storage.GetUserRequest{ID: created.ID})
	require.NoError(t, err)
	assert.Equal(t, createUserReq.Password, pwHashByID)

	pwHashByUsername, err := client.GetUserPasswordHash(ctx, &storage.GetUserRequest{Username: created.Username})
	require.NoError(t, err)
	assert.Equal(t, createUserReq.Password, pwHashByUsername)
}

func TestUserStorage_GetUserPasswordHash_NotFound(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	pwHash, err := client.GetUserPasswordHash(ctx, &storage.GetUserRequest{ID: storage.NewID()})
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
	assert.Empty(t, pwHash)
}

func TestUserStorage_GetUserPasswordHash_ErrorOnEmptyRequest(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	pwHash, err := client.GetUserPasswordHash(ctx, &storage.GetUserRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrValidation)
	assert.Empty(t, pwHash)
}

func TestUserStorage_CreateOrUpdateByExternalID(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	req := storage.CreateOrUpdateRequest{
		ExternalID:        "123",
		CreateUserRequest: *userReq1,
	}
	user, err := client.CreateOrUpdateByExternalID(ctx, &req)
	require.NoError(t, err)

	assert.Equal(t, userReq1.Username, user.Username)
	assert.Equal(t, userReq1.AvatarURL, user.AvatarURL)

	req.AvatarURL = "https://golang.org"
	newUser, err := client.CreateOrUpdateByExternalID(ctx, &req)
	require.NoError(t, err)

	assert.Equal(t, userReq1.AvatarURL, newUser.AvatarURL)
}

func TestUserStorage_DeleteUser(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	userReq := createUserReq
	created, err := client.CreateUser(ctx, &userReq)
	require.NoError(t, err)

	require.NoError(t, client.DeleteUser(ctx, &storage.DeleteUserRequest{ID: created.ID}))

	got, err := client.GetUser(ctx, &storage.GetUserRequest{ID: created.ID})
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
	assert.Nil(t, got)
}

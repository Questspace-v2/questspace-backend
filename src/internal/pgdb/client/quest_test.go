package pgdb

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/spkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"questspace/internal/pgdb/pgtest"
	"questspace/pkg/storage"
)

func assertQuestEqual(t *testing.T, expected, actual storage.Quest) {
	t.Helper()
	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.Access, actual.Access)
	assert.Equal(t, expected.StartTime.Unix(), actual.StartTime.Unix())
	assert.Equal(t, expected.MediaLink, actual.MediaLink)
	if expected.FinishTime != nil {
		require.NotNil(t, actual.FinishTime)
		assert.Equal(t, expected.FinishTime.Unix(), actual.FinishTime.Unix())
	} else {
		assert.Nil(t, actual.FinishTime)
	}
	if expected.RegistrationDeadline != nil {
		require.NotNil(t, actual.RegistrationDeadline)
		assert.Equal(t, expected.RegistrationDeadline.Unix(), actual.RegistrationDeadline.Unix())
	} else {
		assert.Nil(t, actual.RegistrationDeadline)
	}
	if expected.MaxTeamCap != nil {
		require.NotNil(t, actual.MaxTeamCap)
		assert.Equal(t, expected.MaxTeamCap, actual.MaxTeamCap)
	} else {
		assert.Nil(t, actual.MaxTeamCap)
	}
}

func TestQuestStorage_GetQuest(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))
	user, err := client.CreateUser(ctx, &storage.CreateUserRequest{
		Username:  "svayp11",
		AvatarURL: "avatar_url",
	})
	require.NoError(t, err)
	now := time.Now().UTC()

	testCases := []struct {
		name      string
		createReq storage.CreateQuestRequest
		expected  storage.Quest
	}{
		{
			name: "necessary fields only",
			createReq: storage.CreateQuestRequest{
				Name:      "new_quest",
				Creator:   &storage.User{Username: user.Username},
				Access:    storage.Public,
				StartTime: &now,
			},
			expected: storage.Quest{
				Name:      "new_quest",
				Creator:   user,
				Access:    storage.Public,
				StartTime: &now,
			},
		},
		{
			name: "all fields",
			createReq: storage.CreateQuestRequest{
				Name:                 "full",
				Creator:              user,
				Access:               storage.LinkOnly,
				Description:          "some desc",
				RegistrationDeadline: &now,
				StartTime:            &now,
				FinishTime:           &now,
				MediaLink:            "https://ya.ru",
				MaxTeamCap:           ptr.Int(3),
			},
			expected: storage.Quest{
				Name:                 "full",
				Creator:              user,
				Access:               storage.LinkOnly,
				Description:          "some desc",
				RegistrationDeadline: &now,
				StartTime:            &now,
				FinishTime:           &now,
				MediaLink:            "https://ya.ru",
				MaxTeamCap:           ptr.Int(3),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cr := tc.createReq
			q, err := client.CreateQuest(ctx, &cr)
			require.NoError(t, err)
			require.NotNil(t, q)
			assert.Equal(t, *cr.Creator, *q.Creator)

			got, err := client.GetQuest(ctx, &storage.GetQuestRequest{ID: q.ID})
			require.NoError(t, err)
			require.NotNil(t, got)
			assertQuestEqual(t, *q, *got)
			assert.Equal(t, *user, *got.Creator)
		})
	}
}

func TestQuestStorage_GetQuest_NotFound(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	got, err := client.GetQuest(ctx, &storage.GetQuestRequest{ID: uuid.Must(uuid.NewV4()).String()})
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
	assert.Nil(t, got)
}

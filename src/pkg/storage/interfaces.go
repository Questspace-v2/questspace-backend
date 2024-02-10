package storage

import "context"

//go:generate mockgen -source=interfaces.go -destination mocks/client.go -package mocks
type QuestSpaceStorage interface {
	UserStorage
	QuestStorage
}

type UserStorage interface {
	CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
	GetUser(ctx context.Context, req *GetUserRequest) (*User, error)
	UpdateUser(ctx context.Context, req *UpdateUserRequest) (*User, error)
	GetUserPasswordHash(ctx context.Context, req *GetUserRequest) (string, error)
	DeleteUser(ctx context.Context, req *DeleteUserRequest) error
}

type QuestStorage interface {
	CreateQuest(ctx context.Context, req *CreateQuestRequest) (*Quest, error)
	GetQuest(ctx context.Context, req *GetQuestRequest) (*Quest, error)
	UpdateQuest(ctx context.Context, req *UpdateQuestRequest) (*Quest, error)
	DeleteQuest(ctx context.Context, req *DeleteQuestRequest) error
}

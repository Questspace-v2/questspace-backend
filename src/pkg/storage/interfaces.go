package storage

import "context"

//go:generate mockgen -source=interfaces.go -destination mocks/client.go -package mocks
type QuestSpaceStorage interface {
	UserStorage
	QuestStorage
	TaskGroupStorage
}

type UserStorage interface {
	CreateUser(context.Context, *CreateUserRequest) (*User, error)
	GetUser(context.Context, *GetUserRequest) (*User, error)
	UpdateUser(context.Context, *UpdateUserRequest) (*User, error)
	GetUserPasswordHash(context.Context, *GetUserRequest) (string, error)
	DeleteUser(context.Context, *DeleteUserRequest) error
}

type QuestStorage interface {
	CreateQuest(context.Context, *CreateQuestRequest) (*Quest, error)
	GetQuest(context.Context, *GetQuestRequest) (*Quest, error)
	UpdateQuest(context.Context, *UpdateQuestRequest) (*Quest, error)
	DeleteQuest(context.Context, *DeleteQuestRequest) error
}

type TaskGroupStorage interface {
	CreateTaskGroup(context.Context, *CreateTaskGroupRequest) (*TaskGroup, error)
	GetTaskGroup(context.Context, *GetTaskGroupRequest) (*TaskGroup, error)
	GetTaskGroups(context.Context, *GetTaskGroupsRequest) ([]*TaskGroup, error)
	UpdateTaskGroup(context.Context, *UpdateTaskGroupRequest) (*TaskGroup, error)
	DeleteTaskGroup(context.Context, *DeleteTaskGroupRequest) error
}

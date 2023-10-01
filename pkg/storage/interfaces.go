package storage

//go:generate mockgen -source=interfaces.go -destination mocks/client.go -package mocks
type QuestSpaceStorage interface {
	QuestStorage
	UserStorage
	TeamStorage
}

type QuestStorage interface {
	CreateQuest(req *CreateQuestRequest) (*Quest, error)
	GetQuest(req *GetQuestRequest) (*Quest, error)
	UpdateQuest(req *UpdateQuestRequest) (*Quest, error)
	DeleteQuest(req *DeleteQuestRequest) error
}

type UserStorage interface {
	CreateUser(req *CreateUserRequest) (*User, error)
	GetUser(req *GetUserRequest) (*User, error)
	UpdateUser(req *UpdateUserRequest) (*User, error)
	DeleteUser(req *DeleteUserRequest) error
}

type TeamStorage interface {
	CreateTeam(req *CreateTeamRequest) (*Team, error)
}

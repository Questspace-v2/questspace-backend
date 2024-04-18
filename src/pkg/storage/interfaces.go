package storage

import "context"

//go:generate mockgen -source=interfaces.go -destination mocks/client.go -package mocks
type QuestSpaceStorage interface {
	UserStorage
	QuestStorage
	TaskGroupStorage
	TaskStorage
	TeamStorage
	AnswerHintStorage
}

type UserStorage interface {
	CreateUser(context.Context, *CreateUserRequest) (*User, error)
	GetUser(context.Context, *GetUserRequest) (*User, error)
	UpdateUser(context.Context, *UpdateUserRequest) (*User, error)
	GetUserPasswordHash(context.Context, *GetUserRequest) (string, error)
	CreateOrUpdateByExternalID(context.Context, *CreateOrUpdateRequest) (*User, error)
	DeleteUser(context.Context, *DeleteUserRequest) error
}

type QuestStorage interface {
	CreateQuest(context.Context, *CreateQuestRequest) (*Quest, error)
	GetQuest(context.Context, *GetQuestRequest) (*Quest, error)
	GetQuests(context.Context, *GetQuestsRequest) (*GetQuestsResponse, error)
	UpdateQuest(context.Context, *UpdateQuestRequest) (*Quest, error)
	DeleteQuest(context.Context, *DeleteQuestRequest) error
}

type TaskGroupStorage interface {
	CreateTaskGroup(context.Context, *CreateTaskGroupRequest) (*TaskGroup, error)
	GetTaskGroup(context.Context, *GetTaskGroupRequest) (*TaskGroup, error)
	GetTaskGroups(context.Context, *GetTaskGroupsRequest) ([]TaskGroup, error)
	UpdateTaskGroup(context.Context, *UpdateTaskGroupRequest) (*TaskGroup, error)
	DeleteTaskGroup(context.Context, *DeleteTaskGroupRequest) error
}

type TaskStorage interface {
	CreateTask(context.Context, *CreateTaskRequest) (*Task, error)
	GetTask(context.Context, *GetTaskRequest) (*Task, error)
	GetAnswerData(context.Context, *GetTaskRequest) (*Task, error)
	GetTasks(context.Context, *GetTasksRequest) (GetTasksResponse, error)
	UpdateTask(context.Context, *UpdateTaskRequest) (*Task, error)
	DeleteTask(context.Context, *DeleteTaskRequest) error
}

type TeamStorage interface {
	CreateTeam(context.Context, *CreateTeamRequest) (*Team, error)
	GetTeam(context.Context, *GetTeamRequest) (*Team, error)
	GetTeams(context.Context, *GetTeamsRequest) ([]Team, error)
	ChangeTeamName(context.Context, *ChangeTeamNameRequest) (*Team, error)
	SetInviteLink(context.Context, *SetInvitePathRequest) error
	JoinTeam(context.Context, *JoinTeamRequest) (*User, error)
	DeleteTeam(context.Context, *DeleteTeamRequest) error
	ChangeLeader(context.Context, *ChangeLeaderRequest) (*Team, error)
	RemoveUser(context.Context, *RemoveUserRequest) error
}

type AnswerHintStorage interface {
	AnswerStorage
	HintStorage
}

type HintStorage interface {
	GetHintTakes(context.Context, *GetHintTakesRequest) (HintTakes, error)
	TakeHint(context.Context, *TakeHintRequest) (*Hint, error)
}

type AnswerStorage interface {
	GetAcceptedTasks(context.Context, *GetAcceptedTasksRequest) (AcceptedTasks, error)
	CreateAnswerTry(context.Context, *CreateAnswerTryRequest) error
	GetScoreResults(context.Context, *GetResultsRequest) (ScoreResults, error)
}

package storage

import (
	"time"
)

type CreateUserRequest struct {
	Username  string `json:"username" example:"svayp11"`
	Password  string `json:"password,omitempty" example:"12345"`
	AvatarURL string `json:"avatar_url,omitempty" example:"https://api.dicebear.com/7.x/thumbs/svg"`
}

type GetUserRequest struct {
	ID       string
	Username string
}

type UpdateUserRequest struct {
	ID        string
	Username  string
	Password  string
	AvatarURL string
}

type CreateOrUpdateRequest struct {
	CreateUserRequest

	ExternalID string
}

type DeleteUserRequest struct {
	ID string
}

type CreateQuestRequest struct {
	Name                 string     `json:"name"`
	Description          string     `json:"description,omitempty"`
	Access               AccessType `json:"access"`
	Creator              *User      `json:"-"`
	RegistrationDeadline *time.Time `json:"registration_deadline,omitempty" example:"2024-04-14T12:00:00+05:00"`
	StartTime            *time.Time `json:"start_time" example:"2024-04-14T14:00:00+05:00"`
	FinishTime           *time.Time `json:"finish_time,omitempty" example:"2024-04-21T14:00:00+05:00"`
	MediaLink            string     `json:"media_link"`
	MaxTeamCap           *int       `json:"max_team_cap,omitempty"`
}

type GetQuestRequest struct {
	ID string
}

type GetQuestsRequest struct {
	User     *User
	Type     GetQuestType
	PageSize int
	Page     *Page
}

type GetQuestsResponse struct {
	Quests   []Quest
	NextPage *Page
}

type UpdateQuestRequest struct {
	ID                   string     `json:"-"`
	Name                 string     `json:"name,omitempty"`
	Description          string     `json:"description,omitempty"`
	Access               AccessType `json:"access,omitempty"`
	RegistrationDeadline *time.Time `json:"registration_deadline,omitempty"`
	StartTime            *time.Time `json:"start_time,omitempty"`
	FinishTime           *time.Time `json:"finish_time,omitempty"`
	MediaLink            string     `json:"media_link,omitempty"`
	MaxTeamCap           *int       `json:"max_team_cap,omitempty"`
}

type DeleteQuestRequest struct {
	ID string
}

type CreateTeamRequest struct {
	Name    string
	QuestID string
	Creator *User
}

type UserRegistration struct {
	UserID  string
	QuestID string
}

type GetTeamRequest struct {
	ID               string
	InvitePath       string
	UserRegistration *UserRegistration
	IncludeMembers   bool
}

type GetTeamsRequest struct {
	User     *User
	QuestIDs []string
}

type ChangeTeamNameRequest struct {
	ID   string
	Name string
}

type SetInvitePathRequest struct {
	TeamID     string
	InvitePath string
}

type JoinTeamRequest struct {
	InvitePath string
	User       *User
}

type DeleteTeamRequest struct {
	ID string
}

type ChangeLeaderRequest struct {
	ID        string
	CaptainID string
}

type RemoveUserRequest struct {
	ID     string
	UserID string
}

type CreateTaskGroupRequest struct {
	QuestID  string              `json:"-"`
	OrderIdx int                 `json:"order_idx"`
	Name     string              `json:"name"`
	PubTime  *time.Time          `json:"pub_time,omitempty"`
	Tasks    []CreateTaskRequest `json:"tasks"`
}

type GetTaskGroupRequest struct {
	ID           string
	IncludeTasks bool
}

type GetTaskGroupsRequest struct {
	QuestID      string
	IncludeTasks bool
}

type UpdateTaskGroupRequest struct {
	QuestID  string                  `json:"-"`
	ID       string                  `json:"id"`
	OrderIdx int                     `json:"order_idx"`
	Name     string                  `json:"name"`
	PubTime  *time.Time              `json:"pub_time"`
	Tasks    *TasksBulkUpdateRequest `json:"tasks"`
}

type DeleteTaskGroupRequest struct {
	ID string `json:"id"`
}

type TaskGroupsBulkUpdateRequest struct {
	QuestID string                   `json:"-"`
	Create  []CreateTaskGroupRequest `json:"create"`
	Update  []UpdateTaskGroupRequest `json:"update"`
	Delete  []DeleteTaskGroupRequest `json:"delete"`
}

type CreateTaskRequest struct {
	OrderIdx       int              `json:"order_idx"`
	GroupID        string           `json:"group_id"`
	Name           string           `json:"name"`
	Question       string           `json:"question"`
	Reward         int              `json:"reward"`
	CorrectAnswers []string         `json:"correct_answers"`
	Verification   VerificationType `json:"verification"`
	Hints          []string         `json:"hints"`
	PubTime        *time.Time       `json:"pub_time"`
	MediaLink      string           `json:"media_link"`
}

type GetTaskRequest struct {
	ID string
}

type GetTasksRequest struct {
	GroupIDs []string
	QuestID  string
}

type GetTasksResponse map[string][]Task

type UpdateTaskRequest struct {
	QuestID        string           `json:"-"`
	ID             string           `json:"id"`
	OrderIdx       int              `json:"order_idx"`
	GroupID        string           `json:"group_id"`
	Name           string           `json:"name"`
	Question       string           `json:"question"`
	Reward         int              `json:"reward"`
	CorrectAnswers []string         `json:"correct_answers"`
	Verification   VerificationType `json:"verification"`
	Hints          []string         `json:"hints"`
	PubTime        *time.Time       `json:"pub_time"`
	MediaLink      string           `json:"media_link"`
}

type DeleteTaskRequest struct {
	ID string `json:"id"`
}

type TasksBulkUpdateRequest struct {
	QuestID string              `json:"-"`
	GroupID string              `json:"-"`
	Create  []CreateTaskRequest `json:"create"`
	Update  []UpdateTaskRequest `json:"update"`
	Delete  []DeleteTaskRequest `json:"delete"`
}

type GetHintTakesRequest struct {
	TeamID  string
	QuestID string
	TaskID  string
}

type TakeHintRequest struct {
	TeamID string
	TaskID string
	Index  int
}

type GetAcceptedTasksRequest struct {
	TeamID  string
	QuestID string
}

type CreateAnswerTryRequest struct {
	TeamID   string
	TaskID   string
	Text     string
	Accepted bool
	Score    int
}

type GetResultsRequest struct {
	QuestID string
}

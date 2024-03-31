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
	RegistrationDeadline *time.Time `json:"registration_deadline"`
	StartTime            *time.Time `json:"start_time"`
	FinishTime           *time.Time `json:"finish_time"`
	MediaLink            string     `json:"media_link"`
	MaxTeamCap           *int       `json:"max_team_cap"`
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
	Quests   []*Quest
	NextPage *Page
}

type UpdateQuestRequest struct {
	ID                   string     `json:"-"`
	Name                 string     `json:"name,omitempty"`
	Description          string     `json:"description,omitempty"`
	Access               AccessType `json:"access,omitempty"`
	CreatorName          string     `json:"creator_name,omitempty"`
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

type GetTeamRequest struct {
	ID             string
	InvitePath     string
	IncludeMembers bool
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
	QuestID  string     `json:"-"`
	OrderIdx int        `json:"order_idx"`
	Name     string     `json:"name"`
	PubTime  *time.Time `json:"pub_time,omitempty"`
}

type GetTaskGroupRequest struct {
	ID string
}

type GetTaskGroupsRequest struct {
	QuestID string
}

type UpdateTaskGroupRequest struct {
	QuestID  string     `json:"-"`
	ID       string     `json:"id"`
	OrderIdx int        `json:"order_idx"`
	Name     string     `json:"name"`
	PubTime  *time.Time `json:"pub_time"`
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

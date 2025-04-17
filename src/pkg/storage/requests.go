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
	ID       ID
	Username string
}

type UpdateUserRequest struct {
	ID        ID
	Username  string
	Password  string
	AvatarURL string
}

type CreateOrUpdateRequest struct {
	CreateUserRequest

	ExternalID string
}

type DeleteUserRequest struct {
	ID ID
}

type CreateQuestRequest struct {
	Name                 string           `json:"name"`
	Description          string           `json:"description,omitempty"`
	Access               AccessType       `json:"access"`
	Creator              *User            `json:"-"`
	RegistrationDeadline *time.Time       `json:"registration_deadline,omitempty" example:"2024-04-14T12:00:00+05:00"`
	StartTime            *time.Time       `json:"start_time" example:"2024-04-14T14:00:00+05:00"`
	FinishTime           *time.Time       `json:"finish_time,omitempty" example:"2024-04-21T14:00:00+05:00"`
	MediaLink            string           `json:"media_link"`
	MaxTeamCap           *int             `json:"max_team_cap,omitempty"`
	HasBrief             bool             `json:"has_brief,omitempty"`
	Brief                string           `json:"brief,omitempty"`
	MaxTeamsAmount       *int             `json:"max_teams_amount,omitempty"`
	RegistrationType     RegistrationType `json:"registration_type,omitempty" enums:"AUTO,VERIFY"`
	QuestType            QuestType        `json:"quest_type,omitempty" enums:"ASSAULT,LINEAR"`
	FeedbackLink         *string          `json:"feedback_link,omitempty"`
}

type GetQuestRequest struct {
	ID ID
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
	ID                   ID               `json:"-"`
	Name                 string           `json:"name,omitempty"`
	Description          string           `json:"description,omitempty"`
	Access               AccessType       `json:"access,omitempty"`
	RegistrationDeadline *time.Time       `json:"registration_deadline,omitempty"`
	StartTime            *time.Time       `json:"start_time,omitempty"`
	FinishTime           *time.Time       `json:"finish_time,omitempty"`
	MediaLink            string           `json:"media_link,omitempty"`
	MaxTeamCap           *int             `json:"max_team_cap,omitempty"`
	HasBrief             *bool            `json:"has_brief,omitempty"`
	Brief                *string          `json:"brief,omitempty"`
	MaxTeamsAmount       *int             `json:"max_teams_amount,omitempty"`
	RegistrationType     RegistrationType `json:"registration_type,omitempty" enums:"AUTO,VERIFY"`
	QuestType            QuestType        `json:"quest_type,omitempty" enums:"ASSAULT,LINEAR"`
	FeedbackLink         *string          `json:"feedback_link,omitempty"`
}

type DeleteQuestRequest struct {
	ID ID
}

type FinishQuestRequest struct {
	ID ID
}

type CreateTeamRequest struct {
	Name               string
	QuestID            ID
	Creator            *User
	RegistrationStatus RegistrationStatus
}

type UserRegistration struct {
	UserID  ID
	QuestID ID
}

type GetTeamRequest struct {
	ID               ID
	InvitePath       string
	UserRegistration *UserRegistration
	IncludeMembers   bool
}

type GetTeamsRequest struct {
	User           *User
	QuestIDs       []ID
	IncludeMembers bool
	AcceptedOnly   bool
}

type ChangeTeamNameRequest struct {
	ID   ID
	Name string
}

type SetInvitePathRequest struct {
	TeamID     ID
	InvitePath string
}

type JoinTeamRequest struct {
	InvitePath string
	User       *User
}

type AcceptTeamRequest struct {
	ID ID
}

type DeleteTeamRequest struct {
	ID ID
}

type ChangeLeaderRequest struct {
	ID        ID
	CaptainID ID
}

type RemoveUserRequest struct {
	ID     ID
	UserID ID
}

type CreateTaskGroupRequest struct {
	QuestID      ID                  `json:"-"`
	OrderIdx     int                 `json:"order_idx"`
	Name         string              `json:"name"`
	Description  string              `json:"description,omitempty"`
	PubTime      *time.Time          `json:"pub_time,omitempty"`
	Tasks        []CreateTaskRequest `json:"tasks"`
	Sticky       bool                `json:"sticky,omitempty"`
	HasTimeLimit bool                `json:"has_time_limit,omitempty"`
	TimeLimit    *Duration           `json:"time_limit,omitempty"`
}

type TeamData struct {
	UserID *ID
	TeamID *ID
}

type GetTaskGroupRequest struct {
	ID           ID
	IncludeTasks bool
	TeamData     *TeamData
}

type GetTaskGroupsRequest struct {
	QuestID      ID
	GroupIDs     []ID
	IncludeTasks bool
	TeamData     *TeamData
}

type UpdateTaskGroupRequest struct {
	QuestID      ID                      `json:"-"`
	ID           ID                      `json:"id"`
	OrderIdx     int                     `json:"order_idx"`
	Name         string                  `json:"name"`
	Description  *string                 `json:"description,omitempty"`
	PubTime      *time.Time              `json:"pub_time"`
	Tasks        *TasksBulkUpdateRequest `json:"tasks"`
	Sticky       *bool                   `json:"sticky,omitempty"`
	HasTimeLimit *bool                   `json:"has_time_limit,omitempty"`
	TimeLimit    *Duration               `json:"time_limit,omitempty"`
}

type DeleteTaskGroupRequest struct {
	ID ID `json:"id"`
}

type TaskGroupsBulkUpdateRequest struct {
	QuestID ID                       `json:"-"`
	Create  []CreateTaskGroupRequest `json:"create"`
	Update  []UpdateTaskGroupRequest `json:"update"`
	Delete  []DeleteTaskGroupRequest `json:"delete"`
}

type CreateHintRequest struct {
	Name    *string      `json:"name,omitempty"`
	Text    string       `json:"text,omitempty"`
	Penalty PenaltyOneOf `json:"penalty"`
}

type CreateTaskRequest struct {
	OrderIdx       int                 `json:"order_idx"`
	GroupID        ID                  `json:"group_id"`
	Name           string              `json:"name"`
	Question       string              `json:"question"`
	Reward         int                 `json:"reward"`
	CorrectAnswers []string            `json:"correct_answers"`
	Verification   VerificationType    `json:"verification"`
	Hints          []string            `json:"hints"`
	FullHints      []CreateHintRequest `json:"hints_full"`
	PubTime        *time.Time          `json:"pub_time"`
	MediaLinks     []string            `json:"media_links,omitempty"`
	// Deprecated
	MediaLink string `json:"media_link" example:"deprecated"`
}

type GetTaskRequest struct {
	ID ID
}

type GetTasksRequest struct {
	GroupIDs []ID
	QuestID  ID
}

type GetTasksResponse map[ID][]Task

type UpdateTaskRequest struct {
	QuestID        ID                   `json:"-"`
	ID             ID                   `json:"id"`
	OrderIdx       int                  `json:"order_idx"`
	GroupID        ID                   `json:"group_id"`
	Name           string               `json:"name"`
	Question       string               `json:"question"`
	Reward         int                  `json:"reward"`
	CorrectAnswers []string             `json:"correct_answers"`
	Verification   VerificationType     `json:"verification"`
	Hints          []string             `json:"hints"`
	FullHints      *[]CreateHintRequest `json:"hints_full"`
	PubTime        *time.Time           `json:"pub_time"`
	MediaLinks     []string             `json:"media_links,omitempty"`
	// Deprecated
	MediaLink *string `json:"media_link" example:"deprecated"`
}

type DeleteTaskRequest struct {
	ID ID `json:"id"`
}

type TasksBulkUpdateRequest struct {
	QuestID ID                  `json:"-"`
	GroupID ID                  `json:"-"`
	Create  []CreateTaskRequest `json:"create"`
	Update  []UpdateTaskRequest `json:"update"`
	Delete  []DeleteTaskRequest `json:"delete"`
}

type GetHintTakesRequest struct {
	TeamID  ID
	QuestID ID
	TaskID  ID
}

type TakeHintRequest struct {
	TeamID ID
	TaskID ID
	Index  int
}

type GetAcceptedTasksRequest struct {
	TeamID  ID
	QuestID ID
}

type CreateAnswerTryRequest struct {
	TeamID   ID
	UserID   ID
	TaskID   ID
	Text     string
	Accepted bool
	Score    int
}

type GetResultsRequest struct {
	QuestID ID
	TeamIDs []ID
}

type GetPenaltiesRequest struct {
	QuestID ID
	TeamIDs []ID
}

type CreatePenaltyRequest struct {
	TeamID  ID
	Penalty int
}

type TaskRequestLogFilterOptions struct {
	TaskID       ID
	GroupID      ID
	TeamID       ID
	UserID       ID
	OnlyAccepted bool
	DateDesc     bool

	PageSize   int
	PageNumber *int
	PageToken  *int64
}

func NewDefaultLogOpts() TaskRequestLogFilterOptions {
	return TaskRequestLogFilterOptions{
		PageSize: 50,
	}
}

type FilteringOption func(*TaskRequestLogFilterOptions)

func WithTaskID(taskID ID) FilteringOption {
	return func(o *TaskRequestLogFilterOptions) {
		o.TaskID = taskID
	}
}

func WithGroupID(groupID ID) FilteringOption {
	return func(o *TaskRequestLogFilterOptions) {
		o.GroupID = groupID
	}
}

func WithTeamID(teamID ID) FilteringOption {
	return func(o *TaskRequestLogFilterOptions) {
		o.TeamID = teamID
	}
}

func WithUserID(userID ID) FilteringOption {
	return func(o *TaskRequestLogFilterOptions) {
		o.UserID = userID
	}
}

func WithOnlyAccepted() FilteringOption {
	return func(o *TaskRequestLogFilterOptions) {
		o.OnlyAccepted = true
	}
}

func WithDateDesc() FilteringOption {
	return func(o *TaskRequestLogFilterOptions) {
		o.DateDesc = true
	}
}

func WithPageSize(pageSize int) FilteringOption {
	return func(o *TaskRequestLogFilterOptions) {
		o.PageSize = pageSize
	}
}

func WithPageNumber(pageNumber int) FilteringOption {
	return func(o *TaskRequestLogFilterOptions) {
		o.PageNumber = &pageNumber
	}
}

func WithPageToken(pageToken int64) FilteringOption {
	return func(o *TaskRequestLogFilterOptions) {
		o.PageToken = &pageToken
	}
}

type GetAnswerTriesRequest struct {
	QuestID ID
}

type UpsertTeamInfoRequest struct {
	TeamID      ID
	TaskGroupID ID
	OpeningTime time.Time
	ClosingTime *time.Time
}

type GetTeamInfoRequest struct {
	TaskGroupID ID
	TeamData    TeamData
}

type GetTeamInfosRequest struct {
	QuestID      ID
	TaskGroupIDs []ID
	TeamData     TeamData
}

// GetTeamInfosResponse TaskGroup.ID -> *TaskGroupTeamInfo
type GetTeamInfosResponse map[ID]*TaskGroupTeamInfo

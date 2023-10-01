package storage

import "time"

type AccessType string

var (
	Public   AccessType = "public"
	LinkOnly AccessType = "link_only"
)

type VerificationType string

var (
	Auto   VerificationType = "auto"
	Manual VerificationType = "manual"
)

type Quest struct {
	Id                   string
	Name                 string
	Description          string
	Access               AccessType
	CreatorName          string
	Creator              *User
	RegistrationDeadline *time.Time
	StartTime            *time.Time
	FinishTime           *time.Time
	MediaLink            string
	MaxTeamCap           *int
}

type CreateQuestRequest Quest

type GetQuestRequest struct {
	Id string
}

type UpdateQuestRequest Quest

type DeleteQuestRequest GetQuestRequest

type Team struct {
	Id          string
	Name        string
	QuestId     string
	Quest       *Quest
	CapitanName string
	Capitan     *User
	Score       int
	InviteLink  string
}

type CreateTeamRequest Team

type User struct {
	Id         string
	Username   string
	Password   string
	FirstName  string
	SecondName string
	AvatarUrl  string
}

type CreateUserRequest User

type GetUserRequest struct {
	Id       string
	Username string
}

type UpdateUserRequest struct {
	Id        string
	Username  string
	Password  string
	AvatarUrl string
}

type DeleteUserRequest struct {
	Id       string
	Username string
}

type TaskGroup struct {
	Id      string
	QuestId string
	Quest   *Quest
	Name    string
	PubTime *time.Time
}

type Task struct {
	Id             string
	GroupId        string
	Group          *TaskGroup
	Name           string
	Question       string
	Reward         int
	CorrectAnswers []string
	Verification   VerificationType
	Hints          []string
	PubTime        *time.Time
	MediaUrl       string
}

type AnswerTry struct {
	TaskId string
	Task   *Task
	UserId string
	User   *User
	Answer string
}

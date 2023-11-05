package storage

import (
	"time"

	"golang.org/x/xerrors"
)

type AccessType string

var (
	ErrExists     = xerrors.New("already exists")
	ErrNotFound   = xerrors.New("not found")
	ErrValidation = xerrors.New("validation error")
)

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
	Creator              *User
	RegistrationDeadline *time.Time
	StartTime            *time.Time
	FinishTime           *time.Time
	MediaLink            string
	MaxTeamCap           *int
}

type Team struct {
	Id         string
	Name       string
	Quest      *Quest
	Capitan    *User
	Score      int
	InviteLink string
}

type User struct {
	Id        string `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type TaskGroup struct {
	Id      string
	Quest   *Quest
	Name    string
	PubTime *time.Time
}

type Task struct {
	Id             string
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
	Task   *Task
	User   *User
	Answer string
}

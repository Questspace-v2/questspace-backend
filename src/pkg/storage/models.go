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
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Description          string     `json:"description,omitempty"`
	Access               AccessType `json:"access"`
	Creator              *User      `json:"creator"`
	RegistrationDeadline *time.Time `json:"registration_deadline"`
	StartTime            *time.Time `json:"start_time"`
	FinishTime           *time.Time `json:"finish_time,omitempty"`
	MediaLink            string     `json:"media_link"`
	MaxTeamCap           *int       `json:"max_team_cap,omitempty"`
}

type Team struct {
	ID         string
	Name       string
	Quest      *Quest
	Capitan    *User
	Score      int
	InviteLink string
}

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type TaskGroup struct {
	ID      string
	Quest   *Quest
	Name    string
	PubTime *time.Time
}

type Task struct {
	ID             string
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

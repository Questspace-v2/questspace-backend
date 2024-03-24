package storage

import (
	"strconv"
	"strings"
	"time"

	"golang.org/x/xerrors"
)

type AccessType string

var (
	ErrExists          = xerrors.New("already exists")
	ErrNotFound        = xerrors.New("not found")
	ErrValidation      = xerrors.New("validation error")
	ErrTeamAlreadyFull = xerrors.New("team already has maximum amount of members")
)

const (
	AccessPublic   AccessType = "public"
	AccessLinkOnly AccessType = "link_only"
)

type VerificationType string

const (
	VerificationAuto   VerificationType = "auto"
	VerificationManual VerificationType = "manual"
)

type QuestStatus string

const (
	StatusUnspecified      = ""
	StatusOnRegistration   = "ON_REGISTRATION"
	StatusRegistrationDone = "REGISTRATION_DONE"
	StatusRunning          = "RUNNING"
	StatusWaitResults      = "WAIT_RESULTS"
	StatusFinished         = "FINISHED"
)

type Quest struct {
	ID                   string      `json:"id"`
	Name                 string      `json:"name"`
	Description          string      `json:"description,omitempty"`
	Access               AccessType  `json:"access"`
	Creator              *User       `json:"creator"`
	RegistrationDeadline *time.Time  `json:"registration_deadline"`
	StartTime            *time.Time  `json:"start_time"`
	FinishTime           *time.Time  `json:"finish_time,omitempty"`
	MediaLink            string      `json:"media_link"`
	MaxTeamCap           *int        `json:"max_team_cap,omitempty"`
	Status               QuestStatus `json:"status"`
}

type GetQuestType int

const (
	GetAll        GetQuestType = 0
	GetRegistered GetQuestType = 1
	GetOwned      GetQuestType = 2
)

type Page struct {
	Timestamp int64
	Finished  bool
}

func PageFromIDString(id string) (*Page, error) {
	if id == "" {
		return &Page{Finished: false, Timestamp: 0}, nil
	}
	done, timestamp := id[:1], id[1:]
	var doneFlag bool
	var tsInt int64

	switch done[0] {
	case 'f':
		doneFlag = false
	case 't':
		doneFlag = true
	default:
		return nil, xerrors.Errorf("invalid page id format: %w", ErrValidation)
	}

	tsInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, xerrors.Errorf("%w: %w", ErrValidation, err)
	}

	return &Page{Finished: doneFlag, Timestamp: tsInt}, nil
}

func (p *Page) ID() string {
	var b strings.Builder
	if p.Finished {
		_ = b.WriteByte('t')
	} else {
		_ = b.WriteByte('f')
	}

	_, _ = b.WriteString(strconv.FormatInt(p.Timestamp, 10))
	return b.String()
}

type Team struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Quest        *Quest  `json:"-"`
	Captain      *User   `json:"captain"`
	Score        int     `json:"score"`
	InviteLink   string  `json:"invite_link"`
	InviteLinkID int64   `json:"-"`
	Members      []*User `json:"members"`
}

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type TaskGroup struct {
	ID       string
	OrderIdx int
	Quest    *Quest
	Name     string
	PubTime  *time.Time
}

type Task struct {
	ID             string
	OrderIdx       int
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

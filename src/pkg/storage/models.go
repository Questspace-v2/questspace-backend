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
	RegistrationDeadline *time.Time  `json:"registration_deadline,omitempty" example:"2024-04-14T12:00:00+05:00"`
	StartTime            *time.Time  `json:"start_time" example:"2024-04-14T14:00:00+05:00"`
	FinishTime           *time.Time  `json:"finish_time,omitempty" example:"2024-04-21T14:00:00+05:00"`
	MediaLink            string      `json:"media_link"`
	MaxTeamCap           *int        `json:"max_team_cap,omitempty"`
	Status               QuestStatus `json:"status" enums:"ON_REGISTRATION,REGISTRATION_DONE,RUNNING,WAIT_RESULTS,FINISHED"`
}

type GetQuestType int

const (
	GetPublic     GetQuestType = 0
	GetAll        GetQuestType = 1
	GetRegistered GetQuestType = 2
	GetOwned      GetQuestType = 3
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
	ID           string `json:"id"`
	Name         string `json:"name"`
	Quest        *Quest `json:"-"`
	Captain      *User  `json:"captain,omitempty"`
	Score        int    `json:"score"`
	InviteLink   string `json:"invite_link"`
	InviteLinkID int64  `json:"-"`
	Members      []User `json:"members,omitempty"`
}

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type TaskGroup struct {
	ID       string     `json:"id"`
	OrderIdx int        `json:"order_idx"`
	Quest    *Quest     `json:"-"`
	Name     string     `json:"name"`
	PubTime  *time.Time `json:"pub_time,omitempty"`
	Tasks    []Task     `json:"tasks"`
}

type Task struct {
	ID             string           `json:"id"`
	OrderIdx       int              `json:"order_idx"`
	Group          *TaskGroup       `json:"-"`
	Name           string           `json:"name"`
	Question       string           `json:"question"`
	Reward         int              `json:"reward"`
	CorrectAnswers []string         `json:"correct_answers"`
	Verification   VerificationType `json:"verification_type" enums:"auto,manual"`
	Hints          []string         `json:"hints"`
	PubTime        *time.Time       `json:"pub_time,omitempty"`
	MediaLink      string           `json:"media_link"`
}

//TODO(svayp11): commit the rest of play-mode

type AnswerTry struct {
	Team       *Team
	TaskID     string
	Answer     string
	AnswerTime *time.Time
}

type Hint struct {
	Index int    `json:"index"`
	Text  string `json:"text,omitempty"`
}

type HintTake struct {
	TaskID string
	Hint   Hint
}

type HintTakes map[string][]HintTake

type AcceptedTask struct {
	Text  string
	Score int
}

type AcceptedTasks map[string]AcceptedTask

type SingleTaskResult struct {
	TeamID    string
	TeamName  string
	GroupID   string
	GroupName string
	TaskID    string
	TaskName  string
	Score     int
	ScoreTime *time.Time
}

// ScoreResults [team_id] -> [task_id] -> Result
type ScoreResults map[string]map[string]SingleTaskResult

type Penalty struct {
	TeamID string
	Value  int
}

// TeamPenalties [team_id] -> []Penalty
type TeamPenalties map[string][]Penalty

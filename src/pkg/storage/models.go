package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"golang.org/x/xerrors"
)

type ID string

func NewID() ID {
	return ID(uuid.Must(uuid.NewV4()).String())
}

func (id ID) String() string {
	return string(id)
}

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

type QuestStatus int

const (
	StatusUnspecified QuestStatus = iota
	StatusOnRegistration
	StatusRegistrationDone
	StatusRunning
	StatusWaitResults
	StatusFinished
)

var (
	questStatusToStr = map[QuestStatus]string{
		StatusUnspecified:      "",
		StatusOnRegistration:   "ON_REGISTRATION",
		StatusRegistrationDone: "REGISTRATION_DONE",
		StatusRunning:          "RUNNING",
		StatusWaitResults:      "WAIT_RESULTS",
		StatusFinished:         "FINISHED",
	}
	strToQuestStatus = map[string]QuestStatus{
		"":                  StatusUnspecified,
		"ON_REGISTRATION":   StatusOnRegistration,
		"REGISTRATION_DONE": StatusRegistrationDone,
		"RUNNING":           StatusRunning,
		"WAIT_RESULTS":      StatusWaitResults,
		"FINISHED":          StatusFinished,
	}
)

func (q QuestStatus) String() string {
	return questStatusToStr[q]
}

func (q *QuestStatus) UnmarshalJSON(data []byte) error {
	if len(data) <= 2 {
		*q = StatusUnspecified
		return nil
	}
	str := string(data[1 : len(data)-1])
	status, ok := strToQuestStatus[str]
	if !ok {
		return xerrors.Errorf("unknown status %q", str)
	}
	*q = status
	return nil
}

func (q QuestStatus) MarshalJSON() ([]byte, error) {
	qStr := questStatusToStr[q]
	var b bytes.Buffer
	b.Grow(len(qStr) + 2)
	if err := b.WriteByte('"'); err != nil {
		return nil, err
	}
	if _, err := b.WriteString(qStr); err != nil {
		return nil, err
	}
	if err := b.WriteByte('"'); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

type RegistrationType string

const (
	RegistrationUnspecified RegistrationType = ""
	RegistrationAuto        RegistrationType = "AUTO"
	RegistrationVerify      RegistrationType = "VERIFY"
)

type QuestType string

const (
	TypeDefault = ""
	TypeAssault = "ASSAULT"
	TypeLinear  = "LINEAR"
)

type Quest struct {
	ID                   ID               `json:"id"`
	Name                 string           `json:"name"`
	Description          string           `json:"description,omitempty"`
	Access               AccessType       `json:"access"`
	Creator              *User            `json:"creator"`
	RegistrationDeadline *time.Time       `json:"registration_deadline,omitempty" example:"2024-04-14T12:00:00+05:00"`
	StartTime            *time.Time       `json:"start_time" example:"2024-04-14T14:00:00+05:00"`
	FinishTime           *time.Time       `json:"finish_time,omitempty" example:"2024-04-21T14:00:00+05:00"`
	MediaLink            string           `json:"media_link"`
	MaxTeamCap           *int             `json:"max_team_cap,omitempty"`
	Status               QuestStatus      `json:"status"`
	HasBrief             bool             `json:"has_brief,omitempty"`
	Brief                string           `json:"brief,omitempty"`
	MaxTeamsAmount       *int             `json:"max_teams_amount,omitempty"`
	RegistrationType     RegistrationType `json:"registration_type,omitempty" enums:"AUTO,VERIFY"`
	QuestType            QuestType        `json:"quest_type,omitempty" enums:"ASSAULT,LINEAR"`
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

type RegistrationStatus string

const (
	RegistrationStatusUnspecified     RegistrationStatus = ""
	RegistrationStatusOnConsideration RegistrationStatus = "ON_CONSIDERATION"
	RegistrationStatusAccepted        RegistrationStatus = "ACCEPTED"
)

type Team struct {
	ID                 ID                 `json:"id"`
	Name               string             `json:"name"`
	Quest              *Quest             `json:"-"`
	Captain            *User              `json:"captain,omitempty"`
	Score              int                `json:"score"`
	InviteLink         string             `json:"invite_link,omitempty"`
	InviteLinkID       int64              `json:"-"`
	Members            []User             `json:"members,omitempty"`
	RegistrationStatus RegistrationStatus `json:"registration_status,omitempty" enums:"ON_CONSIDERATION,ACCEPTED"`
}

type User struct {
	ID        ID     `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type Duration time.Duration

func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(*d).String())
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(value)
		return nil
	case string:
		var err error
		dur, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(dur)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

type TaskGroup struct {
	ID           ID         `json:"id"`
	OrderIdx     int        `json:"order_idx"`
	Quest        *Quest     `json:"-"`
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	PubTime      *time.Time `json:"pub_time,omitempty"`
	Sticky       bool       `json:"sticky,omitempty"`
	Tasks        []Task     `json:"tasks"`
	HasTimeLimit bool       `json:"has_time_limit,omitempty"`
	TimeLimit    *Duration  `json:"time_limit,omitempty"`
}

type Task struct {
	ID             ID         `json:"id"`
	OrderIdx       int        `json:"order_idx"`
	Group          *TaskGroup `json:"-"`
	Name           string     `json:"name"`
	Question       string     `json:"question"`
	Reward         int        `json:"reward"`
	CorrectAnswers []string   `json:"correct_answers"`
	// Deprecated
	Verification    VerificationType `json:"verification_type" example:"deprecated"`
	VerificationNew VerificationType `json:"verification" enums:"auto,manual"`
	Hints           []string         `json:"hints"`
	FullHints       []Hint           `json:"hints_full"`
	PubTime         *time.Time       `json:"pub_time,omitempty"`
	MediaLinks      []string         `json:"media_links,omitempty"`
	// Deprecated
	MediaLink string `json:"media_link,omitempty" example:"deprecated"`
}

type AnswerTry struct {
	Team       *Team
	User       *User
	TaskID     ID
	Answer     string
	AnswerTime *time.Time
}

func init() {
	DefaultPenalty, _ = NewPercentagePenalty(20)
}

var DefaultPenalty PenaltyOneOf

type PenaltyOneOf struct {
	percent *int
	score   *int

	Percent_DO_NOT_USE int `json:"percent"`
	Score_DO_NOT_USE   int `json:"score"`
}

func NewPercentagePenalty(percent int) (PenaltyOneOf, error) {
	if percent < 0 || percent > 100 {
		return PenaltyOneOf{}, xerrors.Errorf("percentage should be in bounds [0; 100], but got %d", percent)
	}
	return PenaltyOneOf{percent: &percent}, nil
}

func NewScorePenalty(score int) PenaltyOneOf {
	return PenaltyOneOf{score: &score}
}

func (p *PenaltyOneOf) GetPenaltyPoints(score int) int {
	if p.percent != nil {
		return score * *p.percent / 100
	}
	return *p.score
}

func (p *PenaltyOneOf) IsPercent() bool {
	return p.percent != nil
}

func (p *PenaltyOneOf) IsScore() bool {
	return p.score != nil
}

func (p *PenaltyOneOf) Empty() bool {
	return p.score == nil && p.percent == nil
}

func (p *PenaltyOneOf) Percent() int {
	if !p.IsPercent() {
		panic("penalty has no percent")
	}
	return *p.percent
}

func (p *PenaltyOneOf) Score() int {
	if !p.IsScore() {
		panic("penalty has no score")
	}
	return *p.score
}

func (p *PenaltyOneOf) PercentOpt() *int {
	if !p.IsPercent() {
		return nil
	}

	cp := *p.percent
	return &cp
}

func (p *PenaltyOneOf) ScoreOpt() *int {
	if !p.IsScore() {
		return nil
	}
	cp := *p.score
	return &cp
}

func (p *PenaltyOneOf) UnmarshalJSON(data []byte) error {
	var percent struct {
		Percent *int `json:"percent"`
	}
	var score struct {
		Score *int `json:"score"`
	}
	if err := json.Unmarshal(data, &percent); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &score); err != nil {
		return err
	}

	if percent.Percent != nil && score.Score != nil {
		return xerrors.New("penalty should have only one of types [score,percent]")
	}
	if percent.Percent == nil && score.Score == nil {
		return xerrors.New("penalty should have one of types [score,percent]")
	}

	if percent.Percent != nil {
		p.percent = percent.Percent
		return nil
	}

	p.score = score.Score
	return nil
}

func (p *PenaltyOneOf) MarshalJSON() ([]byte, error) {
	penalty := struct {
		Percent *int `json:"percent,omitempty"`
		Score   *int `json:"score,omitempty"`
	}{Percent: p.percent, Score: p.score}

	return json.Marshal(penalty)
}

type Hint struct {
	TaskID  ID           `json:"-"`
	Index   int          `json:"index"`
	Name    *string      `json:"name,omitempty"`
	Text    string       `json:"text,omitempty"`
	Penalty PenaltyOneOf `json:"penalty"`
}

type HintTake struct {
	TaskID ID
	Hint   Hint
}

// HintTakes [task_id] -> []HintTake
type HintTakes map[ID][]HintTake

type AcceptedTask struct {
	Text  string
	Score int
}

type AcceptedTasks map[ID]AcceptedTask

type SingleTaskResult struct {
	TeamID    ID
	TeamName  string
	GroupID   ID
	GroupName string
	TaskID    ID
	TaskName  string
	Score     int
	ScoreTime *time.Time
}

// ScoreResults [team_id] -> [task_id] -> Result
type ScoreResults map[ID]map[ID]SingleTaskResult

type Penalty struct {
	TeamID ID
	Value  int
}

// TeamPenalties [team_id] -> []Penalty
type TeamPenalties map[ID][]Penalty

type AnswerLog struct {
	Team       *Team
	User       *User
	TaskGroup  *TaskGroup
	Task       *Task
	Accepted   bool
	Answer     string
	AnswerTime time.Time
}

type AnswerLogRecords struct {
	AnswerLogs []AnswerLog
	NextToken  int64
	TotalPages int
}

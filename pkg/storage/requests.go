package storage

import "time"

type CreateUserRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type GetUserRequest struct {
	Id       string
	Username string
}

type UpdateUserRequest struct {
	Id        string `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type CreateQuestRequest struct {
	Name                 string     `json:"name"`
	Description          string     `json:"description,omitempty"`
	Access               AccessType `json:"access"`
	CreatorName          string     `json:"creator_name"`
	Creator              *User      `json:"-"`
	RegistrationDeadline *time.Time `json:"registration_deadline"`
	StartTime            *time.Time `json:"start_time"`
	FinishTime           *time.Time `json:"finish_time"`
	MediaLink            string     `json:"media_link"`
	MaxTeamCap           *int       `json:"max_team_cap"`
}

type GetQuestRequest struct {
	Id string
}

type UpdateQuestRequest struct {
	Id                   string     `json:"id"`
	Name                 string     `json:"name"`
	Description          string     `json:"description,omitempty"`
	Access               AccessType `json:"access"`
	CreatorName          string     `json:"creator_name"`
	Creator              *User      `json:"-"`
	RegistrationDeadline *time.Time `json:"registration_deadline"`
	StartTime            *time.Time `json:"start_time"`
	FinishTime           *time.Time `json:"finish_time"`
	MediaLink            string     `json:"media_link"`
	MaxTeamCap           *int       `json:"max_team_cap"`
}

type DeleteQuestRequest struct {
	Id string
}

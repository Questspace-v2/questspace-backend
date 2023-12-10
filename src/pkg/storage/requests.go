package storage

import "time"

type CreateUserRequest struct {
	Username  string `json:"username" example:"svayp11"`
	Password  string `json:"password,omitempty" example:"12345"`
	AvatarURL string `json:"avatar_url,omitempty" example:"https://api.dicebear.com/7.x/thumbs/svg"`
}

type GetUserRequest struct {
	Id       string
	Username string
}

type UpdateUserRequest struct {
	Id          string `json:"-"`
	Username    string `json:"username" example:"svayp11"`
	OldPassword string `json:"old_password" example:"12345"`
	NewPassword string `json:"new_password,omitempty" example:"complex_password_here"`
	AvatarURL   string `json:"avatar_url,omitempty" example:"https://i.pinimg.com/originals/7a/62/cb/7a62cb80e20da2d68a37b8db26833dc0.jpg"`
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
	Id                   string     `json:"-"`
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

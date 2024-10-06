package usertypes

import "questspace/pkg/storage"

type User struct {
	ID        storage.ID `json:"id" example:"0f1151b0-a81f-4bea-80e7-82deae0a5528"`
	Username  string     `json:"username" example:"svayp11"`
	AvatarURL string     `json:"avatar_url,omitempty" example:"https://api.dicebear.com/7.x/thumbs/svg"`
}

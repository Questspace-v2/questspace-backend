package requests

import (
	"time"

	"questspace/pkg/storage"
)

type CreateTaskRequest struct {
	Name           string                   `json:"name"`
	Question       string                   `json:"question"`
	Reward         int                      `json:"reward"`
	CorrectAnswers []string                 `json:"correct_answers"`
	Verification   storage.VerificationType `json:"verification" enums:"auto,manual"`
	Hints          []string                 `json:"hints" maxLength:"3"`
	PubTime        *time.Time               `json:"pub_time,omitempty"`
	MediaLinks     []string                 `json:"media_links,omitempty"`
	// Deprecated
	MediaLink string `json:"media_link" example:"deprecated"`
}

type CreateRequest struct {
	QuestID string              `json:"-"`
	Name    string              `json:"name"`
	PubTime *time.Time          `json:"pub_time,omitempty"`
	Tasks   []CreateTaskRequest `json:"tasks"`
}

type CreateFullRequest struct {
	QuestID    string          `json:"-"`
	TaskGroups []CreateRequest `json:"task_groups"`
}

type CreateFullResponse struct {
	TaskGroups []storage.TaskGroup `json:"task_groups"`
}

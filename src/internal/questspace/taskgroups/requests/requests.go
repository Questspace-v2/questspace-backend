package requests

import (
	"context"
	"time"

	"questspace/pkg/storage"
)

type ImageValidator interface {
	ValidateImageURLs(context.Context, ...string) error
}

type NopValidator struct{}

func (n NopValidator) ValidateImageURLs(ctx context.Context, urls ...string) error {
	return nil
}

type CreateTaskRequest struct {
	Name           string                      `json:"name"`
	Question       string                      `json:"question"`
	Reward         int                         `json:"reward"`
	CorrectAnswers []string                    `json:"correct_answers"`
	Verification   storage.VerificationType    `json:"verification" enums:"auto,manual"`
	Hints          []string                    `json:"hints" maxLength:"3"`
	FullHints      []storage.CreateHintRequest `json:"hints_full" maxLength:"3"`
	PubTime        *time.Time                  `json:"pub_time,omitempty"`
	MediaLinks     []string                    `json:"media_links,omitempty"`
	// Deprecated
	MediaLink string `json:"media_link" example:"deprecated"`
}

type CreateRequest struct {
	QuestID      storage.ID          `json:"-"`
	Name         string              `json:"name"`
	Description  string              `json:"description"`
	PubTime      *time.Time          `json:"pub_time,omitempty"`
	Sticky       bool                `json:"sticky,omitempty"`
	Tasks        []CreateTaskRequest `json:"tasks"`
	HasTimeLimit bool                `json:"has_time_limit,omitempty"`
	TimeLimit    *storage.Duration   `json:"time_limit,omitempty"`
}

type CreateFullRequest struct {
	QuestID    storage.ID      `json:"-"`
	TaskGroups []CreateRequest `json:"task_groups"`
}

type CreateFullResponse struct {
	TaskGroups []storage.TaskGroup `json:"task_groups"`
}

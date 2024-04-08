package quests

import (
	"context"
	"net/http"

	"golang.org/x/xerrors"

	"questspace/pkg/application/httperrors"
	"questspace/pkg/storage"
)

const (
	allFieldName        = "all"
	registeredFieldName = "registered"
	ownedFieldName      = "owned"
)

var existingFieldsMap = map[string]struct{}{
	allFieldName:        {},
	registeredFieldName: {},
	ownedFieldName:      {},
}

type PaginatedQuestsResponse struct {
	Quests     []storage.Quest `json:"quests"`
	NextPageID string          `json:"next_page_id,omitempty"`
}

type Quests struct {
	All        *PaginatedQuestsResponse `json:"all,omitempty"`
	Registered *PaginatedQuestsResponse `json:"registered,omitempty"`
	Owned      *PaginatedQuestsResponse `json:"owned,omitempty"`
}

func getQuestsOfType(ctx context.Context, s storage.QuestStorage, user *storage.User, page *storage.Page, getType storage.GetQuestType, pageSize int) (*PaginatedQuestsResponse, error) {
	req := &storage.GetQuestsRequest{
		User:     user,
		Type:     getType,
		PageSize: pageSize,
		Page:     page,
	}
	allQuests, err := s.GetQuests(ctx, req)
	if err != nil {
		return nil, xerrors.Errorf("get quests: %w", err)
	}
	quests := &PaginatedQuestsResponse{Quests: allQuests.Quests}
	if allQuests.NextPage != nil {
		quests.NextPageID = allQuests.NextPage.ID()
	}
	for _, q := range quests.Quests {
		SetStatus(&q)
	}
	return quests, nil
}

func ReadQuests(ctx context.Context, s storage.QuestStorage, user *storage.User, fields []string, pageID string, pageSize int) (*Quests, error) {
	allowedFields := make([]string, 0, len(existingFieldsMap))
	for _, f := range fields {
		if _, ok := existingFieldsMap[f]; ok {
			allowedFields = append(allowedFields, f)
		}
	}
	if len(allowedFields) > 1 && pageID != "" {
		return nil, httperrors.New(http.StatusBadRequest, "cannot set page id for request with more than one field")
	}
	var page *storage.Page
	if pageID != "" {
		var err error
		page, err = storage.PageFromIDString(pageID)
		if err != nil {
			return nil, httperrors.Errorf(http.StatusBadRequest, "parse page id: %w", err)
		}
	}

	quests := &Quests{}
	for _, field := range allowedFields {
		var err error

		switch field {
		case allFieldName:
			quests.All, err = getQuestsOfType(ctx, s, user, page, storage.GetAll, pageSize)
			if err != nil {
				return nil, xerrors.Errorf("get all: %w", err)
			}
		case registeredFieldName:
			quests.Registered, err = getQuestsOfType(ctx, s, user, page, storage.GetRegistered, pageSize)
			if err != nil {
				return nil, xerrors.Errorf("get registered: %w", err)
			}
		case ownedFieldName:
			quests.Owned, err = getQuestsOfType(ctx, s, user, page, storage.GetOwned, pageSize)
			if err != nil {
				return nil, xerrors.Errorf("get owned: %w", err)
			}
		}
	}

	return quests, nil
}

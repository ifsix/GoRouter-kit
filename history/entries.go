package history

import (
	"context"

	"github.com/ifsix/GoRouter-kit/schema"
)

type ApiCallMetadata struct {
	CallID               string        `json:"call_id,omitempty"`
	ModelUsed            string        `json:"model_used,omitempty"`
	Usage                *schema.Usage `json:"usage,omitempty"`
	Cost                 *float64      `json:"cost,omitempty"`
	Timestamp            int64         `json:"timestamp,omitempty"`
	FinishReason         string        `json:"finish_reason,omitempty"`
	RequestMessagesCount int           `json:"request_messages_count,omitempty"`
}

type HistoryEntry struct {
	Message         schema.Message   `json:"message"`
	ApiCallMetadata *ApiCallMetadata `json:"api_call_metadata,omitempty"`
}

type EntryStore interface {
	LoadEntries(ctx context.Context, sessionID string) ([]HistoryEntry, error)
	SaveEntries(ctx context.Context, sessionID string, entries []HistoryEntry) error
	DeleteEntries(ctx context.Context, sessionID string) error
	ListEntryKeys(ctx context.Context) ([]string, error)
}

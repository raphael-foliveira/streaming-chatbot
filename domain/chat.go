package domain

import "context"

type ChatSession struct {
	ID   string `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type ChatMessage struct {
	ID               string  `json:"id" db:"id"`
	Role             string  `json:"role" db:"role"`
	Content          string  `json:"content" db:"content"`
	ChatSessionID    string  `json:"chat_session_id" db:"chat_session_id"`
	ReasoningSummary *string `json:"reasoning_summary" db:"reasoning_summary"`
	Name             *string `json:"name" db:"name"`
	Args             *string `json:"args" db:"args"`
	CallID           *string `json:"call_id" db:"call_id"`
	Result           *string `json:"result" db:"result"`
}

type MessageEnqueuer interface {
	EnqueueUserMessage(ctx context.Context, chatName, message string) error
}

type ChatRepository interface {
	GetMessages(ctx context.Context, chatName string) ([]ChatMessage, error)
	SaveMessage(ctx context.Context, chatName string, messages ...ChatMessage) error
	CreateChat(ctx context.Context, chatName string) error
}

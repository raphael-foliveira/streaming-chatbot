package domain

import (
	"context"
	"time"
)

type ChatSession struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type ChatMessage struct {
	ID               string    `json:"id" db:"id"`
	Role             string    `json:"role" db:"role"`
	Content          string    `json:"content" db:"content"`
	ChatSessionID    string    `json:"chat_session_id" db:"chat_session_id"`
	ReasoningSummary *string   `json:"reasoning_summary" db:"reasoning_summary"`
	Name             *string   `json:"name" db:"name"`
	Args             *string   `json:"args" db:"args"`
	CallID           *string   `json:"call_id" db:"call_id"`
	Result           *string   `json:"result" db:"result"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

type MessageEnqueuer interface {
	EnqueueUserMessage(ctx context.Context, chatName, message string) error
}

type ChatRepository interface {
	GetMessages(ctx context.Context, params GetMessagesParams) ([]ChatMessage, error)
	SaveMessage(ctx context.Context, sessionId string, messages ...ChatMessage) error
	CreateChat(ctx context.Context, sessionId string) (ChatSession, error)
	ListSessions(ctx context.Context) ([]ChatSession, error)
	DeleteSession(ctx context.Context, chatId string) error
}

type GetMessagesParams struct {
	ChatSessionId string
	Before        time.Time
	Limit         int
}

func (g *GetMessagesParams) ApplyDefaults() {
	if g.Before.IsZero() {
		g.Before = time.Now()
	}

	if g.Limit == 0 {
		g.Limit = 20
	}
}

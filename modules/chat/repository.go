package chat

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raphael-foliveira/htmbot/domain"
)

var _ domain.ChatRepository = &PGXRepository{}

type PGXRepository struct {
	pool *pgxpool.Pool
}

func NewPGXRepository(pool *pgxpool.Pool) *PGXRepository {
	return &PGXRepository{
		pool: pool,
	}
}

const createChatQuery = `
INSERT INTO chats (name) VALUES ($1)
`

func (p *PGXRepository) CreateChat(ctx context.Context, chatName string) error {
	_, err := p.pool.Exec(ctx, createChatQuery, chatName)
	return err
}

const getMessagesQuery = `
SELECT id, role, content, name, args, call_id, result, chat_session_id
FROM chat_messages
WHERE chat_session_id = $1
`

func (p *PGXRepository) GetMessages(ctx context.Context, chatSessionId string) ([]domain.ChatMessage, error) {
	rows, err := p.pool.Query(ctx, getMessagesQuery, chatSessionId)
	if err != nil {
		return nil, fmt.Errorf("failed to query chat messages: %w", err)
	}

	messages := []domain.ChatMessage{}

	for rows.Next() {
		var message domain.ChatMessage
		if err := rows.Scan(
			&message.ID,
			&message.Role,
			&message.Content,
			&message.Name,
			&message.Args,
			&message.CallID,
			&message.Result,
			&message.ChatSessionID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan chat message: %w", err)
		}
		messages = append(messages, message)
	}

	return messages, nil
}

func (p *PGXRepository) SaveMessage(ctx context.Context, chatSessionId string, messages ...domain.ChatMessage) error {
	rows := [][]any{}

	for _, message := range messages {
		rows = append(rows, []any{
			message.Role,
			message.Content,
			message.Name,
			message.Args,
			message.CallID,
			message.Result,
			chatSessionId,
		})
	}

	_, err := p.pool.CopyFrom(
		ctx,
		pgx.Identifier([]string{"chat_messages"}),
		[]string{"role", "content", "name", "args", "call_id", "result", "chat_session_id"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("failed to save chat messages: %w", err)
	}

	return nil
}

type InMemoryRepository struct {
	storage map[string][]domain.ChatMessage
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		storage: make(map[string][]domain.ChatMessage),
	}
}

func (r *InMemoryRepository) GetMessages(ctx context.Context, chatName string) ([]domain.ChatMessage, error) {
	messages, ok := r.storage[chatName]
	if !ok {
		if err := r.CreateChat(ctx, chatName); err != nil {
			return nil, fmt.Errorf("failed to create chat: %w", err)
		}
		return r.GetMessages(ctx, chatName)
	}
	return messages, nil
}

func (r *InMemoryRepository) SaveMessage(
	ctx context.Context,
	chatName string,
	messages ...domain.ChatMessage,
) error {
	r.storage[chatName] = append(r.storage[chatName], messages...)
	return nil
}

func (r *InMemoryRepository) CreateChat(ctx context.Context, chatName string) error {
	if _, exists := r.storage[chatName]; exists {
		return fmt.Errorf("chat %s already exists", chatName)
	}
	r.storage[chatName] = []domain.ChatMessage{}
	return nil
}

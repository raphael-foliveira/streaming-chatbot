package chat

import (
	"context"
	"fmt"

	"github.com/raphael-foliveira/htmbot/domain"
)

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

package chat

import (
	"context"
	"fmt"

	"github.com/raphael-foliveira/htmbot/domain"
)

var _ domain.ChatService = &Service{}

type Service struct {
	repository domain.ChatRepository
	pubsub     domain.PubSub[domain.ChatEvent]
	enqueuer   domain.MessageEnqueuer
}

func NewService(
	repository domain.ChatRepository,
	pubsub domain.PubSub[domain.ChatEvent],
	enqueuer domain.MessageEnqueuer,
) *Service {
	return &Service{
		repository: repository,
		pubsub:     pubsub,
		enqueuer:   enqueuer,
	}
}

func (s *Service) ListSessions(ctx context.Context) ([]domain.ChatSession, error) {
	return s.repository.ListSessions(ctx)
}

func (s *Service) CreateChat(ctx context.Context, name string) (domain.ChatSession, error) {
	return s.repository.CreateChat(ctx, name)
}

func (s *Service) GetChatPageData(ctx context.Context, chatId string) (domain.ChatPageData, error) {
	chatMessages, err := s.repository.GetMessages(ctx, domain.GetMessagesParams{
		ChatSessionId: chatId,
		Limit:         100,
	})
	if err != nil {
		return domain.ChatPageData{}, fmt.Errorf("failed to get messages: %w", err)
	}

	chatName, err := s.repository.GetSessionName(ctx, chatId)
	if err != nil {
		return domain.ChatPageData{}, fmt.Errorf("failed to get session name: %w", err)
	}

	return domain.ChatPageData{
		Name:     chatName,
		Messages: chatMessages,
	}, nil
}

func (s *Service) SendMessage(ctx context.Context, chatId, text string) error {
	newMessage := domain.ChatMessage{Role: "user", Content: text}

	if err := s.repository.SaveMessage(ctx, chatId, newMessage); err != nil {
		return fmt.Errorf("failed to save user message: %w", err)
	}

	if err := s.pubsub.Publish(chatId, domain.ChatEvent{
		Type:          "message",
		ChatSessionID: chatId,
		OfMessage:     newMessage,
	}); err != nil {
		return fmt.Errorf("failed to publish user message: %w", err)
	}

	if err := s.enqueuer.EnqueueUserMessage(ctx, chatId, text); err != nil {
		return fmt.Errorf("failed to enqueue user message: %w", err)
	}

	return nil
}

func (s *Service) DeleteChat(ctx context.Context, chatId string) error {
	return s.repository.DeleteSession(ctx, chatId)
}

func (s *Service) SubscribeToMessages(chatId string) (chan domain.ChatEvent, func(), error) {
	return s.pubsub.Subscribe(chatId)
}

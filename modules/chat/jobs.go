package chat

import (
	"context"

	"github.com/raphael-foliveira/htmbot/domain"
)

type MessageEnqueuer struct {
	ch chan ChatEvent
}

func NewMessageEnqueuer(ch chan ChatEvent) *MessageEnqueuer {
	return &MessageEnqueuer{
		ch: ch,
	}
}

func (e *MessageEnqueuer) EnqueueUserMessage(ctx context.Context, chatName, message string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case e.ch <- ChatEvent{
		ChatName: chatName,
		Message:  domain.ChatMessage{Role: "user", Content: message},
	}:
		return nil
	}
}

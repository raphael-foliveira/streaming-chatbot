package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/labstack/gommon/log"
	"github.com/raphael-foliveira/htmbot/domain"
)

type MessageProcessor struct {
	ch         chan ChatEvent
	publisher  domain.PubSub[ChatEvent]
	agent      domain.LLMAgent
	repository domain.ChatRepository
}

func NewMessageProcessor(
	ch chan ChatEvent,
	publisher domain.PubSub[ChatEvent],
	agent domain.LLMAgent,
	repository domain.ChatRepository,
) *MessageProcessor {
	return &MessageProcessor{
		ch:         ch,
		publisher:  publisher,
		agent:      agent,
		repository: repository,
	}
}

func (p *MessageProcessor) ProcessUserMessages(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case newMessage := <-p.ch:
			chatMessages, err := p.repository.GetMessages(ctx, newMessage.ChatName)
			if err != nil {
				log.Errorf("failed to get chat messages: %w", err)
				continue
			}

			deltaBuilder := strings.Builder{}

			response, err := p.agent.StreamResponse(
				ctx,
				append(chatMessages, newMessage.OfMessage),
				[]domain.LLMTool{&TestTool{}},
				func(delta string) {
					deltaBuilder.WriteString(delta)

					if err := p.publisher.Publish(newMessage.ChatName, ChatEvent{
						ChatName: newMessage.ChatName,
						Type:     "delta",
						OfDelta:  deltaBuilder.String(),
					}); err != nil {
						log.Errorf("failed to publish assistant message: %w", err)
					}
				},
			)
			if err != nil {
				return fmt.Errorf("failed to stream response: %w", err)
			}

			if err := p.repository.SaveMessage(ctx, newMessage.ChatName, response...); err != nil {
				return fmt.Errorf("failed to save assistant message: %w", err)
			}

		}
	}
}

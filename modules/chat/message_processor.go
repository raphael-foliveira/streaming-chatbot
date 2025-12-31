package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
	"github.com/raphael-foliveira/htmbot/domain"
)

type MessageProcessor struct {
	ch         chan domain.ChatEvent
	publisher  domain.PubSub[domain.ChatEvent]
	agent      domain.LLMAgent
	repository domain.ChatRepository
}

func NewMessageProcessor(
	ch chan domain.ChatEvent,
	publisher domain.PubSub[domain.ChatEvent],
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
			chatMessages, err := p.repository.GetMessages(ctx, domain.GetMessagesParams{
				ChatSessionId: newMessage.ChatSessionID,
				Limit:         30,
			})
			if err != nil {
				log.Errorf("failed to get chat messages: %w", err)
				continue
			}

			deltaId := uuid.New().String()

			builder := strings.Builder{}

			if err := p.publisher.Publish(newMessage.ChatSessionID, domain.ChatEvent{
				Type:          "delta_start",
				ChatSessionID: newMessage.ChatSessionID,
				OfDelta: domain.ChatDelta{
					ID: deltaId,
				},
			}); err != nil {
				log.Errorf("failed to publish delta_start event: %w", err)
			}

			response, err := p.agent.StreamResponse(
				ctx,
				append(chatMessages, newMessage.OfMessage),
				[]domain.LLMTool{NewTestTool()},
				func(delta string) {
					builder.WriteString(delta)
					if err := p.publisher.Publish(newMessage.ChatSessionID, domain.ChatEvent{
						Type:          "delta",
						ChatSessionID: newMessage.ChatSessionID,
						OfDelta: domain.ChatDelta{
							ID:   deltaId,
							Text: builder.String(),
						},
					}); err != nil {
						log.Errorf("failed to publish delta event: %w", err)
					}
				},
			)
			if err != nil {
				return fmt.Errorf("failed to stream response: %w", err)
			}

			if err := p.repository.SaveMessage(ctx, newMessage.ChatSessionID, response...); err != nil {
				return fmt.Errorf("failed to save assistant message: %w", err)
			}

		}
	}
}

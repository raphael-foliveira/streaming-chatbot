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

			deltaId := uuid.New().String()

			builder := strings.Builder{}

			isFirstDelta := true
			response, err := p.agent.StreamResponse(
				ctx,
				append(chatMessages, newMessage.OfMessage),
				[]domain.LLMTool{&TestTool{}},
				func(delta string) {
					if isFirstDelta {
						if err := p.publisher.Publish(newMessage.ChatName, ChatEvent{
							Type:     "delta_start",
							ChatName: newMessage.ChatName,
							OfDelta: ChatDelta{
								ID: deltaId,
							},
						}); err != nil {
							log.Errorf("failed to publish delta_start event: %w", err)
						}
						isFirstDelta = false
					}

					builder.WriteString(delta)
					if err := p.publisher.Publish(newMessage.ChatName, ChatEvent{
						Type:     "delta",
						ChatName: newMessage.ChatName,
						OfDelta: ChatDelta{
							ID:    deltaId,
							Delta: builder.String(),
						},
					}); err != nil {
						log.Errorf("failed to publish delta event: %w", err)
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

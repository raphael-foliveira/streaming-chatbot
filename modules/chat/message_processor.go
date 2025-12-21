package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/labstack/gommon/log"
	"github.com/raphael-foliveira/htmbot/domain"
)

type MessageProcessor struct {
	ch        chan ChatEvent
	publisher domain.PubSub[ChatEvent]
	agent     domain.LLMAgent
}

func NewMessageProcessor(
	ch chan ChatEvent,
	publisher domain.PubSub[ChatEvent],
	agent domain.LLMAgent,
) *MessageProcessor {
	return &MessageProcessor{
		ch:        ch,
		publisher: publisher,
		agent:     agent,
	}
}

var messagesDb = map[string][]domain.ChatMessage{}

func (p *MessageProcessor) ProcessUserMessages(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case newMessage := <-p.ch:
			chatMessages := messagesDb[newMessage.ChatName]

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
						OfDelta: fmt.Sprintf(
							`
								<div class="chat chat-start mr-auto" id="delta-container" hx-swap-oob="true">
									<div class="chat-bubble chat-bubble-secondary min-w-[100px] text-left">
										<span>%s</span>
									</div>
								</div>
							`,
							deltaBuilder.String(),
						),
					}); err != nil {
						log.Errorf("failed to publish assistant message: %w", err)
					}
				},
			)
			if err != nil {
				return fmt.Errorf("failed to stream response: %w", err)
			}

			messagesDb[newMessage.ChatName] = append(chatMessages, response...)

		}
	}
}

var _ domain.LLMTool = &TestTool{}

type TestTool struct{}

// Description implements domain.LLMTool.
func (t *TestTool) Description() string {
	return "Call this tool when prompted to test a tool"
}

// Execute implements domain.LLMTool.
func (t *TestTool) Execute(ctx context.Context, args string) (string, error) {
	var argsMap map[string]any
	if err := json.Unmarshal([]byte(args), &argsMap); err != nil {
		return "", err
	}
	name, ok := argsMap["name"].(string)
	if !ok {
		return "", fmt.Errorf("name is required")
	}
	return fmt.Sprintf("Tool executed successfully with name set to: %s", name), nil
}

// Name implements domain.LLMTool.
func (t *TestTool) Name() string {
	return "test-tool"
}

// Parameters implements domain.LLMTool.
func (t *TestTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "a random name",
			},
		},
	}
}

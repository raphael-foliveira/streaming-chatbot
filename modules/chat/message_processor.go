package chat

import (
	"context"
	"encoding/json"
	"fmt"

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
			chatMessages = append(chatMessages, newMessage.Message)
			messagesDb[newMessage.ChatName] = chatMessages

			response, err := p.agent.GenerateResponse(ctx, chatMessages, []domain.LLMTool{&TestTool{}})
			if err != nil {
				continue
			}

			messagesDb[newMessage.ChatName] = append(chatMessages, response...)

			for _, responseMessage := range response {
				if responseMessage.Role == "assistant" {
					assistantMessage := responseMessage.Content
					if assistantMessage != "" {
						p.publisher.Publish(newMessage.ChatName, ChatEvent{
							ChatName: newMessage.ChatName,
							Message:  domain.ChatMessage{Role: "assistant", Content: assistantMessage},
						})
					}
				}
			}
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

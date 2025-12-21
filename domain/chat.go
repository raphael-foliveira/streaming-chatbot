package domain

import "context"

type ChatMessage struct {
	ID               string
	Role             string
	Content          string
	OfFunctionCall   *ChatFunctionCallMessage
	OfFunctionResult *ChatFunctionResultMessage
	OfReasoning      *ChatReasoningMessage
}

type ChatReasoningMessageSummary struct {
	Text string `json:"text"`
}

type ChatReasoningMessage struct {
	Summary string `json:"summary"`
	Content string `json:"content"`
}

type ChatFunctionCallMessage struct {
	Name   string `json:"name"`
	Args   string `json:"args"`
	CallID string `json:"call_id"`
}

type ChatFunctionResultMessage struct {
	Name   string `json:"name"`
	Result string `json:"result"`
	ID     string `json:"id"`
	CallID string `json:"call_id"`
}

type MessagePublisher interface {
	PublishUserMessage(ctx context.Context, chatName, message string) error
	PublishAssistantMessage(ctx context.Context, chatName, message string) error
}

type MessageEnqueuer interface {
	EnqueueUserMessage(ctx context.Context, chatName, message string) error
}

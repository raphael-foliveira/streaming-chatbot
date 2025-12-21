package domain

import "context"

type ChatMessage struct {
	ID               string                     `json:"id"`
	Role             string                     `json:"role"`
	Content          string                     `json:"content"`
	OfFunctionCall   *ChatFunctionCallMessage   `json:"of_function_call"`
	OfFunctionResult *ChatFunctionResultMessage `json:"of_function_result"`
	OfReasoning      *ChatReasoningMessage      `json:"of_reasoning"`
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

type MessageEnqueuer interface {
	EnqueueUserMessage(ctx context.Context, chatName, message string) error
}

type ChatRepository interface {
	GetMessages(ctx context.Context, chatName string) ([]ChatMessage, error)
	SaveMessage(ctx context.Context, chatName string, messages ...ChatMessage) error
	CreateChat(ctx context.Context, chatName string) error
}

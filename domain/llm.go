package domain

import (
	"context"
)

type LLMAgent interface {
	GenerateResponse(
		ctx context.Context,
		messages []ChatMessage,
		tools []LLMTool,
	) ([]ChatMessage, error)

	StreamResponse(
		ctx context.Context,
		messages []ChatMessage,
		tools []LLMTool,
		callback func(delta string),
	) ([]ChatMessage, error)
}

type LLMTool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(context.Context, string) (string, error)
}

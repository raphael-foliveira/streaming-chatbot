package agents

import (
	"context"
	"encoding/json"
)

type LLMTool struct {
	name        string
	description string
	parameters  map[string]any
	execute     func(context.Context, string) (string, error)
}

func NewLLMTool[T any, K any](
	name, description string,
	parameters map[string]any,
	executeFn func(context.Context, T) (K, error),
) *LLMTool {
	return &LLMTool{
		name:        name,
		description: description,
		parameters:  parameters,
		execute: func(ctx context.Context, args string) (string, error) {
			var parsedArgs T
			if err := json.Unmarshal([]byte(args), &parsedArgs); err != nil {
				return "", err
			}
			result, err := executeFn(ctx, parsedArgs)
			if err != nil {
				return "", err
			}
			resultBytes, err := json.Marshal(result)
			if err != nil {
				return "", err
			}
			return string(resultBytes), nil
		},
	}
}

func (t *LLMTool) Name() string {
	return t.name
}

func (t *LLMTool) Description() string {
	return t.description
}

func (t *LLMTool) Parameters() map[string]any {
	return t.parameters
}

func (t *LLMTool) Execute(ctx context.Context, args string) (string, error) {
	return t.execute(ctx, args)
}

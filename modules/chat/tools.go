package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/raphael-foliveira/htmbot/domain"
)

var _ domain.LLMTool = &TestTool{}

type TestTool struct{}

func (t *TestTool) Description() string {
	return "Call this tool when prompted to test a tool"
}

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

func (t *TestTool) Name() string {
	return "test-tool"
}

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

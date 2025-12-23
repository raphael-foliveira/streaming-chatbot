package chat

import (
	"context"
	"fmt"

	"github.com/raphael-foliveira/htmbot/platform/agents"
)

func NewTestTool() *agents.LLMTool {
	return agents.NewLLMTool(
		"test-tool",
		"Call this tool when prompted to test a tool",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "a random name",
				},
			},
		},
		func(ctx context.Context, args map[string]any) (string, error) {
			name, ok := args["name"].(string)
			if !ok {
				return "", fmt.Errorf("name is required")
			}
			return fmt.Sprintf("Tool executed successfully with name set to: %s", name), nil
		},
	)
}

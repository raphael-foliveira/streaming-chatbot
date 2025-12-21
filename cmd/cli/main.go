package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/raphael-foliveira/htmbot/domain"
	"github.com/raphael-foliveira/htmbot/modules/agents"
	"github.com/raphael-foliveira/htmbot/modules/chat"
)

func main() {
	agent := agents.NewOpenAI(os.Getenv("OPENAI_API_KEY"))

	responses, err := agent.StreamResponse(context.Background(), []domain.ChatMessage{
		{Role: "user", Content: "Can you call the available tool and tell me how it went?"},
	}, []domain.LLMTool{&chat.TestTool{}}, func(delta string) {
		fmt.Print(delta)
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println()
	responsesJson, _ := json.MarshalIndent(responses, "", "  ")
	log.Println(string(responsesJson))
}

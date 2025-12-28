package main

import (
	"context"
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/raphael-foliveira/htmbot/assets"
	"github.com/raphael-foliveira/htmbot/modules/chat"
	"github.com/raphael-foliveira/htmbot/platform/agents"
	"github.com/raphael-foliveira/htmbot/platform/pubsub"
)

func main() {
	e := echo.New()

	e.Use(middleware.RequestLogger())

	e.StaticFS("/assets", echo.MustSubFS(assets.Assets, ""))

	agent := agents.NewOpenAI(os.Getenv("OPENAI_API_KEY"))

	chatRepository := chat.NewInMemoryRepository()
	messagesChannel := make(chan chat.ChatEvent, 1000)
	enqueuer := chat.NewMessageEnqueuer(messagesChannel)
	publisher := pubsub.NewChannel(map[string][]chan chat.ChatEvent{})

	chatHandler := chat.NewHandler(enqueuer, publisher, chatRepository)
	chatHandler.Register(e)

	messagesProcessor := chat.NewMessageProcessor(
		messagesChannel,
		publisher,
		agent,
		chatRepository,
	)
	go messagesProcessor.ProcessUserMessages(context.Background())

	log.Fatal(e.Start(":8080"))
}

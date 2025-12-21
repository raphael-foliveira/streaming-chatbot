package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/raphael-foliveira/htmbot"
	"github.com/raphael-foliveira/htmbot/modules/agents"
	"github.com/raphael-foliveira/htmbot/modules/chat"
	"github.com/raphael-foliveira/htmbot/platform/pubsub"
)

func main() {
	fileHandler := http.FileServer(http.FS(htmbot.Assets))

	e := echo.New()

	e.Use(middleware.LoggerWithConfig(middleware.DefaultLoggerConfig))

	e.GET("/assets/*", echo.WrapHandler(fileHandler))

	agent := agents.NewOpenAI(os.Getenv("OPENAI_API_KEY"))

	messagesChannel := make(chan chat.ChatEvent, 1000)
	enqueuer := chat.NewMessageEnqueuer(messagesChannel)
	publisher := pubsub.NewChannel(map[string][]chan chat.ChatEvent{})

	chatHandler := chat.NewHandler(enqueuer, publisher)
	chatHandler.Register(e)

	messagesProcessor := chat.NewMessageProcessor(
		messagesChannel,
		publisher,
		agent,
	)
	go messagesProcessor.ProcessUserMessages(context.Background())

	log.Fatal(e.Start(":8080"))
}

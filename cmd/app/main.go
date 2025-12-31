package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/raphael-foliveira/htmbot/assets"
	"github.com/raphael-foliveira/htmbot/domain"
	"github.com/raphael-foliveira/htmbot/modules/chat"
	"github.com/raphael-foliveira/htmbot/platform/agents"
	"github.com/raphael-foliveira/htmbot/platform/pubsub"
)

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("environment variable %s is required", key)
	}
	return value
}

func main() {
	e := echo.New()

	e.Use(middleware.RequestLogger())

	e.StaticFS("/assets", assets.Assets)

	apiKey := mustEnv("OPENAI_API_KEY")
	agent := agents.NewOpenAI(apiKey)

	dbConn, err := pgxpool.New(context.Background(), mustEnv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	chatRepository := chat.NewPGXRepository(dbConn)
	messagesChannel := make(chan domain.ChatEvent, 1000)
	enqueuer := chat.NewMessageEnqueuer(messagesChannel)
	publisher := pubsub.NewChannel(map[string][]chan domain.ChatEvent{})

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

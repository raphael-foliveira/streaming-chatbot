package chat

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/raphael-foliveira/htmbot/domain"
	"github.com/raphael-foliveira/htmbot/platform/httpx"
)

type Handler struct {
	agent    domain.LLMAgent
	enqueuer domain.MessageEnqueuer
	pubsub   domain.PubSub[ChatEvent]
}

func NewHandler(
	agent domain.LLMAgent,
	enqueuer domain.MessageEnqueuer,
	pubsub domain.PubSub[ChatEvent],
) *Handler {
	return &Handler{
		agent:    agent,
		enqueuer: enqueuer,
		pubsub:   pubsub,
	}
}

func (h *Handler) Register(e *echo.Echo) {
	g := e.Group("/chat")
	g.GET("", h.Index)
	g.POST("", h.Create)
	g.GET("/:chat-name", h.ChatPage)
	g.POST("/:chat-name/send-message", h.SendMessage)
	g.GET("/:chat-name/sse", h.ListenForMessages)
}

func (h *Handler) Index(c echo.Context) error {
	return httpx.Render(c, Index(nil))
}

func (h *Handler) Create(c echo.Context) error {
	name := c.FormValue("chat-name")
	if name == "" {
		return httpx.Render(c, Index(errors.New("chat name is required")))
	}
	return c.Redirect(http.StatusFound, fmt.Sprintf("/chat/%s", name))
}

func (h *Handler) ChatPage(c echo.Context) error {
	chatName := c.Param("chat-name")
	chatMessages, ok := messagesDb[chatName]
	if !ok {
		chatMessages = []domain.ChatMessage{}
	}

	return httpx.Render(c, ChatPage(chatName, chatMessages))
}

func (h *Handler) SendMessage(c echo.Context) error {
	chatName := c.Param("chat-name")

	text := c.FormValue("chat-input")
	if text == "" {
		return c.NoContent(http.StatusOK)
	}

	newMessage := domain.ChatMessage{Role: "user", Content: text}

	if err := h.pubsub.Publish(chatName, ChatEvent{ChatName: chatName, Message: newMessage}); err != nil {
		return fmt.Errorf("failed to publish user message: %w", err)
	}

	if err := h.enqueuer.EnqueueUserMessage(c.Request().Context(), chatName, text); err != nil {
		return fmt.Errorf("failed to enqueue user message: %w", err)
	}

	return httpx.Render(c, ChatContainer(chatName, append(messagesDb[chatName], newMessage)))
}

func (h *Handler) ListenForMessages(c echo.Context) error {
	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	chatName := c.Param("chat-name")
	messagesChannel, unsub, err := h.pubsub.Subscribe(chatName)
	if err != nil {
		return fmt.Errorf("failed to subscribe to chat: %w", err)
	}
	defer unsub()

	for {
		select {

		case <-c.Request().Context().Done():
			return c.Request().Context().Err()

		case message := <-messagesChannel:
			if message.ChatName != c.Param("chat-name") {
				continue
			}

			templBuffer := bytes.NewBuffer(nil)

			if err := Message(message.Message).Render(
				c.Request().Context(),
				templBuffer,
			); err != nil {
				return fmt.Errorf("failed to render message: %w", err)
			}

			if err := httpx.WriteEventStream(
				w,
				"chat-messages",
				templBuffer.String(),
			); err != nil {
				return err
			}

			w.Flush()
		}
	}
}

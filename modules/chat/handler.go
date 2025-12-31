package chat

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/raphael-foliveira/htmbot/domain"
	chatviews "github.com/raphael-foliveira/htmbot/modules/chat/views"
	"github.com/raphael-foliveira/htmbot/platform/httpx"
)

type Handler struct {
	enqueuer   domain.MessageEnqueuer
	pubsub     domain.PubSub[domain.ChatEvent]
	repository domain.ChatRepository
}

func NewHandler(
	enqueuer domain.MessageEnqueuer,
	pubsub domain.PubSub[domain.ChatEvent],
	repository domain.ChatRepository,
) *Handler {
	return &Handler{
		enqueuer:   enqueuer,
		pubsub:     pubsub,
		repository: repository,
	}
}

func (h *Handler) Register(e *echo.Echo) {
	g := e.Group("/chat")
	g.GET("", h.Index)
	g.POST("", h.Create)
	g.GET("/:chat-id", h.ChatPage)
	g.POST("/:chat-id/send-message", h.SendMessage)
	g.GET("/:chat-id/sse", h.ListenForMessages)
	g.DELETE("/:chat-id", h.DeleteChat)
}

func (h *Handler) Index(c echo.Context) error {
	chatSessions, err := h.repository.ListSessions(c.Request().Context())
	if err != nil {
		return fmt.Errorf("failed to list chat sessions: %w", err)
	}
	return httpx.Render(c, chatviews.Index(chatSessions, nil))
}

func (h *Handler) Create(c echo.Context) error {
	name := c.FormValue("chat-name")
	if name == "" {
		return httpx.HxRedirect(c, "/chat")
	}

	newSession, err := h.repository.CreateChat(c.Request().Context(), name)
	if err != nil {
		return httpx.HxRedirect(c, "/chat")
	}

	return httpx.Render(c, chatviews.ChatLink(newSession))
}

func (h *Handler) ChatPage(c echo.Context) error {
	chatId := c.Param("chat-id")
	chatMessages, err := h.repository.GetMessages(c.Request().Context(), domain.GetMessagesParams{
		ChatSessionId: chatId,
		Limit:         100,
	})
	if err != nil {
		return c.Redirect(http.StatusFound, "/chat")
	}
	chatName, err := h.repository.GetSessionName(c.Request().Context(), chatId)
	if err != nil {
		log.Println(err)
		return c.Redirect(http.StatusFound, "/chat")
	}

	return httpx.Render(c, chatviews.ChatPage(chatName, chatMessages))
}

func (h *Handler) SendMessage(c echo.Context) error {
	chatName := c.Param("chat-id")

	text := c.FormValue("chat-input")
	if text == "" {
		return c.NoContent(http.StatusOK)
	}

	newMessage := domain.ChatMessage{Role: "user", Content: text}

	if err := h.repository.SaveMessage(c.Request().Context(), chatName, newMessage); err != nil {
		return fmt.Errorf("failed to save user message: %w", err)
	}

	if err := h.pubsub.Publish(chatName, domain.ChatEvent{
		Type:          "message",
		ChatSessionID: chatName,
		OfMessage:     newMessage,
	}); err != nil {
		return fmt.Errorf("failed to publish user message: %w", err)
	}

	if err := h.enqueuer.EnqueueUserMessage(c.Request().Context(), chatName, text); err != nil {
		return fmt.Errorf("failed to enqueue user message: %w", err)
	}

	return httpx.Render(c, chatviews.ChatForm(chatName))
}

func (h *Handler) DeleteChat(c echo.Context) error {
	chatId := c.Param("chat-id")
	if err := h.repository.DeleteSession(c.Request().Context(), chatId); err != nil {
		return fmt.Errorf("failed to delete chat session: %w", err)
	}
	return httpx.HxRedirect(c, "/chat")
}

func (h *Handler) ListenForMessages(c echo.Context) error {
	httpx.SetupSSE(c)

	chatName := c.Param("chat-id")
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
			if err := httpx.WriteEventStreamTemplate(
				c,
				"chat-messages",
				chatviews.GetMessageTemplate(message),
			); err != nil {
				return err
			}

			c.Response().Flush()
		}
	}
}

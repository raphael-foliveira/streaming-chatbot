package chat

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/raphael-foliveira/htmbot/domain"
	chatviews "github.com/raphael-foliveira/htmbot/modules/chat/views"
	"github.com/raphael-foliveira/htmbot/platform/httpx"
)

type Handler struct {
	service domain.ChatService
}

func NewHandler(service domain.ChatService) *Handler {
	return &Handler{
		service: service,
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
	chatSessions, err := h.service.ListSessions(c.Request().Context())
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

	newSession, err := h.service.CreateChat(c.Request().Context(), name)
	if err != nil {
		return httpx.HxRedirect(c, "/chat")
	}

	return httpx.Render(c, chatviews.ChatLink(newSession))
}

func (h *Handler) ChatPage(c echo.Context) error {
	chatId := c.Param("chat-id")
	chatPageData, err := h.service.GetChatPageData(c.Request().Context(), chatId)
	if err != nil {
		return c.Redirect(http.StatusFound, "/chat")
	}

	return httpx.Render(c, chatviews.ChatPage(chatPageData.Name, chatPageData.Messages))
}

func (h *Handler) SendMessage(c echo.Context) error {
	chatName := c.Param("chat-id")

	text := c.FormValue("chat-input")
	if text == "" {
		return c.NoContent(http.StatusNoContent)
	}

	if err := h.service.SendMessage(c.Request().Context(), chatName, text); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return httpx.Render(c, chatviews.ChatForm(chatName))
}

func (h *Handler) DeleteChat(c echo.Context) error {
	chatId := c.Param("chat-id")
	if err := h.service.DeleteChat(c.Request().Context(), chatId); err != nil {
		return fmt.Errorf("failed to delete chat session: %w", err)
	}
	return httpx.HxRedirect(c, "/chat")
}

func (h *Handler) ListenForMessages(c echo.Context) error {
	httpx.SetupSSE(c)
	ctx := c.Request().Context()
	chatName := c.Param("chat-id")
	messagesChannel, unsub, err := h.service.SubscribeToMessages(chatName)
	if err != nil {
		return fmt.Errorf("failed to subscribe to chat: %w", err)
	}
	defer unsub()

	for {
		select {

		case <-ctx.Done():
			return ctx.Err()

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

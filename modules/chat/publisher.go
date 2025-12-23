package chat

import (
	"github.com/raphael-foliveira/htmbot/domain"
)

type ChatEvent struct {
	ChatName  string
	Type      string
	OfMessage domain.ChatMessage
	OfDelta   ChatDelta
}

func (c *ChatEvent) Delta() ChatDelta {
	return c.OfDelta
}

func (c *ChatEvent) Message() domain.ChatMessage {
	return c.OfMessage
}

type ChatDelta struct {
	ID   string
	Text string
}

package chat

import (
	"github.com/raphael-foliveira/htmbot/domain"
)

type ChatEvent struct {
	ChatName  string
	Type      string
	OfMessage domain.ChatMessage
	OfDelta   string
}

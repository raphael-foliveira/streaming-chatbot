package chat

import (
	"github.com/raphael-foliveira/htmbot/domain"
)

type ChatEvent struct {
	ChatName string
	Message  domain.ChatMessage
}

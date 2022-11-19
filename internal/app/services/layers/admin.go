package layers

import (
	"github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/BlackRRR/middleware-bot/internal/app/services/bot"
)

type AdminHandlers struct {
	Handlers   map[string]model.Handler
	BotService *bot.BotService
}

func (h *AdminHandlers) GetHandler(command string) model.Handler {
	return h.Handlers[command]
}

func (h *AdminHandlers) Init() {
	//admin messages
	h.OnCommand("/admin", h.BotService.Admin)
	//admin callback
	h.OnCommand("/chat_admin", h.BotService.SupMessages)
	h.OnCommand("/sup_messages", h.BotService.SendMessages)
}

func (h *AdminHandlers) OnCommand(command string, handler model.Handler) {
	h.Handlers[command] = handler
}

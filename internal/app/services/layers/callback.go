package layers

import (
	"github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/BlackRRR/middleware-bot/internal/app/services/bot"
)

type CallBackHandlers struct {
	Handlers   map[string]model.Handler
	BotService *bot.BotService
}

func (h *CallBackHandlers) GetHandler(command string) model.Handler {
	return h.Handlers[command]
}

func (h *CallBackHandlers) Init() {
	//Money command
	h.OnCommand("/call_admin", h.BotService.CallAdmin)
	h.OnCommand("/write_to_user", h.BotService.WriteToUser)
	h.OnCommand("/delete_chat", h.BotService.DeleteChat)
	h.OnCommand("/chat_number", h.BotService.SelectChat)
}

func (h *CallBackHandlers) OnCommand(command string, handler model.Handler) {
	h.Handlers[command] = handler
}
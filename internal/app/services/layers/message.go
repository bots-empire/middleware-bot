package layers

import (
	"github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/BlackRRR/middleware-bot/internal/app/services/bot"
)

type MessagesHandlers struct {
	Handlers   map[string]model.Handler
	BotService *bot.BotService
}

func (h *MessagesHandlers) GetHandler(command string) model.Handler {
	return h.Handlers[command]
}

func (h *MessagesHandlers) Init() {
	//Start command
	h.OnCommand("/start", h.BotService.StartCommand)
	h.OnCommand("/answer", h.BotService.AnswerToUser)
	h.OnCommand("/question_to_admin", h.BotService.QuestionAdmin)
	h.OnCommand("/live_chats", h.BotService.LiveChats)
	h.OnCommand("/delete_chats", h.BotService.EndingChat)
	h.OnCommand("/chat_info", h.BotService.InfoChatLists)
	h.OnCommand("/set_note_msg", h.BotService.SetNote)
}

func (h *MessagesHandlers) OnCommand(command string, handler model.Handler) {
	h.Handlers[command] = handler
}

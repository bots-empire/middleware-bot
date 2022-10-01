package model

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type GlobalHandlers interface {
	GetHandler(command string) Handler
}

type Handler func(situation *Situation) error

type Situation struct {
	Message       *tgbotapi.Message
	CallbackQuery *tgbotapi.CallbackQuery
	BotLang       string
	User          *User
	Command       string
	Params        *Parameters
	Err           error
}

type Parameters struct {
	Level string
}

package services

import (
	"github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/BlackRRR/middleware-bot/internal/app/repository"
	"github.com/BlackRRR/middleware-bot/internal/app/services/bot"
	"github.com/BlackRRR/middleware-bot/internal/app/services/layers"
	"github.com/BlackRRR/middleware-bot/internal/db/redis"
	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log2 "log"
)

type Services struct {
	BotSrv *bot.BotService
}

func InitServices(repo *repository.Repository, msgsSrv *msgs.Service, globalBot *model.GlobalBot) *Services {
	botSrv := bot.NewBotService(repo, msgsSrv, model.Bot)

	globalBot.MessageHandler = newMessagesHandler(botSrv)
	globalBot.CallbackHandler = newCallbackHandler(botSrv)

	startBot(globalBot)
	model.UploadUpdateStatistic()

	return &Services{BotSrv: botSrv}
}

func startBot(b *model.GlobalBot) {
	var err error
	b.Bot, err = tgbotapi.NewBotAPI(b.BotToken)
	if err != nil {
		log2.Fatalf("error start bot: %s", err.Error())
	}

	u := tgbotapi.NewUpdate(0)

	b.Chanel = b.Bot.GetUpdatesChan(u)

	b.Rdb = redis.StartRedis()

	b.ParseCommandsList()
	b.ParseLangMap()
}

func newMessagesHandler(botService *bot.BotService) *layers.MessagesHandlers {
	handle := layers.MessagesHandlers{
		Handlers:   map[string]model.Handler{},
		BotService: botService,
	}

	handle.Init()
	return &handle
}

func newCallbackHandler(botService *bot.BotService) *layers.CallBackHandlers {
	handle := layers.CallBackHandlers{
		Handlers:   map[string]model.Handler{},
		BotService: botService,
	}

	handle.Init()
	return &handle
}

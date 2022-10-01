package bot

import (
	"encoding/json"
	"fmt"
	model "github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/BlackRRR/middleware-bot/internal/app/repository"
	"github.com/BlackRRR/middleware-bot/internal/app/utils"
	"github.com/BlackRRR/middleware-bot/internal/db/redis"
	"github.com/BlackRRR/middleware-bot/internal/log"
	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"runtime/debug"
	"strings"
)

var (
	panicLogger = log.NewDefaultLogger().Prefix("panic cather")

	updatePrintHeader = "updates number: %d    // referral-bot-updates:  %s %s"
	extraneousUpdate  = "extraneous updates"
)

type BotService struct {
	Repo       *repository.Repository
	BaseBotSrv *msgs.Service
	GlobalBot  *model.GlobalBot
}

func NewBotService(repo *repository.Repository, msgsSrv *msgs.Service, bot *model.GlobalBot) *BotService {
	return &BotService{Repo: repo, BaseBotSrv: msgsSrv, GlobalBot: bot}
}

func (b *BotService) checkCallbackQuery(s *model.Situation, logger log.Logger) {
	Handler := b.GlobalBot.CallbackHandler.
		GetHandler(s.Command)

	if Handler != nil {
		if err := Handler(s); err != nil {
			logger.Warn("error with serve user callback command: %s", err.Error())
			b.smthWentWrong(s.CallbackQuery.Message.Chat.ID, b.GlobalBot.BotLang)
		}
		return
	}

	logger.Warn("get callback data='%s', but they didn't react in any way", s.CallbackQuery.Data)
}

func (b *BotService) ActionsWithUpdates(logger log.Logger, sortCentre *utils.Spreader) {
	for update := range b.GlobalBot.Chanel {
		localUpdate := update

		go b.checkUpdate(&localUpdate, logger, sortCentre)
	}
}

func (b *BotService) checkUpdate(update *tgbotapi.Update, logger log.Logger, sortCentre *utils.Spreader) {
	defer b.panicCather(update)

	if update.Message == nil && update.CallbackQuery == nil {
		return
	}

	if update.Message != nil && update.Message.PinnedMessage != nil {
		return
	}

	b.printNewUpdate(update, logger)
	if update.Message != nil && update.Message.From != nil {
		user, err := b.Repo.CheckingTheUser(update.Message)
		if err != nil {
			b.smthWentWrong(update.Message.Chat.ID, b.GlobalBot.BotLang)
			logger.Warn("err with check user: %s", err.Error())
			return
		}
		//
		//isAdmin := b.CheckAdmin(user.ID, logger)
		//
		//b.StartAnonymousChat(isAdmin, user, logger, update)

		situation := createSituationFromMsg(b.GlobalBot.BotLang, update.Message, user)

		b.checkMessage(situation, logger, sortCentre)
		return
	}

	if update.CallbackQuery != nil {
		situation, err := b.createSituationFromCallback(b.GlobalBot.BotLang, update.CallbackQuery)
		if err != nil {
			b.smthWentWrong(update.CallbackQuery.Message.Chat.ID, b.GlobalBot.BotLang)
			logger.Warn("err with create situation from callback: %s", err.Error())
			return
		}

		b.checkCallbackQuery(situation, logger)
		return
	}
}

func (b *BotService) StartAnonymousChat(isAdmin bool, user *model.User, logger log.Logger, update *tgbotapi.Update) {
	if isAdmin {
		id, chatStart, err := b.Repo.MessageToUser(user.ID)
		if err != nil {
			logger.Warn("err with get id and chat start to user: %s", err.Error())
		}

		if chatStart {
			err := b.BaseBotSrv.NewParseMessage(id, update.Message.Text)
			if err != nil {
				logger.Warn("err with parse message to user: %s", err.Error())
			}
		}
	} else {
		id, chatStart, err := b.Repo.MessageToAdmin(user.ID)
		if err != nil {
			logger.Warn("err with get id and chat start to admin: %s", err.Error())
		}

		if chatStart {
			err := b.BaseBotSrv.NewParseMessage(id, update.Message.Text)
			if err != nil {
				logger.Warn("err with parse message to user: %s", err.Error())
			}
		}
	}
}

func (b *BotService) CheckAdmin(id int64, logger log.Logger) bool {
	isAdmin, err := b.Repo.GetAdmin(id)
	if err != nil {
		logger.Warn("err with check admin: %s", err.Error())
	}

	return isAdmin
}

func (b *BotService) printNewUpdate(update *tgbotapi.Update, logger log.Logger) {
	model.UpdateStatistic.Mu.Lock()
	defer model.UpdateStatistic.Mu.Unlock()

	model.UpdateStatistic.Counter++
	model.SaveUpdateStatistic()

	model.HandleUpdates.WithLabelValues(
		b.GlobalBot.BotLink,
		b.GlobalBot.BotLang,
	).Inc()

	if update.Message != nil {
		if update.Message.Text != "" {
			logger.Info(updatePrintHeader,
				model.UpdateStatistic.Counter,
				b.GlobalBot.BotLang,
				update.Message.Text,
			)
			return
		}
	}

	if update.CallbackQuery != nil {
		logger.Info(updatePrintHeader,
			model.UpdateStatistic.Counter,
			b.GlobalBot.BotLang,
			update.CallbackQuery.Data,
		)
		return
	}

	logger.Info(updatePrintHeader,
		model.UpdateStatistic.Counter,
		b.GlobalBot.BotLang,
		extraneousUpdate,
	)
}

func createSituationFromMsg(botLang string, message *tgbotapi.Message, user *model.User) *model.Situation {
	return &model.Situation{
		Message: message,
		BotLang: botLang,
		User:    user,
		Params: &model.Parameters{
			Level: redis.GetLevel(user.ID),
		},
	}
}

func (b *BotService) createSituationFromCallback(botLang string, callbackQuery *tgbotapi.CallbackQuery) (*model.Situation, error) {
	user, err := b.Repo.GetUser(callbackQuery.From.ID)
	if err != nil {
		return nil, err
	}

	return &model.Situation{
		CallbackQuery: callbackQuery,
		BotLang:       botLang,
		User:          user,
		Command:       strings.Split(callbackQuery.Data, "?")[0],
		Params: &model.Parameters{
			Level: redis.GetLevel(callbackQuery.From.ID),
		},
	}, nil
}

func (b *BotService) checkMessage(situation *model.Situation, logger log.Logger, sortCentre *utils.Spreader) {
	if situation.Command == "" {
		situation.Command, situation.Err = b.GlobalBot.GetCommandFromText(
			situation.Message, b.GlobalBot.BotLang, situation.User.ID)
	}

	if situation.Err == nil {
		handler := model.Bot.MessageHandler.
			GetHandler(situation.Command)

		if handler != nil {
			sortCentre.ServeHandler(handler, situation, func(err error) {
				text := fmt.Sprintf("%s // %s // error with serve user msg command: %s",
					b.GlobalBot.BotLang,
					b.GlobalBot.BotLink,
					err.Error(),
				)
				b.BaseBotSrv.SendNotificationToDeveloper(text, false)

				logger.Warn(text)
				b.smthWentWrong(situation.Message.Chat.ID, b.GlobalBot.BotLang)
			})
			return
		}
	}

	situation.Command = strings.Split(situation.Params.Level, "?")[0]

	handler := model.Bot.MessageHandler.
		GetHandler(situation.Command)

	if handler != nil {
		sortCentre.ServeHandler(handler, situation, func(err error) {
			text := fmt.Sprintf("%s // %s // error with serve user level command: %s",
				b.GlobalBot.BotLang,
				b.GlobalBot.BotLink,
				err.Error(),
			)
			b.BaseBotSrv.SendNotificationToDeveloper(text, false)

			logger.Warn(text)
			b.smthWentWrong(situation.Message.Chat.ID, b.GlobalBot.BotLang)
		})
		return
	}

	b.smthWentWrong(situation.Message.Chat.ID, b.GlobalBot.BotLang)
	if situation.Err != nil {
		logger.Info(situation.Err.Error())
	}
}

func (b *BotService) smthWentWrong(chatID int64, lang string) {
	msg := tgbotapi.NewMessage(chatID, b.GlobalBot.LangText(lang, "user_level_not_defined"))
	_ = b.BaseBotSrv.SendMsgToUser(msg, chatID)
}

func (b *BotService) panicCather(update *tgbotapi.Update) {
	msg := recover()
	if msg == nil {
		return
	}

	panicText := fmt.Sprintf("%s // %s\npanic in backend: message = %s\n%s",
		b.GlobalBot.BotLang,
		b.GlobalBot.BotLink,
		msg,
		string(debug.Stack()),
	)
	panicLogger.Warn(panicText)

	b.BaseBotSrv.SendNotificationToDeveloper(panicText, false)

	data, err := json.MarshalIndent(update, "", "  ")
	if err != nil {
		return
	}

	b.BaseBotSrv.SendNotificationToDeveloper(string(data), false)
}

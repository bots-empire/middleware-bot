package bot

import (
	"github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/BlackRRR/middleware-bot/internal/db/redis"
	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
	"strings"
)

func (b *BotService) StartCommand(s *model.Situation) error {
	text := b.GlobalBot.LangText(s.BotLang, "call_admin")
	markUp := msgs.NewIlMarkUp(msgs.NewIlRow(msgs.NewIlDataButton("call", "/call_admin"))).Build(b.GlobalBot.Language[s.BotLang])

	return b.BaseBotSrv.NewParseMarkUpMessage(s.User.ID, &markUp, text)
}

func (b *BotService) StartChat(s *model.Situation) error {
	idStr := strings.Split(s.Message.Text, "?")[1]

	userId, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return err
	}

	text := b.GlobalBot.LangText(s.BotLang, "chat_started_user")
	err = b.BaseBotSrv.NewParseMessage(userId, text)
	if err != nil {
		return err
	}

	redis.RdbSetUser(userId, "/question_to_admin?"+strconv.FormatInt(s.Message.From.ID, 10))

	text = b.GlobalBot.LangText(s.BotLang, "chat_started_admin")
	markUp := msgs.NewIlMarkUp(msgs.NewIlRow(msgs.NewIlDataButton("answer_to_user", "/write_to_user?"+strconv.FormatInt(userId, 10)))).Build(b.GlobalBot.Language[s.BotLang])
	err = b.BaseBotSrv.NewParseMarkUpMessage(s.Message.From.ID, &markUp, text)
	if err != nil {
		return err
	}

	chatNum, started, err := b.Repo.GetChatConflict(s.Message.From.ID, userId)
	if err != nil {
		return err
	}

	if chatNum != 0 && started {

	}

	err = b.Repo.ChangeStartChat(s.Message.From.ID, userId, true)
	if err != nil {
		return err
	}

	return nil
}

func (b *BotService) AnswerToUser(s *model.Situation) error {
	level := redis.GetLevel(s.User.ID)
	userID := strings.Split(level, "?")[1]
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return err
	}
	redis.RdbSetUser(s.User.ID, "/answer?"+userID)

	err = b.Repo.AddMessageFromAdmin(s.Message.Text, id, s.Message.From.ID)
	if err != nil {
		return err
	}

	return b.BaseBotSrv.NewParseMessage(id, s.Message.Text)
}

func (b *BotService) QuestionAdmin(s *model.Situation) error {
	level := redis.GetLevel(s.Message.From.ID)
	adminID := strings.Split(level, "?")[1]
	id, err := strconv.ParseInt(adminID, 10, 64)
	if err != nil {
		return err
	}
	redis.RdbSetUser(s.User.ID, "/question_to_admin?"+adminID)

	err = b.Repo.AddMessageFromUser(s.Message.Text, s.Message.From.ID, id)
	if err != nil {
		return err
	}

	return b.BaseBotSrv.NewParseMessage(id, s.Message.Text)
}

func (b *BotService) LiveChats(s *model.Situation) error {
	admin, err := b.Repo.GetAdmin(s.User.ID)
	if err != nil {
		return err
	}

	if admin {
		chats, err := b.Repo.GetChatNumberWhereLiveChat(s.User.ID, true)
		if err != nil {
			return err
		}

		var buttons []tgbotapi.InlineKeyboardButton
		for _, val := range chats {

			data := "/chat_number?" + strconv.Itoa(val)
			button := tgbotapi.InlineKeyboardButton{
				Text:         b.GlobalBot.LangText(s.BotLang, "chat_number", val),
				CallbackData: &data,
			}

			buttons = append(buttons, button)
		}

		var markUp tgbotapi.InlineKeyboardMarkup
		markUp.InlineKeyboard = append(markUp.InlineKeyboard, buttons)

		text := b.GlobalBot.LangText(s.BotLang, "live_chats")

		err = b.BaseBotSrv.NewParseMarkUpMessage(s.User.ID, &markUp, text)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *BotService) EndingChat(s *model.Situation) error {
	chats, err := b.Repo.GetChatNumberWhereLiveChat(s.User.ID, true)
	if err != nil {
		return err
	}

	var buttons []tgbotapi.InlineKeyboardButton
	for _, val := range chats {
		data := "/chat_delete?" + strconv.Itoa(val)
		button := tgbotapi.InlineKeyboardButton{
			Text:         b.GlobalBot.LangText(s.BotLang, "chat_number", val),
			CallbackData: &data,
		}

		buttons = append(buttons, button)
	}

	var markUp tgbotapi.InlineKeyboardMarkup
	markUp.InlineKeyboard = append(markUp.InlineKeyboard, buttons)

	text := b.GlobalBot.LangText(s.BotLang, "delete_chat_num")

	err = b.BaseBotSrv.NewParseMarkUpMessage(s.User.ID, &markUp, text)
	if err != nil {
		return err
	}

	return nil
}

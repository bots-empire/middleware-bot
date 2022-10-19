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
	adminIDs, err := b.getAdmins(s.BotLang)
	if err != nil {
		return err
	}

	for _, id := range adminIDs {
		if s.User.ID == id {
			text := b.GlobalBot.LangText(s.BotLang, "you_are_admin")
			return b.BaseBotSrv.NewParseMessage(s.User.ID, text)
		}
	}

	text := b.GlobalBot.LangText(s.BotLang, "call_admin")
	markUp := msgs.NewIlMarkUp(msgs.NewIlRow(msgs.NewIlDataButton("call", "/call_admin"))).Build(b.GlobalBot.Language[s.BotLang])
	return b.BaseBotSrv.NewParseMarkUpMessage(s.User.ID, &markUp, text)
}

func (b *BotService) AnswerToUser(s *model.Situation) error {
	level := redis.GetLevel(s.User.ID)
	userID := strings.Split(level, "?")[1]
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return err
	}
	redis.RdbSetUser(s.User.ID, "/answer?"+userID)

	number, err := b.Repo.GetChatNumber(id, s.User.ID)
	if err != nil {
		return err
	}

	if s.Message.Photo != nil {
		photoFileId := s.Message.Photo[0].FileID

		err := b.Repo.AddPhotoMessageFromAdmin(photoFileId, s.Message.Caption, s.User.ID, number)
		if err != nil {
			return err
		}

		return b.BaseBotSrv.NewParseMarkUpPhotoMessage(id, nil, s.Message.Caption, tgbotapi.FileID(photoFileId))
	}

	err = b.Repo.AddMessageFromAdmin(s.Message.Text, s.User.ID, number)
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

	number, err := b.Repo.GetChatNumber(s.User.ID, id)
	if err != nil {
		return err
	}

	if s.Message.Photo != nil {
		photoFileId := s.Message.Photo[0].FileID

		err := b.Repo.AddPhotoMessageFromUser(photoFileId, s.Message.Caption, s.User.ID, number)
		if err != nil {
			return err
		}

		text := b.GlobalBot.LangText(s.BotLang, "msg_to_admin", s.Message.From.FirstName, s.Message.Caption)

		return b.BaseBotSrv.NewParseMarkUpPhotoMessage(id, nil, text, tgbotapi.FileID(photoFileId))
	}

	err = b.Repo.AddMessageFromUser(s.Message.Text, s.Message.From.ID, number)
	if err != nil {
		return err
	}

	text := b.GlobalBot.LangText(s.BotLang, "msg_to_admin", s.Message.From.FirstName, s.Message.Text)

	return b.BaseBotSrv.NewParseMessage(id, text)
}

func (b *BotService) LiveChats(s *model.Situation) error {
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

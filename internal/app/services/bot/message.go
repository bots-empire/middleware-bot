package bot

import (
	"fmt"
	"github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/BlackRRR/middleware-bot/internal/db/redis"
	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
	"strings"
	"time"
)

const (
	lengthInWide   = 9
	lengthInHeight = 3
)

func (b *BotService) StartCommand(s *model.Situation) error {
	info, err := b.getIncomeInfo(s.User.ID)
	if err != nil {
		return err
	}
	var adminIDs []int64

	if info == nil {
		req := &model.GetAdmins{
			Code: Sup,
		}

		adminIDs, err = b.getAdmins(req)
		if err != nil {
			return err
		}
	} else {
		req := &model.GetAdmins{
			Code:       Sup,
			Additional: []string{info.BotName},
		}

		adminIDs, err = b.getAdmins(req)
		if err != nil {
			return err
		}
	}

	if adminIDs == nil {
		//no admins in set
		text := b.GlobalBot.LangText(s.BotLang, "no_admins", s.User.UserName)
		return b.BaseBotSrv.NewParseMessage(s.User.ID, text)
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

	formatUserID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return err
	}
	//set farther connection from admin to user
	redis.RdbSetUser(s.User.ID, "/answer?"+userID)

	number, err := b.Repo.GetChatNumber(formatUserID, s.User.ID)
	if err != nil {
		return err
	}

	//check if admin send photo
	if s.Message.Photo != nil {
		adminPhotoModel := &model.CommonMessages{
			ChatNumber:        number,
			AdminID:           s.User.ID,
			AdminMessage:      s.Message.Caption,
			AdminPhotoMessage: s.Message.Photo[0].FileID,
			MessageTime:       time.Now(),
		}
		photoFileId := s.Message.Photo[0].FileID

		err := b.Repo.AddMessageToCommon(adminPhotoModel)
		if err != nil {
			return err
		}

		err = b.Repo.UpdateLastMessageTime(time.Now())
		if err != nil {
			return err
		}

		return b.BaseBotSrv.NewParseMarkUpPhotoMessage(formatUserID, nil, s.Message.Caption, tgbotapi.FileID(photoFileId))
	}

	//add message from admin in db
	adminMessageModel := &model.CommonMessages{
		ChatNumber:        number,
		AdminID:           s.User.ID,
		AdminMessage:      s.Message.Text,
		AdminPhotoMessage: "",
		MessageTime:       time.Now(),
	}
	err = b.Repo.AddMessageToCommon(adminMessageModel)
	if err != nil {
		return err
	}

	err = b.Repo.UpdateLastMessageTime(time.Now())
	if err != nil {
		return err
	}

	return b.BaseBotSrv.NewParseMessage(formatUserID, s.Message.Text)
}

func (b *BotService) QuestionAdmin(s *model.Situation) error {
	level := redis.GetLevel(s.Message.From.ID)
	adminID := strings.Split(level, "?")[1]
	fmt.Println(level)
	formatAdminID, err := strconv.ParseInt(adminID, 10, 64)
	if err != nil {
		return err
	}
	//set farther connection from user to admin
	redis.RdbSetUser(s.User.ID, "/question_to_admin?"+adminID)

	number, err := b.Repo.GetChatNumber(s.User.ID, formatAdminID)
	if err != nil {
		return err
	}

	//check if user send photo
	if s.Message.Photo != nil {
		userMessagesModel := &model.CommonMessages{
			ChatNumber:       number,
			UserID:           s.User.ID,
			UserMessage:      s.Message.Caption,
			UserPhotoMessage: s.Message.Photo[0].FileID,
			MessageTime:      time.Now(),
		}

		err := b.Repo.AddMessageToCommon(userMessagesModel)
		if err != nil {
			return err
		}

		err = b.Repo.UpdateLastMessageTime(time.Now())
		if err != nil {
			return err
		}

		text := b.GlobalBot.LangText(s.BotLang, "msg_to_admin",
			number,
			s.Message.From.ID,
			s.Message.From.FirstName,
			s.Message.Caption)

		return b.BaseBotSrv.NewParseMarkUpPhotoMessage(formatAdminID, nil, text, tgbotapi.FileID(s.Message.Photo[0].FileID))
	}

	//add message from user in db
	userMessagesModel := &model.CommonMessages{
		ChatNumber:       number,
		UserID:           s.User.ID,
		UserMessage:      s.Message.Text,
		UserPhotoMessage: "",
		MessageTime:      time.Now(),
	}

	err = b.Repo.AddMessageToCommon(userMessagesModel)
	if err != nil {
		return err
	}

	err = b.Repo.UpdateLastMessageTime(time.Now())
	if err != nil {
		return err
	}
	//text from user to admin with some infomation
	text := b.GlobalBot.LangText(s.BotLang, "msg_to_admin",
		number,
		s.Message.From.ID,
		s.Message.From.FirstName,
		s.Message.Text)

	return b.BaseBotSrv.NewParseMessage(formatAdminID, text)
}

func (b *BotService) LiveChats(s *model.Situation) error {
	return b.firstList(s, "number")
}

func (b *BotService) EndingChat(s *model.Situation) error {
	return b.firstList(s, "delete")
}

func (b *BotService) InfoChatLists(s *model.Situation) error {
	return b.firstList(s, "lists")
}

func (b *BotService) SetNote(s *model.Situation) error {
	level := redis.GetLevel(s.User.ID)
	chatNumber := strings.Split(level, "?")[1]
	userID := strings.Split(level, "?")[2]

	num, err := strconv.Atoi(chatNumber)
	if err != nil {
		return err
	}

	err = b.Repo.SetUserNote(num, s.Message.Text)
	if err != nil {
		return err
	}

	msgID := redis.GetMsgID(s.User.ID)
	redis.RdbSetUser(s.User.ID, "/answer?"+userID)

	msg := tgbotapi.NewDeleteMessage(s.User.ID, msgID)
	err = b.BaseBotSrv.SendMsgToUser(msg, s.User.ID)
	if err != nil {
		return err
	}

	text := b.GlobalBot.LangText(s.BotLang, "note_successful")

	err = b.BaseBotSrv.NewParseMessage(s.User.ID, text)
	if err != nil {
		return err
	}

	s.CallbackQuery = &tgbotapi.CallbackQuery{
		Data: "chat_info?" + chatNumber,
	}

	return b.ChatInfo(s)

}

func (b *BotService) firstList(s *model.Situation, action string) error {
	var chats []*model.AnonymousChat
	if action == "admin" {
		chat, err := b.Repo.GetChatNumberWhereLiveChat(true)
		if err != nil {
			return err
		}

		chats = chat

	} else {
		chat, err := b.Repo.GetChatNumberAndTimeWhereLiveChat(s.User.ID, true)
		if err != nil {
			return err
		}

		chats = chat
	}

	var (
		inlineMarkUp tgbotapi.InlineKeyboardMarkup
		buttons      []tgbotapi.InlineKeyboardButton
		wideCount    int
		heightCount  int
		listNum      int
	)

	if inlineMarkUp.InlineKeyboard == nil {
		inlineMarkUp.InlineKeyboard = make([][]tgbotapi.InlineKeyboardButton, 0)
	}

	for _, val := range chats {
		button := b.getChatNumberButton(s.BotLang, val.ChatNumber, action)

		//set array buttons in height
		buttons = append(buttons, button)
		wideCount += 1
		if heightCount == lengthInHeight {
			listNum += 1
			prevAndNextButtons := b.getPrevAndNextButtons(s.BotLang, listNum, action)

			//create keyboard
			inlineMarkUp.InlineKeyboard = append(inlineMarkUp.InlineKeyboard, prevAndNextButtons)

			text := b.GlobalBot.LangText(s.BotLang, "live_"+action)
			return b.BaseBotSrv.NewParseMarkUpMessage(s.User.ID, &inlineMarkUp, text)
		}

		//set next row
		if wideCount == lengthInWide {
			wideCount = 0
			heightCount += 1
			inlineMarkUp.InlineKeyboard = append(inlineMarkUp.InlineKeyboard, buttons)
			buttons = nil
		}
	}

	if buttons == nil {
		text := b.GlobalBot.LangText(s.BotLang, "live_"+action)
		return b.BaseBotSrv.NewParseMarkUpMessage(s.User.ID, nil, text)
	}

	inlineMarkUp.InlineKeyboard = append(inlineMarkUp.InlineKeyboard, buttons)

	text := b.GlobalBot.LangText(s.BotLang, "live_"+action)
	return b.BaseBotSrv.NewParseMarkUpMessage(s.User.ID, &inlineMarkUp, text)
}

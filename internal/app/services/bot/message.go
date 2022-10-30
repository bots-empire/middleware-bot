package bot

import (
	"fmt"
	"github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/BlackRRR/middleware-bot/internal/db/redis"
	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
	"strings"
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
		fmt.Println(info.BotName)
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

	fmt.Println(adminIDs)
	for _, id := range adminIDs {
		if s.User.ID == id {
			text := b.GlobalBot.LangText(s.BotLang, "you_are_admin")
			return b.BaseBotSrv.NewParseMessage(s.User.ID, text)
		}

		fmt.Println(id)
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
		photoFileId := s.Message.Photo[0].FileID

		err := b.Repo.AddPhotoMessageFromAdmin(photoFileId, s.Message.Caption, s.User.ID, number)
		if err != nil {
			return err
		}

		return b.BaseBotSrv.NewParseMarkUpPhotoMessage(formatUserID, nil, s.Message.Caption, tgbotapi.FileID(photoFileId))
	}

	//add message from admin in db
	err = b.Repo.AddMessageFromAdmin(s.Message.Text, s.User.ID, number)
	if err != nil {
		return err
	}

	return b.BaseBotSrv.NewParseMessage(formatUserID, s.Message.Text)
}

func (b *BotService) QuestionAdmin(s *model.Situation) error {
	level := redis.GetLevel(s.Message.From.ID)
	adminID := strings.Split(level, "?")[1]
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
		photoFileId := s.Message.Photo[0].FileID

		err := b.Repo.AddPhotoMessageFromUser(photoFileId, s.Message.Caption, s.User.ID, number)
		if err != nil {
			return err
		}

		text := b.GlobalBot.LangText(s.BotLang, "msg_to_admin", s.Message.From.FirstName, s.Message.Caption)

		return b.BaseBotSrv.NewParseMarkUpPhotoMessage(formatAdminID, nil, text, tgbotapi.FileID(photoFileId))
	}

	//add message from user in db
	err = b.Repo.AddMessageFromUser(s.Message.Text, s.User.ID, number)
	if err != nil {
		return err
	}

	//text from user to admin with some infomation
	text := b.GlobalBot.LangText(s.BotLang, "msg_to_admin", s.Message.From.FirstName, s.Message.Text)

	return b.BaseBotSrv.NewParseMessage(formatAdminID, text)
}

func (b *BotService) LiveChats(s *model.Situation) error {
	return b.firstList(s, "number")
}

func (b *BotService) EndingChat(s *model.Situation) error {
	return b.firstList(s, "delete")
}

func (b *BotService) InfoChatLists(s *model.Situation) error {
	return b.firstList(s, "info")
}

//func (b *BotService) getRows(buttons []tgbotapi.InlineKeyboardButton) [][]tgbotapi.InlineKeyboardButton {
//	var rows [][]tgbotapi.InlineKeyboardButton
//	rows = append(rows, buttons)
//
//	return rows
//}
//
//func (b *BotService) getButtons(button tgbotapi.InlineKeyboardButton) []tgbotapi.InlineKeyboardButton {
//	var buttons []tgbotapi.InlineKeyboardButton
//	buttons = append(buttons, button)
//
//	return buttons
//}
//
//func (b *BotService) getMarkup(rows [][]tgbotapi.InlineKeyboardButton) tgbotapi.InlineKeyboardMarkup {
//	var markUp tgbotapi.InlineKeyboardMarkup
//	markUp.InlineKeyboard = rows
//
//	return markUp
//}

func (b *BotService) firstList(s *model.Situation, action string) error {
	chats, err := b.Repo.GetChatNumberWhereLiveChat(s.User.ID, true)
	if err != nil {
		return err
	}

	var (
		markUp      tgbotapi.InlineKeyboardMarkup
		buttons     []tgbotapi.InlineKeyboardButton
		rows        [][]tgbotapi.InlineKeyboardButton
		wideCount   int
		heightCount int
		listNum     int
	)

	for _, val := range chats {
		button := b.getChatNumberButton(s.BotLang, val, action)

		//set array buttons in height
		buttons = append(buttons, button)
		wideCount += 1
		if heightCount == lengthInHeight {
			listNum += 1
			prevAndNextButtons := b.getPrevAndNextButtons(s.BotLang, listNum, action)

			//create keyboard
			rows = append(rows, prevAndNextButtons)
			markUp.InlineKeyboard = rows

			text := b.GlobalBot.LangText(s.BotLang, "live_chats")
			return b.BaseBotSrv.NewParseMarkUpMessage(s.User.ID, &markUp, text)
		}

		//set next row
		if wideCount == lengthInWide {
			wideCount = 0
			heightCount += 1
			rows = append(rows, buttons)
			buttons = nil
		}
	}

	//create keyboard
	markUp.InlineKeyboard = rows

	text := b.GlobalBot.LangText(s.BotLang, "live_chats")
	return b.BaseBotSrv.NewParseMarkUpMessage(s.User.ID, &markUp, text)
}

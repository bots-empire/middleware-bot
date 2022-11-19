package bot

import (
	"fmt"
	"github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

func (b *BotService) Admin(s *model.Situation) error {
	//TODO: take admins from ams
	//if s.User.ID != 872383555 {
	//	return b.BaseBotSrv.SendSimpleMsg(s.User.ID, b.GlobalBot.LangText(s.BotLang, "start_text"))
	//}

	return b.firstList(s, "admin")
}

func (b *BotService) SupMessages(s *model.Situation) error {
	chatNumber := strings.Split(s.CallbackQuery.Data, "?")[1]
	chatNumInt, err := strconv.Atoi(chatNumber)
	if err != nil {
		return err
	}

	userID, err := b.Repo.GetUserIDChat(chatNumInt)
	if err != nil {
		return err
	}

	userName, err := b.Repo.GetUserName(userID)
	if err != nil {
		return err
	}

	info, err := b.getIncomeInfo(userID)
	if err != nil {
		return err
	}

	_, adminID, err := b.Repo.GetAdminAndUserID(chatNumInt)
	if err != nil {
		return err
	}

	markUp := msgs.NewIlMarkUp(msgs.NewIlRow(msgs.NewIlDataButton("check_messages", "/sup_messages?"+chatNumber))).Build(b.GlobalBot.Language[s.BotLang])

	if info == nil {
		text := b.GlobalBot.LangText(s.BotLang, "user_info_unknown_admin",
			chatNumber,
			userID,
			userName,
			adminID)
		return b.BaseBotSrv.NewParseMarkUpMessage(s.User.ID, &markUp, text)
	}

	text := b.GlobalBot.LangText(s.BotLang, "user_info_admin",
		chatNumber,
		userID,
		userName,
		adminID,
		info.IncomeSource,
		info.BotName,
		info.TypeBot,
		info.BotLink)

	return b.BaseBotSrv.NewParseMarkUpMessage(s.User.ID, markUp, text)
}

func (b *BotService) SendMessages(s *model.Situation) error {
	var text string
	var medias []interface{}

	chatNumber := strings.Split(s.CallbackQuery.Data, "?")[1]
	chatNumInt, err := strconv.Atoi(chatNumber)
	if err != nil {
		return err
	}

	messages, err := b.Repo.GetSupsMessages(chatNumInt)
	if err != nil {
		return err
	}

	for _, val := range messages {
		//TODO: handle 10 photos max in tg
		sprintText, msg := splitMessages(val)

		if msg != nil {
			medias = append(medias, msg)
		}

		text += sprintText
	}

	config := tgbotapi.MediaGroupConfig{
		ChatID: s.User.ID,
		Media:  medias,
	}

	if text == "" && config.Media == nil {
		err = b.BaseBotSrv.NewParseMessage(s.User.ID, b.GlobalBot.LangText(s.BotLang, "no_messages_subs"))
		if err != nil {
			return err
		}
	}

	if config.Media == nil {
		subsMsg := b.GlobalBot.LangText(s.BotLang, "subs_messages", text)

		return b.BaseBotSrv.NewParseMessage(s.User.ID, subsMsg)
	}

	subsMsg := b.GlobalBot.LangText(s.BotLang, "subs_messages", text)

	err = b.BaseBotSrv.NewParseMessage(s.User.ID, subsMsg)
	if err != nil {
		return err
	}

	_, err = b.GlobalBot.GetBot().SendMediaGroup(config)
	if err != nil {
		return errors.Wrap(err, "failed to send media")
	}

	return nil
}

func splitMessages(val *model.CommonMessages) (string, *tgbotapi.InputMediaPhoto) {
	var text string

	if val.UserPhotoMessage != "" {
		if val.UserMessage != "" {
			msg := photoConfig(val.UserMessage)
			text += fmt.Sprintf(strconv.FormatInt(val.UserID, 10) + ": " + val.UserMessage + "  (Сообщение с медиа, считать по очереди добавления)" + "\n")

			return text, msg
		}

		msg := photoConfig(val.UserPhotoMessage)

		return "", msg
	}

	if val.UserMessage != "" {
		text += fmt.Sprintf(strconv.FormatInt(val.UserID, 10) + ": " + val.UserMessage + "\n")
		return text, nil
	}

	if val.AdminPhotoMessage != "" {
		if val.AdminPhotoMessage != "" {
			msg := photoConfig(val.AdminPhotoMessage)
			text += fmt.Sprintf(strconv.FormatInt(val.AdminID, 10) + ": " + val.AdminMessage + "  (Сообщение с медиа, считать по очереди добавления)" + "\n")

			return text, msg
		}

		msg := photoConfig(val.AdminPhotoMessage)

		return "", msg
	}

	if val.AdminMessage != "" {
		text += fmt.Sprintf(strconv.FormatInt(val.AdminID, 10) + ": " + val.AdminMessage + "\n")
		return text, nil
	}

	return "", nil
}

func photoConfig(photoMessage string) *tgbotapi.InputMediaPhoto {
	inputMediaConfig := &tgbotapi.InputMediaPhoto{
		BaseInputMedia: tgbotapi.BaseInputMedia{
			Type:  "photo",
			Media: tgbotapi.FileID(photoMessage),
		}}
	return inputMediaConfig
}

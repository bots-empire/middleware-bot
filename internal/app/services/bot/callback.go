package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/BlackRRR/middleware-bot/internal/db/redis"
	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	GoHttp = "http://"
	Sup    = "SUPPORT-MINER"
	bot    = "MINER-BOT"
)

func (b *BotService) CallAdmin(s *model.Situation) error {
	//get info
	info, err := b.getIncomeInfo(s.User.ID)
	if err != nil {
		return err
	}

	var adminIDs []int64

	if info == nil {
		req := &model.GetAdmins{
			Code: Sup,
		}

		//get all admins
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

		//get current bot name admins
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

	for _, adminID := range adminIDs {
		var (
			text   string
			markUp tgbotapi.InlineKeyboardMarkup
		)
		if info == nil {
			//user unknown
			text = b.GlobalBot.LangText(s.BotLang, "write_answer_null", s.User.UserName)
			markUp = msgs.NewIlMarkUp(msgs.NewIlRow(
				msgs.NewIlDataButton("start_chat_with_user",
					"/start_chat?"+strconv.FormatInt(s.User.ID, 10)))).Build(b.GlobalBot.Language[s.BotLang])

		} else {
			//user known
			text = b.GlobalBot.LangText(s.BotLang, "write_answer",
				s.User.UserName,
				info.IncomeSource,
				info.BotName,
				info.TypeBot,
				info.BotLink)
			markUp = msgs.NewIlMarkUp(msgs.NewIlRow(
				msgs.NewIlDataButton("start_chat_with_user",
					"/start_chat?"+strconv.FormatInt(s.User.ID, 10)))).Build(b.GlobalBot.Language[s.BotLang])

		}

		err := b.BaseBotSrv.NewParseMarkUpMessage(adminID, markUp, text)
		if err != nil {
			return err
		}
	}

	text := b.GlobalBot.LangText(s.BotLang, "admin_called")

	err = b.BaseBotSrv.NewEditMarkUpMessage(s.User.ID, s.CallbackQuery.Message.MessageID, nil, text)
	if err != nil {
		return errors.Wrap(err, "failed to send message to admin")
	}

	return nil
}

func (b *BotService) StartChat(s *model.Situation) error {
	userID := strings.Split(s.CallbackQuery.Data, "?")[1]
	userFormatID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return err
	}
	adminID := s.User.ID

	chatStarts, err := b.Repo.CheckStartedChatsUser(userFormatID)
	if err != nil {
		return err
	}

	//check if chat exists with different sup
	for _, val := range chatStarts {
		if val {
			text := b.GlobalBot.LangText(s.BotLang, "different_admin")
			return b.BaseBotSrv.NewParseMessage(adminID, text)
		}
	}

	//set what chat was started with sup
	chat := &model.AnonymousChat{
		AdminId:         adminID,
		UserId:          userFormatID,
		Note:            " ",
		ChatStart:       true,
		LastMessageTime: time.Now(),
	}

	err = b.Repo.ChangeStartChat(chat)
	if err != nil {
		return err
	}

	chatNuM, err := b.Repo.GetChatNumberWhereLiveChatUser(adminID, userFormatID, true)
	if err != nil {
		return err
	}

	text := b.GlobalBot.LangText(s.BotLang, "write_answer_to_user", chatNuM)

	//set connection for admin with user
	redis.RdbSetUser(adminID, "/answer?"+userID)

	msg := tgbotapi.NewMessage(adminID, text)
	msg.ReplyMarkup = createMainMenu().Build(b.GlobalBot.Language[s.BotLang])

	//send to admin what chat was started and show menu
	err = b.BaseBotSrv.SendMsgToUser(msg, adminID)
	if err != nil {
		return err
	}

	text = b.GlobalBot.LangText(s.BotLang, "chat_started_user")

	//set connection for user with admin
	redis.RdbSetUser(userFormatID, "/question_to_admin?"+strconv.FormatInt(+adminID, 10))

	// send to user what chat was started
	return b.BaseBotSrv.NewParseMessage(userFormatID, text)
}

func (b *BotService) SelectChat(s *model.Situation) error {
	chatNumber := strings.Split(s.CallbackQuery.Data, "?")[1]
	chatNumInt, err := strconv.Atoi(chatNumber)
	if err != nil {
		return err
	}

	userID, err := b.Repo.GetUserIDChat(chatNumInt)
	if err != nil {
		return err
	}

	userIDStr := strconv.FormatInt(userID, 10)

	redis.RdbSetUser(s.User.ID, "/answer?"+userIDStr)

	text := b.GlobalBot.LangText(s.BotLang, "chat_selected", chatNumber)

	return b.BaseBotSrv.NewParseMessage(s.User.ID, text)
}

func (b *BotService) DeleteChat(s *model.Situation) error {
	data := strings.Split(s.CallbackQuery.Data, "?")[1]
	chatNumber, err := strconv.Atoi(data)
	if err != nil {
		return err
	}
	err = b.Repo.StopChat(chatNumber)
	if err != nil {
		return err
	}

	userID, err := b.Repo.GetUserIDChat(chatNumber)
	if err != nil {
		return err
	}

	redis.RdbSetUser(userID, "main")
	redis.RdbSetUser(s.User.ID, "main")

	text := b.GlobalBot.LangText(s.BotLang, "chat_ended_user")

	err = b.BaseBotSrv.NewParseMessage(s.User.ID, text)
	if err != nil {
		return err
	}

	text = b.GlobalBot.LangText(s.BotLang, "chat_deleted", chatNumber)

	return b.BaseBotSrv.NewParseMessage(s.User.ID, text)
}

func (b *BotService) ChatInfo(s *model.Situation) error {
	chatNumber := strings.Split(s.CallbackQuery.Data, "?")[1]
	formatChatNumber, err := strconv.Atoi(chatNumber)
	if err != nil {
		return err
	}

	userID, err := b.Repo.GetUserIDChat(formatChatNumber)
	if err != nil {
		return err
	}

	info, err := b.getIncomeInfo(userID)
	if err != nil {
		return err
	}

	note, err := b.Repo.GetUserNote(formatChatNumber)
	if err != nil {
		return err
	}

	if note == " " || note == "" {
		note = b.GlobalBot.LangText(s.BotLang, "write_note")
	}

	markUp := msgs.NewIlMarkUp(msgs.NewIlRow(msgs.NewIlDataButton("new_note", "/set_note?"+chatNumber))).Build(b.GlobalBot.Language[s.BotLang])

	//lastMessageFromUser, err := b.Repo.GetLastMessageFromUser(userID, formatChatNumber)
	//if err != nil {
	//	return err
	//}
	//
	//lastMessageFromAdmin, err := b.Repo.GetLastMessageFromAdmin(userID, formatChatNumber)
	//if err != nil {
	//	return err
	//}

	userName, err := b.Repo.GetUserName(userID)
	if err != nil {
		return err
	}

	if info == nil {
		text := b.GlobalBot.LangText(s.BotLang, "user_info_unknown",
			chatNumber,
			userID,
			userName,
			note)
		return b.BaseBotSrv.NewParseMarkUpMessage(s.User.ID, &markUp, text)
	}

	text := b.GlobalBot.LangText(s.BotLang, "user_info",
		chatNumber,
		userID,
		userName,
		info.IncomeSource,
		info.BotName,
		info.TypeBot,
		info.BotLink,
		note)

	return b.BaseBotSrv.NewParseMarkUpMessage(s.User.ID, &markUp, text)

}

func (b *BotService) SetNotes(s *model.Situation) error {
	data := strings.Split(s.CallbackQuery.Data, "?")[1]
	level := redis.GetLevel(s.User.ID)
	userID := strings.Split(level, "?")[1]
	redis.RdbSetMessageID(s.User.ID, s.CallbackQuery.Message.MessageID)

	redis.RdbSetUser(s.User.ID, "/set_note_msg?"+data+"?"+userID)

	return b.BaseBotSrv.NewParseMessage(s.User.ID, b.GlobalBot.LangText(s.BotLang, "set_new_note"))
}

func (b *BotService) SetListChats(s *model.Situation) error {
	needList := strings.Split(s.CallbackQuery.Data, "?")[1]
	action := strings.Split(s.CallbackQuery.Data, "?")[2]
	formatNeedList, err := strconv.Atoi(needList)
	if err != nil {
		return err
	}

	chats, err := b.Repo.GetChatNumberAndTimeWhereLiveChat(s.User.ID, true)
	if err != nil {
		return err
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
		if heightCount == 0 {
			return b.BaseBotSrv.SendAnswerCallback(s.CallbackQuery, "lower_than_1")
		}

		if listNum == formatNeedList {
			wideCount += 1
			button := b.getChatNumberButton(s.BotLang, val.ChatNumber, "number")

			buttons = append(buttons, button)

			if heightCount == lengthInHeight {
				prevAndNextButtons := b.getPrevAndNextButtons(s.BotLang, listNum, action)

				inlineMarkUp.InlineKeyboard = append(inlineMarkUp.InlineKeyboard, prevAndNextButtons)

				text := b.GlobalBot.LangText(s.BotLang, "live_"+action)
				return b.BaseBotSrv.NewEditMarkUpMessage(s.User.ID, s.CallbackQuery.Message.MessageID, &inlineMarkUp, text)
			}

			if wideCount == lengthInWide {
				wideCount = 0
				heightCount += 1
				inlineMarkUp.InlineKeyboard = append(inlineMarkUp.InlineKeyboard, buttons)
			}
		}

		wideCount += 1
		if heightCount == lengthInHeight {
			listNum += 1
			heightCount = 0
		}

		if wideCount == lengthInWide {
			wideCount = 0
			heightCount += 1
		}
	}

	if listNum != formatNeedList {
		return b.BaseBotSrv.SendAnswerCallback(s.CallbackQuery, "no_more_lists")
	}

	if inlineMarkUp.InlineKeyboard == nil {
		inlineMarkUp.InlineKeyboard = append(inlineMarkUp.InlineKeyboard, buttons)
	}

	prevAndNextButtons := b.getPrevAndNextButtons(s.BotLang, listNum, action)

	inlineMarkUp.InlineKeyboard = append(inlineMarkUp.InlineKeyboard, prevAndNextButtons)

	text := b.GlobalBot.LangText(s.BotLang, "live_"+action)
	return b.BaseBotSrv.NewEditMarkUpMessage(s.User.ID, s.CallbackQuery.Message.MessageID, &inlineMarkUp, text)
}

func createMainMenu() msgs.MarkUp {
	return msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("live_chats")),
		msgs.NewRow(msgs.NewDataButton("delete_chats")),
		msgs.NewRow(msgs.NewDataButton("chat_info")))
}

func (b *BotService) getAdmins(req *model.GetAdmins) ([]int64, error) {
	marshal, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	body := bytes.NewReader(marshal)

	resp, err := http.Post(GoHttp+b.Server.Ip+b.Server.GetAdmins, "application/json", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		log.Fatalf("Get admins: Response failed with status code: %d and\nreq: %s\n", resp.StatusCode, data)
	}

	var adminIDs []int64

	err = json.Unmarshal(data, &adminIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal admin ids")
	}

	return adminIDs, nil
}

func (b *BotService) getIncomeInfo(userID int64) (*model.IncomeInfo, error) {
	req := &model.GetIncomeInfo{
		UserID:  userID,
		TypeBot: bot,
	}

	marshal, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(marshal)

	resp, err := http.Post(GoHttp+b.Server.Ip+b.Server.GetInfo, "application/json", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		log.Fatalf("Get Income info: Response failed with status code: %d and\nbody: %s\n", resp.StatusCode, data)
	}

	var info *model.IncomeInfo

	err = json.Unmarshal(data, &info)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal income info")
	}

	return info, nil
}

func (b *BotService) getPrevAndNextButtons(botLang string, listNum int, action string) []tgbotapi.InlineKeyboardButton {
	var prevAndNextButtons []tgbotapi.InlineKeyboardButton
	// button prev list
	data := "/set_list?" + strconv.Itoa(listNum-1) + "?" + action
	button := tgbotapi.InlineKeyboardButton{
		Text:         b.GlobalBot.LangText(botLang, "prev"),
		CallbackData: &data,
	}
	prevAndNextButtons = append(prevAndNextButtons, button)

	//button next list
	data = "/set_list?" + strconv.Itoa(listNum+1) + "?" + action
	button = tgbotapi.InlineKeyboardButton{
		Text:         b.GlobalBot.LangText(botLang, "next"),
		CallbackData: &data,
	}
	prevAndNextButtons = append(prevAndNextButtons, button)

	return prevAndNextButtons
}

func (b *BotService) getChatNumberButton(botLang string, val int, action string) tgbotapi.InlineKeyboardButton {
	data := "/chat_" + action + "?" + strconv.Itoa(val)
	button := tgbotapi.InlineKeyboardButton{
		Text:         b.GlobalBot.LangText(botLang, "chat_number", val),
		CallbackData: &data,
	}

	return button
}

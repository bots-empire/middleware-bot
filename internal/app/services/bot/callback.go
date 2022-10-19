package bot

import (
	"bytes"
	"encoding/json"
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
)

const (
	GoHttp = "http://"
	Sup    = "SUPPORT-MINER"
	bot    = "MINER-BOT"
)

func (b *BotService) CallAdmin(s *model.Situation) error {
	adminIDs, err := b.getAdmins(s.BotLang)
	if err != nil {
		return err
	}

	info, err := b.getIncomeInfo(s.User.ID)
	if err != nil {
		return err
	}

	if adminIDs == nil {
		text := b.GlobalBot.LangText(s.BotLang, "no_admins", s.CallbackQuery.From.UserName)
		return b.BaseBotSrv.NewParseMessage(s.User.ID, text)
	}

	for _, id := range adminIDs {
		var (
			text   string
			markUp tgbotapi.InlineKeyboardMarkup
		)
		if info == nil {
			text = b.GlobalBot.LangText(s.BotLang, "write_answer_null", s.CallbackQuery.From.UserName)
			markUp = msgs.NewIlMarkUp(msgs.NewIlRow(
				msgs.NewIlDataButton("start_chat_with_user",
					"/start_chat?"+strconv.FormatInt(s.User.ID, 10)))).Build(b.GlobalBot.Language[s.BotLang])

		} else {
			text = b.GlobalBot.LangText(s.BotLang, "write_answer",
				s.CallbackQuery.From.UserName,
				info.IncomeSource,
				info.BotName,
				info.TypeBot,
				info.BotLink)
			markUp = msgs.NewIlMarkUp(msgs.NewIlRow(
				msgs.NewIlDataButton("start_chat_with_user",
					"/start_chat?"+strconv.FormatInt(s.User.ID, 10)))).Build(b.GlobalBot.Language[s.BotLang])

		}

		err := b.BaseBotSrv.NewParseMarkUpMessage(id, markUp, text)
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

func (b *BotService) getAdmins(botLang string) ([]int64, error) {
	req := &model.GetAdmins{
		Code:       Sup,
		Additional: []string{botLang},
	}

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

func (b *BotService) StartChat(s *model.Situation) error {
	userID := strings.Split(s.CallbackQuery.Data, "?")[1]
	formatID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return err
	}

	chatNum, started, err := b.Repo.GetChatConflict(s.CallbackQuery.From.ID, formatID)
	if err != nil {
		return err
	}

	if chatNum != 0 && started {
		text := b.GlobalBot.LangText(s.BotLang, "different_admin")

		return b.BaseBotSrv.NewParseMessage(s.User.ID, text)
	}

	chatStarts, err := b.Repo.CheckStartedChatsUser(formatID)
	if err != nil {
		return err
	}

	for _, val := range chatStarts {
		if val {
			text := b.GlobalBot.LangText(s.BotLang, "different_admin")
			err := b.BaseBotSrv.NewParseMessage(s.User.ID, text)
			if err != nil {
				return err
			}
		}
	}

	err = b.Repo.ChangeStartChat(s.CallbackQuery.From.ID, formatID, true)
	if err != nil {
		return err
	}

	chatNuM, _, err := b.Repo.GetChatConflict(s.CallbackQuery.From.ID, formatID)
	if err != nil {
		return err
	}

	text := b.GlobalBot.LangText(s.BotLang, "write_answer_to_user", chatNuM)

	redis.RdbSetUser(s.User.ID, "/answer?"+userID)

	msg := tgbotapi.NewMessage(s.User.ID, text)
	msg.ReplyMarkup = createMainMenu().Build(b.GlobalBot.Language[s.BotLang])

	err = b.BaseBotSrv.SendMsgToUser(msg, s.User.ID)
	if err != nil {
		return err
	}

	text = b.GlobalBot.LangText(s.BotLang, "chat_started_user")
	data := strings.Split(s.CallbackQuery.Data, "?")[1]

	id, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return err
	}
	redis.RdbSetUser(formatID, "/question_to_admin?"+strconv.FormatInt(+s.User.ID, 10))

	if chatNum != 0 && !started {
		err := b.Repo.ResumeChat(chatNum)
		if err != nil {
			return err
		}

		return b.BaseBotSrv.NewParseMessage(id, text)
	}

	return b.BaseBotSrv.NewParseMessage(id, text)
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

func createMainMenu() msgs.MarkUp {
	return msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("live_chats")),
		msgs.NewRow(msgs.NewDataButton("delete_chats")))
}

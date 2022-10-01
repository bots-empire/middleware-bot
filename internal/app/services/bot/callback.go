package bot

import (
	"github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/BlackRRR/middleware-bot/internal/db/redis"
	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

func (b *BotService) CallAdmin(s *model.Situation) error {
	//resp, err := http.Get("/ams.zenleads.tech/v1/access/check")
	//if err != nil {
	//	return err
	//}
	//
	//data, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	return err
	//}
	//
	//var adminIDs []int64
	//
	//err = json.Unmarshal(data, &adminIDs)
	//if err != nil {
	//	return errors.Wrap(err, "failed to unmarshal admin ids")
	//}

	//for _, id := range adminIDs {
	//	err := b.Repo.AddAdminIDsToDBIfNotExist(id)
	//	if err != nil {
	//		return errors.Wrap(err, "failed to add admin id to database")
	//	}
	//
	//	text := b.GlobalBot.LangText(s.BotLang, "call_admin", s.CallbackQuery.From.ID)
	//
	//	err = b.BaseBotSrv.NewParseMessage(id, text)
	//	if err != nil {
	//		return errors.Wrap(err, "failed to send message to admin")
	//	}
	//}

	var id int64

	id = 872383555
	//1418862576

	err := b.Repo.AddAdminIDsToDBIfNotExist(id)
	if err != nil {
		return errors.Wrap(err, "failed to add admin id to database")
	}

	text := b.GlobalBot.LangText(s.BotLang, "admin_called")

	err = b.BaseBotSrv.NewParseMessage(s.User.ID, text)
	if err != nil {
		return errors.Wrap(err, "failed to send message to admin")
	}

	text = b.GlobalBot.LangText(s.BotLang, "write_answer")
	markUp := msgs.NewIlMarkUp(msgs.NewIlRow(msgs.NewIlDataButton("start_chat_with_user", "/write_to_user?"+strconv.FormatInt(s.User.ID, 10)))).Build(b.GlobalBot.Language[s.BotLang])

	return b.BaseBotSrv.NewParseMarkUpMessage(id, markUp, text)
}

func (b *BotService) WriteToUser(s *model.Situation) error {

	userID := strings.Split(s.CallbackQuery.Data, "?")[1]
	formatID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return err
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

	text := b.GlobalBot.LangText(s.BotLang, "write_answer_to_user")
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

	chatNum, started, err := b.Repo.GetChatConflict(s.CallbackQuery.From.ID, formatID)
	if err != nil {
		return err
	}

	if chatNum != 0 && !started {
		err := b.Repo.ResumeChat(chatNum)
		if err != nil {
			return err
		}

		return b.BaseBotSrv.NewParseMessage(id, text)
	}

	err = b.Repo.ChangeStartChat(s.CallbackQuery.From.ID, formatID, true)
	if err != nil {
		return err
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

	text := b.GlobalBot.LangText(s.BotLang, "chat_deleted")

	return b.BaseBotSrv.NewParseMessage(s.User.ID, text)
}

func createMainMenu() msgs.MarkUp {
	return msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("live_chats")),
		msgs.NewRow(msgs.NewDataButton("delete_chats")))
}

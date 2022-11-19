package repository

import (
	"context"
	"github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"time"
)

type Repository struct {
	Pool      *pgxpool.Pool
	globalBot *model.GlobalBot
	msgs      *msgs.Service
	ctx       context.Context
}

func NewRepository(pool *pgxpool.Pool, msgs *msgs.Service, globalBot *model.GlobalBot) *Repository {
	return &Repository{pool, globalBot, msgs, context.Background()}
}

func (r *Repository) CheckingTheUser(message *tgbotapi.Message) (*model.User, error) {
	rows, err := r.Pool.Query(r.ctx, `
SELECT id, user_name FROM middleware.users 
	WHERE id = $1;`,
		message.From.ID)
	if err != nil {
		return nil, errors.Wrap(err, "get user")
	}

	users, err := readUsers(rows)
	if err != nil {
		return nil, errors.Wrap(err, "read user")
	}

	switch len(users) {
	case 0:
		user := createSimpleUser(message)
		if err := r.addNewUser(user); err != nil {
			return nil, errors.Wrap(err, "add new user")
		}
		return user, nil
	case 1:
		return users[0], nil
	default:
		return nil, model.ErrFoundTwoUsers
	}
}

func (r *Repository) GetUserName(userID int64) (string, error) {
	var userName string
	err := r.Pool.QueryRow(r.ctx, "SELECT user_name FROM middleware.users WHERE id = $1", userID).Scan(&userName)
	if err != nil {
		return "", err
	}

	return userName, nil
}

func (r *Repository) addNewUser(u *model.User) error {
	_, err := r.Pool.Exec(r.ctx, `INSERT INTO middleware.users VALUES ($1,$2);`, u.ID, u.UserName)
	if err != nil {
		return errors.Wrap(err, "insert new user")
	}

	_ = r.msgs.SendSimpleMsg(u.ID, r.globalBot.LangText(r.globalBot.BotLang, "start_text"))

	return nil
}

func createSimpleUser(message *tgbotapi.Message) *model.User {
	if message.From.UserName != "" {
		return &model.User{
			ID:       message.From.ID,
			UserName: message.From.UserName,
		}
	}

	return &model.User{
		ID: message.From.ID,
	}
}

func (r *Repository) GetUser(id int64) (*model.User, error) {
	rows, err := r.Pool.Query(r.ctx, `
SELECT * FROM middleware.users
	WHERE id = $1;`,
		id)
	if err != nil {
		return nil, err
	}

	users, err := readUsers(rows)
	if err != nil || len(users) == 0 {
		return nil, model.ErrUserNotFound
	}
	return users[0], nil
}

func (r *Repository) ChangeStartChat(chat *model.AnonymousChat) error {
	_, err := r.Pool.Exec(r.ctx, `INSERT INTO middleware.anonymous_chat (admin_id,
                                       user_id,
                                       chat_start,
                                       note,
                                       last_message_time) 
VALUES ($1,$2,$3,$4,$5);`,
		chat.AdminId,
		chat.UserId,
		chat.ChatStart,
		chat.Note,
		chat.LastMessageTime,
	)
	if err != nil {
		return errors.Wrap(err, "failed to change start chat")
	}

	return nil
}

func (r *Repository) ResumeChat(chatNumber int) error {
	_, err := r.Pool.Exec(r.ctx, `UPDATE middleware.anonymous_chat SET chat_start = $1 WHERE chat_number = $2`, true, chatNumber)
	if err != nil {
		return errors.Wrap(err, "failed to resume chat")
	}

	return nil
}

func (r *Repository) GetChatConflict(adminID, userID int64) (int, bool, error) {
	var (
		chatNumber int
		chatStart  bool
	)

	err := r.Pool.QueryRow(r.ctx, `
SELECT chat_number,chat_start 
FROM middleware.anonymous_chat 
        WHERE admin_id = $1 
          AND user_id = $2`,
		adminID, userID).Scan(&chatNumber, &chatStart)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return 0, false, nil
		}
		return 0, false, errors.Wrap(err, "failed to get chat conflict")
	}

	return chatNumber, chatStart, nil
}

func (r *Repository) CheckStartedChatsUser(userID int64) ([]bool, error) {
	rows, err := r.Pool.Query(r.ctx, `SELECT chat_start FROM middleware.anonymous_chat WHERE user_id = $1`, userID)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	chatStarts := readChatRows(rows)

	return chatStarts, nil
}

func (r *Repository) GetChatNumber(userID, adminID int64) (int, error) {
	var chatNumber int
	err := r.Pool.QueryRow(r.ctx, `SELECT chat_number FROM middleware.anonymous_chat WHERE admin_id = $1 AND user_id = $2`,
		adminID,
		userID).Scan(&chatNumber)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return 0, nil
		}
		return 0, errors.Wrap(err, "failed to get chat number")
	}

	return chatNumber, nil
}

func (r *Repository) UpdateLastMessageTime(messageTime time.Time) error {
	_, err := r.Pool.Exec(r.ctx, `UPDATE middleware.anonymous_chat SET last_message_time = $1`, messageTime)
	if err != nil {
		return errors.Wrap(err, "failed to update last message time")
	}

	return nil
}

//func (r *Repository) AddMessageFromUser(messages *model.UserMessages) error {
//	_, err := r.Pool.Exec(r.ctx, `
//INSERT INTO middleware.admin_messages (chat_number,
//                                       admin_id,
//                                       admin_message,
//                                       admin_photo_message,
//                                       message_time)
//VALUES ($1,$2,$3,$4,$5)`,
//		messages.ChatNumber,
//		messages.UserID,
//		messages.UserMessage,
//		messages.UserPhotoMessage,
//		messages.MessageTime)
//	if err != nil {
//		return errors.Wrap(err, "failed to add message from user")
//	}
//
//	return nil
//}

//func (r *Repository) AddPhotoMessageFromUser(messages *model.UserMessages) error {
//	_, err := r.Pool.Exec(r.ctx, `
//INSERT INTO middleware.admin_messages (chat_number,
//                                       admin_id,
//                                       admin_message,
//                                       admin_photo_message,
//                                       message_time)
//VALUES ($1,$2,$3,$4,$5)`,
//		messages.ChatNumber,
//		messages.UserID,
//		messages.UserMessage,
//		messages.UserPhotoMessage,
//		messages.MessageTime)
//	if err != nil {
//		return errors.Wrap(err, "failed to add photo message from user")
//	}
//
//	return nil
//}

//func (r *Repository) AddPhotoMessageFromAdmin(messages *model.AdminMessages) error {
//	_, err := r.Pool.Exec(r.ctx, `
//INSERT INTO middleware.admin_messages (chat_number,
//                                       admin_id,
//                                       admin_message,
//                                       admin_photo_message,
//                                       message_time)
//VALUES ($1,$2,$3,$4,$5)`,
//		messages.ChatNumber,
//		messages.AdminID,
//		messages.AdminMessage,
//		messages.AdminPhotoMessage,
//		messages.MessageTime)
//	if err != nil {
//		return errors.Wrap(err, "failed to add photo message from user")
//	}
//	return nil
//}

func (r *Repository) AddMessageToCommon(messages *model.CommonMessages) error {
	_, err := r.Pool.Exec(r.ctx, `
INSERT INTO middleware.common_messages (chat_number, 
                                       admin_id, 
                                       admin_message,
                                       admin_photo_message,
                                       user_id,
                                       user_message,
                                       user_photo_message,
                                       message_time) 
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		messages.ChatNumber,
		messages.AdminID,
		messages.AdminMessage,
		messages.AdminPhotoMessage,
		messages.UserID,
		messages.UserMessage,
		messages.UserPhotoMessage,
		messages.MessageTime)
	if err != nil {
		return errors.Wrap(err, "failed to add message from admin")
	}

	return nil
}

func (r *Repository) GetLastMessageFromUser(userID int64, chatNumber int) (string, error) {
	var userMessage string
	err := r.Pool.QueryRow(r.ctx, "SELECT user_message FROM middleware.messages WHERE user_id = $1 AND chat_number = $2",
		userID,
		chatNumber).Scan(&userMessage)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return "", nil
		}
		return "", errors.Wrap(err, "failed to get last user message")
	}

	return userMessage, nil
}

func (r *Repository) GetLastMessageFromAdmin(adminID int64, chatNumber int) (string, error) {
	var adminMessage string
	err := r.Pool.QueryRow(r.ctx, `SELECT admin_message FROM middleware.messages WHERE admin_id = $1 AND chat_number = $2`,
		adminID,
		chatNumber).Scan(&adminMessage)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return "", nil
		}
		return "", errors.Wrap(err, "failed to get last admin message")
	}

	return adminMessage, nil
}

func (r *Repository) GetSupsMessages(chatNumber int) ([]*model.CommonMessages, error) {
	rows, err := r.Pool.Query(r.ctx, `
SELECT 	chat_number,
   		admin_id,
       	admin_message,
		admin_photo_message,
		user_id,
		user_message,
		user_photo_message,
		message_time FROM middleware.common_messages
		             WHERE chat_number = $1
		             ORDER BY message_time ASC`, &chatNumber)
	if err != nil {
		return nil, err
	}

	messageModel := readRowsSupsMessages(rows)

	return messageModel, nil
}

func readRowsSupsMessages(rows pgx.Rows) []*model.CommonMessages {
	messageModels := make([]*model.CommonMessages, 0)

	for rows.Next() {
		messageModel := &model.CommonMessages{}

		err := rows.Scan(
			&messageModel.ChatNumber,
			&messageModel.AdminID,
			&messageModel.AdminMessage,
			&messageModel.AdminPhotoMessage,
			&messageModel.UserID,
			&messageModel.UserMessage,
			&messageModel.UserPhotoMessage,
			&messageModel.MessageTime)
		if err != nil {
			return nil
		}

		messageModels = append(messageModels, messageModel)
	}

	return messageModels
}

func (r *Repository) GetChatNumberAndTimeWhereLiveChat(adminID int64, chatLive bool) ([]*model.AnonymousChat, error) {
	rows, err := r.Pool.Query(r.ctx, `
SELECT chat_number,last_message_time FROM middleware.anonymous_chat
                   WHERE admin_id = $1 AND chat_start = $2
                   ORDER BY last_message_time ASC`,
		adminID,
		chatLive)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get chat number where live chat")
	}

	readRows, err := r.readRowsModelsAnonymousChat(rows)
	if err != nil {
		return nil, err
	}

	rows.Close()
	return readRows, nil
}

func (r *Repository) GetChatNumberWhereLiveChatUser(adminID, userID int64, chatLive bool) (int, error) {
	var chatStart int
	err := r.Pool.QueryRow(r.ctx, `
SELECT chat_number FROM middleware.anonymous_chat 
                   WHERE admin_id = $1 
                     AND chat_start = $2
                     AND user_id = $3`,
		adminID,
		chatLive,
		userID).Scan(&chatStart)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get chat number where live chat")
	}

	return chatStart, nil
}

func (r *Repository) GetUserIDChat(chatNumber int) (int64, error) {
	var userID int64
	err := r.Pool.QueryRow(r.ctx, `
SELECT user_id FROM middleware.anonymous_chat 
               WHERE chat_number = $1`,
		chatNumber).Scan(&userID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get chat ids")
	}

	return userID, nil
}

func (r *Repository) StopChat(chatNumber int) error {
	_, err := r.Pool.Exec(r.ctx, `
UPDATE middleware.anonymous_chat 
	SET chat_start = $1 
	WHERE chat_number = $2`,
		false,
		chatNumber)
	if err != nil {
		return errors.Wrap(err, "failed to stop chat")
	}

	return nil
}

func (r *Repository) GetAdminAndUserID(chatNumber int) (int64, int64, error) {
	var userID, adminID int64
	err := r.Pool.QueryRow(r.ctx, `
SELECT user_id, admin_id FROM middleware.anonymous_chat
                         WHERE chat_number = $1`,
		chatNumber).Scan(&userID, &adminID)
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to get admin and user id")
	}

	return userID, adminID, nil
}

func (r *Repository) SetUserNote(chatNum int, note string) error {
	_, err := r.Pool.Exec(r.ctx, `UPDATE middleware.anonymous_chat 
SET note = $1
WHERE chat_number = $2`,
		note,
		chatNum)
	if err != nil {
		return errors.Wrap(err, "failed to set user note")
	}

	return nil
}

func (r *Repository) GetUserNote(chatNum int) (string, error) {
	var note string
	err := r.Pool.QueryRow(r.ctx, `SELECT note FROM middleware.anonymous_chat WHERE chat_number = $1`,
		chatNum).Scan(&note)
	if err != nil {
		return "", errors.Wrap(err, "failed to get note")
	}

	return note, nil
}

func (r *Repository) readRowsModelsAnonymousChat(rows pgx.Rows) ([]*model.AnonymousChat, error) {
	modelsAnonymousChat := make([]*model.AnonymousChat, 0)

	for rows.Next() {
		modelAnonymousChat := &model.AnonymousChat{}

		err := rows.Scan(
			&modelAnonymousChat.ChatNumber,
			&modelAnonymousChat.LastMessageTime)
		if err != nil {
			return nil, err
		}

		modelsAnonymousChat = append(modelsAnonymousChat, modelAnonymousChat)
	}

	return modelsAnonymousChat, nil
}

func readChatRows(rows pgx.Rows) []bool {
	var chatStart bool
	var chatStarts []bool
	for rows.Next() {
		err := rows.Scan(&chatStart)
		if err != nil {
			return nil
		}

		chatStarts = append(chatStarts, chatStart)
	}

	return chatStarts
}

func readUsers(rows pgx.Rows) ([]*model.User, error) {
	defer rows.Close()
	var users []*model.User

	for rows.Next() {
		user := &model.User{}

		if err := rows.Scan(
			&user.ID,
			&user.UserName,
		); err != nil {
			return nil, errors.Wrap(err, model.ErrScanSqlRow.Error())
		}

		users = append(users, user)
	}

	return users, nil
}

package repository

import (
	"context"
	model2 "github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
)

type Repository struct {
	Pool      *pgxpool.Pool
	globalBot *model2.GlobalBot
	msgs      *msgs.Service
	ctx       context.Context
}

func NewRepository(pool *pgxpool.Pool, msgs *msgs.Service, globalBot *model2.GlobalBot) *Repository {
	return &Repository{pool, globalBot, msgs, context.Background()}
}

func (r *Repository) CheckingTheUser(message *tgbotapi.Message) (*model2.User, error) {
	rows, err := r.Pool.Query(r.ctx, `
SELECT id FROM middleware.users 
	WHERE id = $1;`,
		message.From.ID)
	if err != nil {
		return nil, errors.Wrap(err, "get user")
	}

	users, err := ReadUsers(rows)
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
		return nil, model2.ErrFoundTwoUsers
	}
}

func (r *Repository) addNewUser(u *model2.User) error {
	_, err := r.Pool.Exec(r.ctx, `INSERT INTO middleware.users VALUES ($1);`, u.ID)
	if err != nil {
		return errors.Wrap(err, "insert new user")
	}

	_ = r.msgs.SendSimpleMsg(u.ID, r.globalBot.LangText(r.globalBot.BotLang, "start_text"))

	return nil
}

func createSimpleUser(message *tgbotapi.Message) *model2.User {
	return &model2.User{
		ID: message.From.ID,
	}
}

func (r *Repository) GetUser(id int64) (*model2.User, error) {
	rows, err := r.Pool.Query(r.ctx, `
SELECT * FROM middleware.users
	WHERE id = $1;`,
		id)
	if err != nil {
		return nil, err
	}

	users, err := ReadUsers(rows)
	if err != nil || len(users) == 0 {
		return nil, model2.ErrUserNotFound
	}
	return users[0], nil
}

func ReadUsers(rows pgx.Rows) ([]*model2.User, error) {
	defer rows.Close()
	var users []*model2.User

	for rows.Next() {
		user := &model2.User{}

		if err := rows.Scan(
			&user.ID,
		); err != nil {
			return nil, errors.Wrap(err, model2.ErrScanSqlRow.Error())
		}

		users = append(users, user)
	}

	return users, nil
}

func (r *Repository) ChangeStartChat(adminID, userID int64, chatStart bool) error {
	_, err := r.Pool.Exec(r.ctx, `INSERT INTO middleware.anonymous_chat (admin_id, user_id, chat_start) 
VALUES ($1,$2,$3);`,
		adminID,
		userID,
		chatStart)
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

	chatStarts := ReadChatRows(rows)

	return chatStarts, nil
}

func ReadChatRows(rows pgx.Rows) []bool {
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

func (r *Repository) GetChatNumber(userID, adminID int64) (int, error) {
	var chatNumber int
	err := r.Pool.QueryRow(r.ctx, `SELECT chat_number FROM middleware.anonymous_chat WHERE admin_id = $1 AND user_id = $2`, adminID, userID).Scan(&chatNumber)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return 0, nil
		}
		return 0, errors.Wrap(err, "failed to get chat number")
	}

	return chatNumber, nil
}

func (r *Repository) AddMessageFromUser(message string, userID int64, chatNumber int) error {
	_, err := r.Pool.Exec(r.ctx, "INSERT INTO middleware.messages (user_message,chat_number,user_id) VALUES ($1,$2,$3)", message, chatNumber, userID)
	if err != nil {
		return errors.Wrap(err, "failed to add message from user")
	}

	return nil
}

func (r *Repository) AddPhotoMessageFromUser(photoMessage string, message string, userID int64, chatNumber int) error {
	_, err := r.Pool.Exec(r.ctx, "INSERT INTO middleware.messages (user_photo_message,user_message,chat_number,user_id) VALUES ($1,$2,$3,$4)", photoMessage, message, chatNumber, userID)
	if err != nil {
		return errors.Wrap(err, "failed to add message from user")
	}

	return nil
}

func (r *Repository) AddMessageFromAdmin(message string, adminID int64, chatNumber int) error {
	_, err := r.Pool.Exec(r.ctx, "INSERT INTO middleware.messages (admin_message,chat_number,admin_id) VALUES ($1,$2,$3)", message, chatNumber, adminID)
	if err != nil {
		return errors.Wrap(err, "failed to add message from admin")
	}

	return nil
}

func (r *Repository) AddPhotoMessageFromAdmin(photoMessage string, message string, adminID int64, chatNumber int) error {
	_, err := r.Pool.Exec(r.ctx, "INSERT INTO middleware.messages (admin_photo_message,admin_message,chat_number,admin_id) VALUES ($1,$2,$3,$4)", photoMessage, message, chatNumber, adminID)
	if err != nil {
		return errors.Wrap(err, "failed to add message from user")
	}

	return nil
}

func (r *Repository) GetChatNumberWhereLiveChat(adminID int64, chatLive bool) ([]int, error) {
	rows, err := r.Pool.Query(r.ctx, `SELECT chat_number FROM middleware.anonymous_chat WHERE admin_id = $1 AND chat_start = $2`, adminID, chatLive)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get chat number where live chat")
	}

	readRows, err := r.ReadRows(rows)
	if err != nil {
		return nil, err
	}

	rows.Close()
	return readRows, nil
}

func (r *Repository) ReadRows(rows pgx.Rows) ([]int, error) {
	var chatStarts []int
	var chatStart int

	for rows.Next() {
		err := rows.Scan(&chatStart)
		if err != nil {
			return nil, err
		}

		chatStarts = append(chatStarts, chatStart)
	}

	return chatStarts, nil
}

func (r *Repository) GetUserIDChat(chatNumber int) (int64, error) {
	var userID int64
	err := r.Pool.QueryRow(r.ctx, `SELECT user_id FROM middleware.anonymous_chat WHERE chat_number = $1`, chatNumber).Scan(&userID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get chat ids")
	}

	return userID, nil
}

func (r *Repository) StopChat(chatNumber int) error {
	_, err := r.Pool.Exec(r.ctx, `UPDATE middleware.anonymous_chat SET chat_start = $1 WHERE chat_number = $2`, false, chatNumber)
	if err != nil {
		return errors.Wrap(err, "failed to stop chat")
	}

	return nil
}

func (r *Repository) GetAdminAndUserID(chatNumber int) (int64, int64, error) {
	var userID, adminID int64
	err := r.Pool.QueryRow(r.ctx, `SELECT user_id, admin_id FROM middleware.anonymous_chat WHERE chat_number = $1`, chatNumber).Scan(&userID, &adminID)
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to get admin and user id")
	}

	return userID, adminID, nil
}

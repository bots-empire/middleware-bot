package repository

import (
	"context"
	model2 "github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"strings"
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
SELECT * FROM users 
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
	_, err := r.Pool.Exec(r.ctx, `INSERT INTO users VALUES ($1,$2);`, u.ID, false)
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
SELECT * FROM users
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
			&user.IsAdmin,
		); err != nil {
			return nil, errors.Wrap(err, model2.ErrScanSqlRow.Error())
		}

		users = append(users, user)
	}

	return users, nil
}

func (r *Repository) AddAdminIDsToDBIfNotExist(id int64) error {
	_, err := r.Pool.Exec(r.ctx, `INSERT INTO users VALUES ($1,$2)`, id, true)
	if err != nil {
		if strings.Contains(err.(*pgconn.PgError).Message, "duplicate key value violates unique constraint") {
			_, err := r.Pool.Exec(r.ctx, `UPDATE users SET is_admin = $1 WHERE id = $2`, true, id)
			if err != nil {
				return errors.Wrap(err, "failed to update is admin")
			}

			return nil
		}
		return errors.Wrap(err, "failed to insert admin id")
	}

	return nil
}

func (r *Repository) ChangeStartChat(adminID, userID int64, chatStart bool) error {
	_, err := r.Pool.Exec(r.ctx, `INSERT INTO anonymous_chat (admin_id, user_id, chat_start) 
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
	_, err := r.Pool.Exec(r.ctx, `UPDATE anonymous_chat SET chat_start = $1 WHERE chat_number = $2`, true, chatNumber)
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
FROM anonymous_chat 
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
	rows, err := r.Pool.Query(r.ctx, `SELECT chat_start FROM anonymous_bot WHERE user_id = $1`, userID)
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

func (r *Repository) MessageToUser(id int64) (int64, bool, error) {
	var userID int64
	var chatStart bool
	err := r.Pool.QueryRow(r.ctx, "SELECT user_id, chat_start FROM anonymous_chat WHERE admin_id = $1", &id).Scan(&userID, &chatStart)
	if err != nil {
		return 0, false, errors.Wrap(err, "failed to message to user")
	}

	return userID, chatStart, nil
}

func (r *Repository) MessageToAdmin(id int64) (int64, bool, error) {
	var adminID int64
	var chatStart bool
	err := r.Pool.QueryRow(r.ctx, "SELECT admin_id, chat_start FROM anonymous_chat WHERE user_id = $1", &id).Scan(&adminID, &chatStart)
	if err != nil {
		return 0, false, errors.Wrap(err, "failed to message to admin")
	}

	return adminID, chatStart, nil
}

func (r *Repository) GetChatNumber(userID, adminID int64) (int, error) {
	var chatNumber int
	err := r.Pool.QueryRow(r.ctx, `SELECT chat_number FROM anonymous_chat WHERE admin_id = $1, user_id = $2`, adminID, userID).Scan(&chatNumber)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get chat number")
	}

	return chatNumber, nil
}

func (r *Repository) AddMessageFromUser(message string, userID, adminID int64) error {
	_, err := r.Pool.Exec(r.ctx, "UPDATE anonymous_chat SET user_message = $1 WHERE user_id = $2 AND admin_id = $3", message, userID, adminID)
	if err != nil {
		return errors.Wrap(err, "failed to add message from user")
	}

	return nil
}

func (r *Repository) AddMessageFromAdmin(message string, userID, adminID int64) error {
	_, err := r.Pool.Exec(r.ctx, "UPDATE anonymous_chat SET admin_message = $1 WHERE user_id = $2 AND admin_id = $3", message, userID, adminID)
	if err != nil {
		return errors.Wrap(err, "failed to add message from admin")
	}

	return nil
}

func (r *Repository) GetAdmin(id int64) (bool, error) {
	var isAdmin bool
	err := r.Pool.QueryRow(r.ctx, "SELECT is_admin FROM users WHERE id = $1", id).Scan(&isAdmin)
	if err != nil {
		return false, errors.Wrap(err, "failed to get admin")
	}

	return isAdmin, nil
}

func (r *Repository) GetChatNumberWhereLiveChat(adminID int64, chatLive bool) ([]int, error) {
	rows, err := r.Pool.Query(r.ctx, `SELECT chat_number FROM anonymous_chat WHERE admin_id = $1 AND chat_start = $2`, adminID, chatLive)
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
	err := r.Pool.QueryRow(r.ctx, `SELECT user_id FROM anonymous_chat WHERE chat_number = $1`, chatNumber).Scan(&userID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get chat ids")
	}

	return userID, nil
}

func (r *Repository) StopChat(chatNumber int) error {
	_, err := r.Pool.Exec(r.ctx, `UPDATE anonymous_chat SET chat_start = $1 WHERE chat_number = $2`, false, chatNumber)
	if err != nil {
		return errors.Wrap(err, "failed to stop chat")
	}

	return nil
}

-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS middleware;

CREATE TABLE IF NOT EXISTS middleware.users
(
    id bigint UNIQUE
);

CREATE TABLE IF NOT EXISTS middleware.anonymous_chat
(
    chat_number SERIAL,
    admin_id    bigint,
    user_id     bigint,
    chat_start  bool
);

CREATE TABLE IF NOT EXISTS middleware.messages
(
    chat_number         int,
    admin_id            bigint,
    user_id             bigint,
    admin_message       text,
    admin_photo_message text,
    user_message        text,
    user_photo_message  text
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA middleware CASCADE;
-- +goose StatementEnd

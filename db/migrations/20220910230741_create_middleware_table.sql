-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS middleware;

CREATE TABLE IF NOT EXISTS middleware.users
(
    id        bigint UNIQUE,
    user_name text
);

CREATE TABLE IF NOT EXISTS middleware.anonymous_chat
(
    chat_number       SERIAL,
    admin_id          bigint,
    user_id           bigint,
    note              text,
    chat_start        bool,
    last_message_time timestamp
);

CREATE TABLE IF NOT EXISTS middleware.common_messages
(
    chat_number         int,
    admin_id            bigint,
    admin_message       text,
    admin_photo_message text,
    user_id             bigint,
    user_message        text,
    user_photo_message  text,
    message_time        timestamp
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA middleware CASCADE;
-- +goose StatementEnd

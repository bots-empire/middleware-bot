-- +goose Up
CREATE TABLE users
(
    id       bigint UNIQUE,
    is_admin bool
);

CREATE TABLE anonymous_chat
(
    chat_number SERIAL,
    admin_id bigint,
    user_id bigint,
    admin_message text,
    user_message text,
    chat_start bool
);

-- +goose Down
DROP TABLE users;

-- +migrate Up
ALTER TABLE users ADD COLUMN telegram_user_id INTEGER UNIQUE;
ALTER TABLE users ADD COLUMN telegram_chat_id INTEGER;

-- +migrate Down
ALTER TABLE users DROP COLUMN telegram_user_id;
ALTER TABLE users DROP COLUMN telegram_chat_id; 
-- +goose Up
-- +goose StatementBegin
REVOKE CREATE ON schema public FROM public;
CREATE SCHEMA IF NOT EXISTS users_api;
CREATE TABLE users_api.users (
    id TEXT PRIMARY KEY,
    email TEXT,
    email_confirmed BOOLEAN NOT NULL,
    email_confirmation_sent timestamptz,
    email_confirmation_key TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users_api.users;
DROP SCHEMA users_api;
-- +goose StatementEnd

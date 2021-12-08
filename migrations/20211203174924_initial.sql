-- +goose Up
-- +goose StatementBegin
REVOKE CREATE ON schema public FROM public;
CREATE SCHEMA IF NOT EXISTS users_api;
CREATE TABLE users_api.users (
    id uuid PRIMARY KEY,
    oidc_subject TEXT UNIQUE NOT NULL,
    joined timestamptz NOT NULL,
    email_address TEXT,
    email_confirmed BOOLEAN NOT NULL,
    email_confirmation_sent timestamptz,
    email_confirmation_key TEXT,
    eth_address TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA users_api CASCADE;
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin
CREATE TABLE users_api.users (
    id TEXT PRIMARY KEY,
    email_address TEXT,
    email_confirmed BOOLEAN NOT NULL,
    email_confirmation_sent timestamptz,
    email_confirmation_key TEXT,
    created_at timestamptz NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users_api.users;
-- +goose StatementEnd

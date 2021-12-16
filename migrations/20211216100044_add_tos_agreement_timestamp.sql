-- +goose Up
-- +goose StatementBegin
ALTER TABLE users_api.users ADD COLUMN agreed_tos_at timestamptz;
UPDATE users_api.users SET agreed_tos_at = NOW();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users_api.users DROP COLUMN agreed_tos_at;
-- +goose StatementEnd

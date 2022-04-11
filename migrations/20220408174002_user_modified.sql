-- +goose Up
-- +goose StatementBegin
SET search_path TO users_api, public;
ALTER TABLE users ADD COLUMN updated_at timestamptz NOT NULL DEFAULT current_timestamp;
UPDATE users SET updated_at = created_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SET search_path TO users_api, public;
ALTER TABLE users DROP COLUMN updated_at;
-- +goose StatementEnd

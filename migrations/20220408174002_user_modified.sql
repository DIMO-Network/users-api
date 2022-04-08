-- +goose Up
-- +goose StatementBegin
SET search_path TO users_api, public;
ALTER TABLE users ADD COLUMN updated_at timestamptz;
UPDATE users SET updated_at = created_at;
ALTER TABLE users ALTER COLUMN updated_at SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SET search_path TO users_api, public;
ALTER TABLE users DROP COLUMN updated_at;
-- +goose StatementEnd

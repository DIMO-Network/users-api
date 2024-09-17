-- +goose Up
-- +goose StatementBegin
SET search_path TO users_api, public;

ALTER TABLE users ADD COLUMN migrated_at timestamptz;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SET search_path TO users_api, public;

ALTER TABLE users DROP COLUMN migrated_at;
-- +goose StatementEnd

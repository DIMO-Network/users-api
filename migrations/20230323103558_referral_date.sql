-- +goose Up
-- +goose StatementBegin
SET search_path TO users_api, public;

ALTER TABLE users_api.users 
    ADD COLUMN referred_at timestamptz;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SET search_path TO users_api, public;

ALTER TABLE users_api.users 
    DROP COLUMN referred_at;
-- +goose StatementEnd

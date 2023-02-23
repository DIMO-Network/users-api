-- +goose Up
-- +goose StatementBegin
SET search_path TO users_api, public;

ALTER TABLE users ADD COLUMN in_app_wallet boolean NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SET search_path TO users_api, public;

ALTER TABLE users DROP COLUMN in_app_wallet;
-- +goose StatementEnd

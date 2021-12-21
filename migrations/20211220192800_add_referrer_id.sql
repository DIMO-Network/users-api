-- +goose Up
-- +goose StatementBegin
ALTER TABLE users_api.users ADD COLUMN referrer_id TEXT REFERENCES users_api.users(id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users_api.users DROP COLUMN referrer_id;
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin
ALTER TABLE users_api.users RENAME COLUMN email_confirmation_sent TO email_confirmation_sent_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users_api.users RENAME COLUMN email_confirmation_sent_at TO email_confirmation_sent;
-- +goose StatementEnd

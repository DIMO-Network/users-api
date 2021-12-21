-- +goose Up
-- +goose StatementBegin
ALTER TABLE users_api.users ADD UNIQUE (referral_code);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users_api.users DROP CONSTRAINT users_referral_code_key;
-- +goose StatementEnd

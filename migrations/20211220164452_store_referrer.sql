-- +goose Up
-- +goose StatementBegin
ALTER TABLE users_api.users ADD UNIQUE (referral_code);
ALTER TABLE users_api.users ADD COLUMN referred_by varchar(12) REFERENCES users_api.users(referral_code);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users_api.users DROP COLUMN referred_by;
ALTER TABLE users_api.users DROP CONSTRAINT users_referral_code_key;
-- +goose StatementEnd

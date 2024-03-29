-- +goose Up
-- +goose StatementBegin
SET search_path TO users_api, public;

ALTER TABLE users_api.users 
    ADD COLUMN referral_code CHAR(12),
    ADD COLUMN referred_by CHAR(12);

ALTER TABLE users_api.users ADD CONSTRAINT users_referral_code_key UNIQUE (referral_code);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SET search_path TO users_api, public;

ALTER TABLE users_api.users 
    DROP COLUMN referral_code,
    DROP COLUMN referred_by;
-- +goose StatementEnd

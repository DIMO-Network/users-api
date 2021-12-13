-- +goose Up
-- +goose StatementBegin
ALTER TABLE users_api.users ALTER COLUMN referral_code TYPE varchar(12) USING rtrim(referral_code);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users_api.users ALTER COLUMN referral_code TYPE char(12);
-- +goose StatementEnd

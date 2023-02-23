-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

ALTER TABLE users_api.users ADD COLUMN referral_code CHAR(12);
ALTER TABLE users_api.users ADD CONSTRAINT users_referral_code_key UNIQUE (referral_code);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

ALTER TABLE users_api.users DROP CONSTRAINT users_referral_code_key;
ALTER TABLE users_api.users DROP COLUMN referral_code;

-- +goose StatementEnd

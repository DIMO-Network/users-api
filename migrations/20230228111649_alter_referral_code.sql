-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

SET search_path TO users_api, public;

ALTER TABLE users_api.users 
    ALTER COLUMN referral_code TYPE varchar(12),
    ALTER COLUMN referred_by TYPE varchar(12);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

SET search_path TO users_api, public;

ALTER TABLE users_api.users 
    ALTER COLUMN referral_code TYPE CHAR(6),
    ALTER COLUMN referred_by TYPE CHAR(6);
-- +goose StatementEnd

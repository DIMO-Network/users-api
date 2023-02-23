-- +goose Up
-- +goose StatementBegin
SET search_path TO users_api, public;

DROP TABLE referrals;

ALTER TABLE users
    DROP COLUMN referral_code,
    DROP COLUMN referrer_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SET search_path TO users_api, public;
-- +goose StatementEnd

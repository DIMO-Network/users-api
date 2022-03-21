-- +goose Up
-- +goose StatementBegin
ALTER TABLE users_api.users ADD COLUMN ethereum_challenge TEXT;
ALTER TABLE users_api.users ADD COLUMN ethereum_challenge_sent timestamptz;
ALTER TABLE users_api.users ADD COLUMN ethereum_confirmed BOOLEAN;

UPDATE users_api.users SET ethereum_confirmed = true WHERE ethereum_address IS NOT NULL;
UPDATE users_api.users SET ethereum_confirmed = false WHERE ethereum_address IS NULL;
ALTER TABLE users_api.users ALTER COLUMN ethereum_confirmed SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users_api.users DROP COLUMN ethereum_challenge_sent;
ALTER TABLE users_api.users DROP COLUMN ethereum_challenge;
ALTER TABLE users_api.users DROP COLUMN ethereum_confirmed;
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin
ALTER TABLE users_api.users ADD COLUMN auth_provider_id TEXT;
UPDATE users_api.users SET auth_provider_id = 'google' WHERE ethereum_address IS NULL;
UPDATE users_api.users SET auth_provider_id = 'web3' WHERE ethereum_address IS NOT NULL;
ALTER TABLE users_api.users ALTER COLUMN auth_provider_id SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users_api.users DROP COLUMN auth_provider_id;
-- +goose StatementEnd

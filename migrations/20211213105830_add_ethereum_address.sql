-- +goose Up
-- +goose StatementBegin
ALTER TABLE users_api.users ADD COLUMN ethereum_address CHAR(42);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users_api.users DROP COLUMN ethereum_address;
-- +goose StatementEnd

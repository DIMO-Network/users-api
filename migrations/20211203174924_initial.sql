-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA users_api;
CREATE TABLE users_api.users (id TEXT PRIMARY KEY, email TEXT, verified BOOLEAN NOT NULL, verification_code TEXT);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users_api.users;
DROP SCHEMA users_api;
-- +goose StatementEnd

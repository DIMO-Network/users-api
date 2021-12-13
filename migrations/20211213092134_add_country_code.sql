-- +goose Up
-- +goose StatementBegin
ALTER TABLE users_api.users ADD COLUMN country_code CHAR(3);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users_api.users DROP COLUMN country_code;
-- +goose StatementEnd

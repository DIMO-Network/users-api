-- +goose Up
-- +goose StatementBegin
SET search_path TO users_api, public;

CREATE INDEX users_ethereum_address_idx ON users (ethereum_address);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SET search_path TO users_api, public;

DROP INDEX users_ethereum_address_idx;
-- +goose StatementEnd

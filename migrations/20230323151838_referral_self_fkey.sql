-- +goose Up
-- +goose StatementBegin
SET search_path TO users_api, public;

ALTER TABLE users DROP COLUMN referred_by;

ALTER TABLE users ADD COLUMN referring_user_id text 
    CONSTRAINT users_referring_user_id_fkey REFERENCES users(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SET search_path TO users_api, public;
-- +goose StatementEnd

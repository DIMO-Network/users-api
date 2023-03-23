-- +goose Up
-- +goose StatementBegin
SET search_path TO users_api, public;

ALTER TABLE users DROP COLUMN referred_by;

ALTER TABLE users_api.users
    ADD COLUMN referred_at timestamptz,
    ADD COLUMN referring_user_id text CONSTRAINT users_referring_user_id_fkey REFERENCES users(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SET search_path TO users_api, public;

ALTER TABLE users_api.users 
    DROP COLUMN referred_at
    DROP COLUMN referring_user_id;

ALTER TABLE users ADD COLUMN referred_by TYPE varchar(12);
-- +goose StatementEnd

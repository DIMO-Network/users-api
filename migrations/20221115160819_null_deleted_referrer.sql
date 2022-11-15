-- +goose Up
-- +goose StatementBegin
SET search_path TO users_api, public;

ALTER TABLE users
    DROP CONSTRAINT users_referrer_id_fkey,
    ADD CONSTRAINT users_referrer_id_fkey FOREIGN KEY (referrer_id) REFERENCES users(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SET search_path TO users_api, public;

ALTER TABLE users
    DROP CONSTRAINT users_referrer_id_fkey,
    ADD CONSTRAINT users_referrer_id_fkey FOREIGN KEY (referrer_id) REFERENCES users(id);
-- +goose StatementEnd

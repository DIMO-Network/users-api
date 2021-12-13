-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION pg_temp.random_string(int) RETURNS TEXT AS $$
DECLARE
    allowed_chars TEXT;
BEGIN
    SELECT '0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ' into allowed_chars;
    RETURN (
        SELECT string_agg(substring(allowed_chars, (floor(length(allowed_chars) * random()) + 1)::integer, 1), '')
        FROM generate_series(1, $1)
    );
END;
$$ language plpgsql;

ALTER TABLE users_api.users ADD COLUMN referral_code CHAR(12);
UPDATE users_api.users SET referral_code = pg_temp.random_string(8);
ALTER TABLE users_api.users ALTER COLUMN referral_code SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users_api.users DROP COLUMN referral_code;
-- +goose StatementEnd

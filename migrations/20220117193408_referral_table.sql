-- +goose Up
-- +goose StatementBegin
CREATE TABLE users_api.referrals (
    user_id TEXT NOT NULL,
    referred_user_id TEXT NOT NULL,
    vin CHAR(17) NOT NULL,
    created_at timestamptz NOT NULL,

    PRIMARY KEY (referred_user_id),
    -- Can't put a foreign key on referred_user_id because that user might be deleted, and we don't
    -- want to take away the referral in that case.
    CONSTRAINT referrals_user_id_fkey FOREIGN KEY (user_id) REFERENCES users_api.users(id) ON DELETE CASCADE,
    CONSTRAINT referrals_vin_key UNIQUE (vin)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users_api.referrals;
-- +goose StatementEnd

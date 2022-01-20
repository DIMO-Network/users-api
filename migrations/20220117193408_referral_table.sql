-- +goose Up
-- +goose StatementBegin
CREATE TABLE users_api.referrals (
    user_id TEXT NOT NULL,
    referred_user_id TEXT NOT NULL,
    vin CHAR(17) NOT NULL,
    created_at timestamptz NOT NULL, -- Will never update.

    PRIMARY KEY (vin),
    CONSTRAINT referrals_user_id_fkey FOREIGN KEY (user_id) REFERENCES users_api.users(id) ON DELETE CASCADE,
    -- Can't put a foreign key on referred_user_id because that user might later be deleted.
    -- We want to keep the referral, and by not nulling out referred_user_id we prevent him from
    -- re-registering with the same credentials and counting as a referral.
    CONSTRAINT referrals_referred_user_id_key UNIQUE (referred_user_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users_api.referrals;
-- +goose StatementEnd

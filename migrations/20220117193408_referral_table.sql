-- +goose Up
-- +goose StatementBegin
CREATE TABLE users_api.referrals (
    user_id TEXT NOT NULL,
    referred_user_id TEXT NOT NULL,
    vin CHAR(17) NOT NULL,

    PRIMARY KEY (user_id, referred_user_id),
    CONSTRAINT referrals_referred_user_id_fkey FOREIGN KEY (referred_user_id) REFERENCES users_api.users(id) ON DELETE CASCADE,
    CONSTRAINT referrals_referred_user_id_key UNIQUE (referred_user_id),
    CONSTRAINT referrals_vin_key UNIQUE (vin)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users_api.referrals;
-- +goose StatementEnd

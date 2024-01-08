-- +goose Up
-- +goose StatementBegin
SET search_path TO users_api, public;

BEGIN TRANSACTION;

SELECT  id,
        ethereum_address as eth
INTO    users_api.users_eth
FROM    users_api.users;

ALTER TABLE users_api.users
    ALTER COLUMN ehtereum_address TYPE bytea 
    USING NULL;

LOCK TABLE users_api.users IN ACCESS EXCLUSIVE MODE;

UPDATE users_api.users
SET ehtereum_address = decode(substr(eth, 3), 'hex')
FROM users_api.users_eth
WHERE users.id = users_eth.id;

DROP TABLE users_api.users_eth;

COMMIT TRANSACTION;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SET search_path TO users_api, public;

BEGIN TRANSACTION;

SELECT  id,
        ehtereum_address as eth
INTO    users_api.users_eth
FROM    users_api.users;

ALTER TABLE users_api.users
    ALTER COLUMN ehtereum_address TYPE TEXT 
    USING NULL;

LOCK TABLE users_api.users IN ACCESS EXCLUSIVE MODE;

UPDATE users_api.users
SET ehtereum_address = '0x' || encode(eth, 'hex')
FROM users_api.users_eth
WHERE users.id = users_eth.id;

DROP TABLE users_api.users_eth;

COMMIT TRANSACTION;
-- +goose StatementEnd
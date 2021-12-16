## Running the app

Make a copy of the settings file and edit as you see fit:
```sh
cp settings.sample.yaml settings.yaml
```
If you kept the default database and SMTP server settings then you'll want to start up the respective containers:
```sh
docker-compose up -d
```
This runs Postgres on port 5432; and [Mailhog](https://github.com/mailhog/MailHog) on port 1025 with a [web interface](http://localhost:8025) on port 8025.

With a fresh database you'll want to run the migrations:
```sh
go run ./cmd/users-api migrate
```
Finally, running the app is simple:
```sh
go run ./cmd/users-api
```

## Developer notes

Check linting with

```
golangci-lint run -E prealloc -E revive -E goimports -E deadcode -E errcheck -E gosimple -E govet -E ineffassign -E staticcheck -E structcheck -E typecheck -E unused -E varcheck --timeout=5m
```

## Database modifications

Create a new Goose migration file:
```
goose -dir migrations postgres "host=localhost port=5432 user=dimo password=dimo dbname=users_api sslmode=disable" create MIGRATION_TITLE sql
```
This will create a file in the `migrations` folder named something like `TIMESTAMP_MIGRATION_TITLE.sql`. Edit this with your new innovations. To run the migrations:
```
go run ./cmd/users-api migrate
```
And then to generate the models:
```
sqlboiler psql --no-tests --wipe
```

## Endpoints

`GET /user`

`200`

Response
```json
{
    "id": "CioweGNGQkFEZTY5MjgzMkFGYTFlOTM2OUM2RUE3MjQ3YjVEZTc5MTI5NjQSBHdlYjM",
    "emailAddress": "joe@dimo.zone",
    "emailVerified": true,
    "createdAt": "2021-12-09T00:57:49.674985Z",
    "countryCode": null,
    "ethereumAddress": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F"
}
```

`PUT /user`

JSON body
```json
{
    "emailAddress": "eric@dimo.zone",
    "countryCode": "PER"
}
```

`200`

Response
```json
{
    "id": "CioweGNGQkFEZTY5MjgzMkFGYTFlOTM2OUM2RUE3MjQ3YjVEZTc5MTI5NjQSBHdlYjM",
    "emailAddress": "eric@dimo.zone",
    "emailVerified": false,
    "createdAt": "2021-12-09T00:57:49.674985Z",
    "countryCode": "PER",
    "ethereumAddress": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F"
}
```

`POST /user/send-confirmation-email`

`POST /user/confirm-email`

JSON body

```json
{
    "key": "60412984"
}
```

`200`

`400`

This can fail for a few reasons:

- We already confirmed the email
- We never sent a confirmation email for the current candidate email address
- The confirmation key expired; the default timeout for this is 15 minutes
- The submitted key does not match the one we've stored

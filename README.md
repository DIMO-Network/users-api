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

## Database modifications

Create a new Goose migration file:  
```
goose -dir migrations postgres "host=localhost port=5432 user=postgres password= dbname=postgres sslmode=disable" create MIGRATION_TITLE sql
```
This will create a file in the `migrations` folder named something like `TIMESTAMP_MIGRATION_TITLE.sql`.

## Endpoints

`GET /user`

`200`

Response
```json
{
    "id": "CioweGNGQkFEZTY5MjgzMkFGYTFlOTM2OUM2RUE3MjQ3YjVEZTc5MTI5NjQSBHdlYjM",
    "email_address": "joe@dimo.zone",
    "email_verified": true,
    "created_at": "2021-12-09T00:57:49.674985Z"
}
```

`PUT /user`

JSON body
```json
{
    "email_address": "eric@dimo.zone"
}
```

`200`

Response
```json
{
    "id": "CioweGNGQkFEZTY5MjgzMkFGYTFlOTM2OUM2RUE3MjQ3YjVEZTc5MTI5NjQSBHdlYjM",
    "email_address": "eric@dimo.zone",
    "email_verified": false,
    "created_at": "2021-12-09T00:57:49.674985Z"
}
```

`POST /send-confirmation-email`

`POST /confirm-email`

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

### Running the app

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

### Endpoints

`GET /user`

`200`

Response
```json
{
    "id": "de5817c0-57a6-11ec-bf63-0242ac130002",
    "email": "joe@dimo.zone",
    "email_verified": true
}
```

`PUT /user`

JSON body
```json
{
    "email": "eric@dimo.zone"
}
```

`200`

Response
Response
```json
{
    "id": "de5817c0-57a6-11ec-bf63-0242ac130002",
    "email": "eric@dimo.zone",
    "email_verified": false
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

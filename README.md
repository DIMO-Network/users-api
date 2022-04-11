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
golangci-lint run
```

Update OpenAPI documentation with 
```
swag init --generalInfo cmd/users-api/main.go --generatedTime true
```

## Database modifications

Create a new Goose migration file:
```
goose -dir migrations create MIGRATION_TITLE sql
```
This will create a file in the `migrations` folder named something like `TIMESTAMP_MIGRATION_TITLE.sql`. Edit this with your new innovations. To run the migrations:
```
go run ./cmd/users-api migrate
```
And then to generate the models:
```
sqlboiler psql --no-tests --wipe
```

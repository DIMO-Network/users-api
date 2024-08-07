package database

import (
	"context"
	"database/sql"

	"github.com/DIMO-Network/shared/db"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
)

func MigrateDatabase(ctx context.Context, _ zerolog.Logger, settings *db.Settings, command, migrationsDir string) error {
	db, err := sql.Open("postgres", settings.BuildConnectionString(true))
	if err != nil {
		return err
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		return err
	}

	if command == "" {
		command = "up"
	}

	_, err = db.Exec("CREATE SCHEMA IF NOT EXISTS users_api;")
	if err != nil {
		return err
	}
	goose.SetTableName("users_api.migrations")
	return goose.RunContext(ctx, command, db, migrationsDir)
}

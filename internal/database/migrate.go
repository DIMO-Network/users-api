package database

import (
	"database/sql"
	"fmt"

	"github.com/DIMO-Network/shared/db"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"

	_ "github.com/lib/pq"
)

func MigrateDatabase(logger zerolog.Logger, settings *db.Settings, command, schemaName string) error {
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

	_, err = db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", schemaName))
	if err != nil {
		return err
	}
	goose.SetTableName(fmt.Sprintf("%s.migrations", schemaName))
	return goose.Run(command, db, "migrations")
}

package main

import (
	"database/sql"
	"fmt"

	"github.com/DIMO-Network/shared/db"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"

	_ "github.com/lib/pq"
)

func migrateDatabase(logger zerolog.Logger, settings *db.Settings, command, schemaName string) {
	var db *sql.DB
	// setup database
	db, err := sql.Open("postgres", settings.BuildConnectionString(true))
	defer func() {
		if err := db.Close(); err != nil {
			logger.Fatal().Msgf("goose: failed to close DB: %v\n", err)
		}
	}()
	if err != nil {
		logger.Fatal().Msgf("failed to open db connection: %v\n", err)
	}
	if err = db.Ping(); err != nil {
		logger.Fatal().Msgf("failed to ping db: %v\n", err)
	}

	// set default
	if command == "" {
		command = "up"
	}
	// must create schema so that can set migrations table to that schema
	_, err = db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", schemaName))
	if err != nil {
		logger.Fatal().Err(err).Msgf("could not create schema, %s", schemaName)
	}
	goose.SetTableName(fmt.Sprintf("%s.migrations", schemaName))
	if err := goose.Run(command, db, "migrations"); err != nil {
		logger.Fatal().Msgf("failed to apply go code migrations: %v\n", err)
	}
	// if we add any code migrations import _ "github.com/DIMO-Network/users-api/migrations" // migrations won't work without this
}

package main

import (
	"context"

	"github.com/DIMO-INC/users-api/internal/config"
	"github.com/DIMO-INC/users-api/internal/controllers"
	"github.com/DIMO-INC/users-api/internal/database"
	"github.com/DIMO-INC/users-api/internal/services"
	"github.com/DIMO-INC/users-api/models"
	"github.com/rs/zerolog"
)

func generateEvents(logger zerolog.Logger, settings *config.Settings, dbs func() *database.DBReaderWriter, eventService *services.EventService) {
	ctx := context.Background()
	users, err := models.Users().All(ctx, dbs().Reader)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to retrieve all users for event generation")
	}
	for _, user := range users {
		method := "google"
		if user.EthereumAddress.Valid {
			// No other way to enter this at the moment
			method = "web3"
		}
		err = eventService.Emit(
			&services.Event{
				Type:    controllers.UserCreationEventType,
				Subject: user.ID,
				Source:  "users-api",
				Data: controllers.UserCreationEventData{
					Timestamp: user.CreatedAt,
					UserID:    user.ID,
					Method:    method,
				},
			},
		)
		if err != nil {
			logger.Err(err).Msgf("Failed to emit creation event for user %s", user.ID)
		}
	}
}

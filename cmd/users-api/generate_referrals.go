package main

import (
	"context"

	"github.com/DIMO-INC/users-api/internal/config"
	"github.com/DIMO-INC/users-api/internal/database"
	"github.com/DIMO-INC/users-api/internal/services"
	"github.com/DIMO-INC/users-api/models"
	"github.com/rs/zerolog"
)

func generateReferrals(logger *zerolog.Logger, settings *config.Settings, pdb database.DbStore, eventService *services.EventService) {
	ctx := context.Background()
	referrals, err := models.Referrals().All(ctx, pdb.DBS().Reader)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to retrieve all users for event generation")
	}
	for _, referral := range referrals {
		err = eventService.Emit(&services.Event{
			Type:    services.ReferralCompleteEventType,
			Subject: referral.UserID,
			Source:  "users-api",
			Data: services.ReferralEventData{
				Timestamp: referral.CreatedAt,
				UserID:    referral.UserID,
			},
		})
		if err != nil {
			logger.Err(err).Msgf("Failed to emit referral for user %s referring user %s", referral.UserID, referral.ReferredUserID)
		}
	}
}

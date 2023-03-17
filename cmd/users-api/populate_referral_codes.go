package main

import (
	"context"
	"fmt"

	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/users-api/internal/config"
	c "github.com/DIMO-Network/users-api/internal/controllers"
	"github.com/DIMO-Network/users-api/models"
	"github.com/rs/zerolog"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

type populateReferralCodeCmd struct {
	Settings *config.Settings
	dbs      db.Store
	log      *zerolog.Logger
	ctx      context.Context
}

func (p populateReferralCodeCmd) Execute() {
	logger := p.log.With().Str("cmd", "populateReferralCodeCmd").Logger()
	uc := c.NewUserController(p.Settings, p.dbs, nil, p.log)

	users, err := models.Users(
		models.UserWhere.EthereumAddress.IsNotNull(),
		qm.Where(fmt.Sprintf("%s = '' OR %s IS NULL", models.UserColumns.ReferralCode, models.UserColumns.ReferralCode)),
	).All(p.ctx, p.dbs.DBS().Reader)
	if err != nil {
		logger.Err(err).Msg("Failed to fetch users")
	}

	if len(users) == 0 {
		logger.Info().Msg("Could not find any users without referral codes")
	}

	for i, u := range users {
		referralCode, err := uc.GenerateReferralCode(p.ctx)
		if err != nil {
			logger.Err(err).Msgf("Failed to generate referral code for user - %s", u.ID)
		}

		u.ReferralCode = null.StringFrom(referralCode)

		_, err = u.Update(p.ctx, p.dbs.DBS().Writer, boil.Infer())
		if err != nil {
			logger.Err(err).Msgf("Failed to update referral code for user - %s", u.ID)
		}

		logger.Info().Str("userId", u.ID).Int("current", i).Int("total", len(users)).Msgf("referral code created for user successfully!")
	}
}

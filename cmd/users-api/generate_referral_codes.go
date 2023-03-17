package main

import (
	"context"

	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/users-api/internal/config"
	"github.com/DIMO-Network/users-api/internal/controllers"
	"github.com/DIMO-Network/users-api/models"
	"github.com/rs/zerolog"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type generateReferralCodeCmd struct {
	Settings *config.Settings
	dbs      db.Store
	log      *zerolog.Logger
	uc       controllers.UserController
}

func (p *generateReferralCodeCmd) Execute(ctx context.Context) error {
	logger := p.log.With().Str("cmd", "generate-referral-codes").Logger()

	p.uc = controllers.NewUserController(p.Settings, p.dbs, nil, p.log)

	users, err := models.Users(
		models.UserWhere.EthereumAddress.IsNotNull(),
		models.UserWhere.EthereumConfirmed.EQ(true),
		models.UserWhere.ReferralCode.IsNull(),
	).All(ctx, p.dbs.DBS().Reader)
	if err != nil {
		return err
	}

	var success, failure int

	for _, u := range users {
		if err := p.forUser(ctx, u.ID); err != nil {
			logger.Err(err).Str("userId", u.ID).Msg("Failed to set referral code for user.")
			failure++
			continue
		}
		success++
	}

	logger.Info().Msgf("Created referral codes for %d users. Failed for %d.", success, failure)

	return nil
}

func (p *generateReferralCodeCmd) forUser(ctx context.Context, id string) error {
	code, err := p.uc.GenerateReferralCode(ctx)
	if err != nil {
		return err
	}

	u := models.User{ID: id, ReferralCode: null.StringFrom(code)}

	if _, err := u.Update(ctx, p.dbs.DBS().Writer, boil.Whitelist(models.UserColumns.ReferralCode)); err != nil {
		return err
	}

	return nil
}

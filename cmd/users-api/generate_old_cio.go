package main

import (
	"context"

	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/users-api/internal/config"
	"github.com/DIMO-Network/users-api/models"
	analytics "github.com/customerio/cdp-analytics-go"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
)

type generateCioCmd struct {
	Settings *config.Settings
	dbs      db.Store
	log      *zerolog.Logger
	cio      analytics.Client
}

func (p *generateCioCmd) Execute(ctx context.Context) error {
	users, err := models.Users(
		models.UserWhere.EthereumConfirmed.EQ(true),
		models.UserWhere.EmailConfirmed.EQ(true),
	).All(ctx, p.dbs.DBS().Reader)
	if err != nil {
		return err
	}

	for _, user := range users {
		if !user.EthereumAddress.Valid || !user.EmailAddress.Valid {
			continue
		}

		addr := common.BytesToAddress(user.EthereumAddress.Bytes)

		typ := "eoa"
		if user.InAppWallet {
			typ = "in_app"
		}

		err := p.cio.Enqueue(
			analytics.Identify{
				UserId: user.EmailAddress.String,
				Traits: analytics.NewTraits().
					Set("ethereum_wallet_address", addr.Hex()).
					Set("legacy_wallet_address", addr.Hex()).
					Set("legacy_wallet_address_type", typ),
			},
		)
		if err != nil {
			return err
		}
	}

	return p.cio.Close()
}

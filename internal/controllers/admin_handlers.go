package controllers

import (
	"math/big"

	"github.com/DIMO-Network/users-api/internal/controllers/contracts"
	"github.com/DIMO-Network/users-api/models"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"
)

// GetUser godoc
// @Summary Get attributes for the authenticated user. If multiple records for the same user, gets the one with the email confirmed.
// @Produce json
// @Param checkEmailRequest body controllers.CheckEmailRequest true "Specify the email to check."
// @Success 200 {object} controllers.CheckEmailResponse
// @Failure 00 {object} controllers.ErrorResponse
// @Failure 500 {object} controllers.ErrorResponse
// @Router /v1/check-email [post]
func (d *UserController) CheckEmail(c *fiber.Ctx) error {
	var cer CheckEmailRequest

	if err := c.BodyParser(&cer); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Couldn't parse body.")
	}

	users, err := models.Users(
		models.UserWhere.EmailAddress.EQ(null.StringFrom(cer.Address)),
		models.UserWhere.EmailConfirmed.EQ(true),
		models.UserWhere.EthereumConfirmed.EQ(true),
	).All(c.Context(), d.dbs.DBS().Reader)
	if err != nil {
		return err
	}

	usedInApp, usedExternal := 0, 0

	client, err := ethclient.Dial(d.Settings.MainRPCURL)
	if err != nil {
		return err
	}

	ad, _ := contracts.NewMultiPrivilege(common.HexToAddress(d.Settings.ADNFTAddr), client)
	v, _ := contracts.NewMultiPrivilege(common.HexToAddress(d.Settings.VehicleNFTAddr), client)
	tk, _ := contracts.NewToken(common.HexToAddress(d.Settings.TokenAddr), client)

	for _, user := range users {
		if !user.EthereumAddress.Valid || len(user.EthereumAddress.Bytes) != common.AddressLength {
			d.log.Warn().Msg("User %s is marked as having a confirmed Ethereum address, but the address is invalid.")
			continue
		}

		addr := common.BytesToAddress(user.EthereumAddress.Bytes)

		zero := big.NewInt(0)

		used, err := func() (bool, error) {
			if adBal, err := ad.BalanceOf(nil, addr); err != nil {
				return false, err
			} else if adBal.Cmp(zero) > 0 {
				return true, nil
			}

			if vBal, err := v.BalanceOf(nil, addr); err != nil {
				return false, err
			} else if vBal.Cmp(zero) > 0 {
				return true, nil
			}

			if tkBal, err := tk.BalanceOf(nil, addr); err != nil {
				return false, err
			} else if tkBal.Cmp(zero) > 0 {
				return true, nil
			}

			return false, nil
		}()
		if err != nil {
			return err
		}
		if !used {
			continue
		}

		if user.InAppWallet {
			usedInApp++
		} else {
			usedExternal++
		}
	}

	return c.JSON(CheckEmailResponse{
		InUse: usedInApp > 0 || usedExternal > 0,
		Wallets: struct {
			External int `json:"external"`
			InApp    int `json:"inApp"`
		}{
			External: usedExternal,
			InApp:    usedInApp,
		},
	})
}

type CheckEmailRequest struct {
	// Address is the email address to check. Must be confirmed.
	Address string `json:"address" example:"thaler@a16z.com"`
}

type CheckEmailResponse struct {
	// InUse specifies whether the email is attached to a DIMO user.
	InUse bool `json:"inUse"`

	Wallets struct {
		External int `json:"external"`
		InApp    int `json:"inApp"`
	}
}

package controllers

import (
	"fmt"
	"math/big"

	"github.com/DIMO-Network/users-api/internal/controllers/contracts"
	"github.com/DIMO-Network/users-api/models"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"
)

var zero = big.NewInt(0)

func nonZero(x *big.Int) bool {
	return x.Cmp(zero) != 0
}

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
		models.UserWhere.EthereumAddress.IsNotNull(),
	).All(c.Context(), d.dbs.DBS().Reader)
	if err != nil {
		return err
	}

	// TODO(elffjs): Don't do this.
	client, err := ethclient.Dial(d.Settings.MainRPCURL)
	if err != nil {
		return err
	}

	ad, _ := contracts.NewMultiPrivilege(common.HexToAddress(d.Settings.ADNFTAddr), client)
	v, _ := contracts.NewMultiPrivilege(common.HexToAddress(d.Settings.VehicleNFTAddr), client)
	tok, _ := contracts.NewToken(common.HexToAddress(d.Settings.TokenAddr), client)

	addrsInAppStatus := make(map[common.Address]bool)

	for _, user := range users {
		if len(user.EthereumAddress.Bytes) != common.AddressLength {
			d.log.Warn().Msg("User %s is marked as having a confirmed Ethereum address, but the address is invalid.")
			continue
		}

		addr := common.BytesToAddress(user.EthereumAddress.Bytes)

		if _, ok := addrsInAppStatus[addr]; ok {
			if user.InAppWallet {
				addrsInAppStatus[addr] = true
			}
		} else {
			addrsInAppStatus[addr] = user.InAppWallet
		}
	}

	usedInApp, usedExternal := 0, 0

	for addr, inApp := range addrsInAppStatus {
		used, err := func() (bool, error) {
			if vBal, err := v.BalanceOf(nil, addr); err != nil {
				return false, err
			} else if nonZero(vBal) {
				return true, nil
			}

			if adBal, err := ad.BalanceOf(nil, addr); err != nil {
				return false, err
			} else if nonZero(adBal) {
				return true, nil
			}

			if inApp {
				if tokBal, err := tok.BalanceOf(nil, addr); err != nil {
					return false, err
				} else if nonZero(tokBal) {
					return true, nil
				}
			}

			return false, nil
		}()
		if err != nil {
			return fmt.Errorf("error checking chain: %w", err)
		}

		if used {
			if inApp {
				usedInApp++
			} else {
				usedExternal++
			}
		}
	}

	return c.JSON(CheckEmailResponse{
		InUse: usedInApp+usedExternal > 0,
		Wallets: CheckWallets{
			External: usedExternal,
			InApp:    usedInApp,
		},
	})
}

type CheckEmailRequest struct {
	// Address is the email address to check. Must be confirmed.
	Address string `json:"address" example:"thaler@a16z.com"`
}

type CheckWallets struct {
	External int `json:"external"`
	InApp    int `json:"inApp"`
}

type CheckEmailResponse struct {
	// InUse specifies whether the email is attached to a DIMO user.
	InUse   bool         `json:"inUse"`
	Wallets CheckWallets `json:"wallets"`
}

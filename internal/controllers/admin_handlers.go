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

	type knownInfo struct {
		HasNFTs        bool
		ConfirmedInApp bool
	}

	ad, _ := contracts.NewMultiPrivilege(common.HexToAddress(d.Settings.ADNFTAddr), client)
	v, _ := contracts.NewMultiPrivilege(common.HexToAddress(d.Settings.VehicleNFTAddr), client)

	addrInfos := make(map[common.Address]*knownInfo)

	for _, user := range users {
		if len(user.EthereumAddress.Bytes) != common.AddressLength {
			d.log.Warn().Msg("User %s is marked as having a confirmed Ethereum address, but the address is invalid.")
			continue
		}

		addr := common.BytesToAddress(user.EthereumAddress.Bytes)

		if _, ok := addrInfos[addr]; !ok {
			// Check the chain.
			used, err := func() (bool, error) {
				if vBal, err := v.BalanceOf(nil, addr); err != nil {
					return false, err
				} else if nonZero(vBal) {
					return true, nil
				}

				if adBal, err := ad.BalanceOf(nil, addr); err != nil {
					return false, err
				} else {
					return nonZero(adBal), nil
				}
			}()
			if err != nil {
				return err
			}

			addrInfos[addr] = &knownInfo{
				HasNFTs: used,
			}
		}

		if user.InAppWallet {
			addrInfos[addr].ConfirmedInApp = true
		}
	}

	usedInApp, usedExternal := 0, 0
	for _, info := range addrInfos {
		if info.HasNFTs {
			if info.ConfirmedInApp {
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

package controllers

import (
	"fmt"

	"github.com/DIMO-INC/users-api/internal/config"
	"github.com/DIMO-INC/users-api/internal/database"
	"github.com/DIMO-INC/users-api/models"
	"github.com/customerio/go-customerio/v3"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

type CustomerIOController struct {
	DBS    func() *database.DBReaderWriter
	client *customerio.CustomerIO
	log    *zerolog.Logger
}

func NewCustomerIOController(settings *config.Settings, dbs func() *database.DBReaderWriter, logger *zerolog.Logger) CustomerIOController {
	return CustomerIOController{
		DBS: dbs,
		log: logger,
		client: customerio.NewTrackClient(
			settings.CIOSiteID,
			settings.CIOApiKey,
			customerio.WithRegion(customerio.RegionUS),
		),
	}
}

func (d *CustomerIOController) Track(c *fiber.Ctx) error {
	userID := getUserID(c)
	var req struct {
		Params map[string]interface{} `json:"params"`
	}
	if err := c.BodyParser(&req); err != nil {
		return errorResponseHandler(c, err, fiber.StatusBadRequest)
	}
	rawName, ok := req.Params["name"]
	if !ok {
		return errorResponseHandler(c, fmt.Errorf("couldn't find key params.name"), fiber.StatusBadRequest)
	}
	name, ok := rawName.(string)
	if !ok {
		return errorResponseHandler(c, fmt.Errorf("params.name should be a string"), fiber.StatusBadRequest)
	}
	if name == "referral_user" || name == "dev-referral_user" {
		iCode, ok := req.Params["referralCode"]
		if !ok {
			return errorResponseHandler(c, fmt.Errorf("referral_user event without params.referralCode"), fiber.StatusBadRequest)
		}
		code, ok := iCode.(string)
		if !ok {
			return errorResponseHandler(c, fmt.Errorf("params.referralCode should be a string"), fiber.StatusBadRequest)
		}

		err := d.setReferrer(c, userID, code)
		if err != nil {
			// Log, but continue and still forward to Customer.io
			d.log.Error().Err(err).Msgf("Failed to set referrer for user %s using code %s", userID, code)
		}
	}
	if err := d.client.Track(userID, name, req.Params); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	return c.JSON(fiber.Map{"success": true})
}

func (d *CustomerIOController) setReferrer(c *fiber.Ctx, userID, code string) (err error) {
	tx, err := d.DBS().Writer.BeginTx(c.Context(), nil)
	if err != nil {
		return
	}
	defer tx.Rollback() //nolint

	referrer, err := models.Users(models.UserWhere.ReferralCode.EQ(code)).One(c.Context(), tx)
	if err != nil {
		return
	}

	if referrer.ID == userID {
		return fmt.Errorf("user %s sent a self-referral", userID)
	}

	user, err := models.FindUser(c.Context(), tx, userID)
	if err != nil {
		return
	}

	err = user.SetReferrer(c.Context(), tx, false, referrer)
	if err != nil {
		return
	}

	err = tx.Commit()
	return
}

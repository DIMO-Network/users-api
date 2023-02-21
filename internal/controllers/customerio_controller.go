package controllers

import (
	"fmt"

	"github.com/DIMO-Network/users-api/internal/config"
	"github.com/DIMO-Network/users-api/internal/database"
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

	if err := d.client.Track(userID, name, req.Params); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	return c.JSON(fiber.Map{"success": true})
}

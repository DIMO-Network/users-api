package controllers

import (
	"fmt"

	"github.com/DIMO-INC/users-api/internal/config"
	"github.com/customerio/go-customerio/v3"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

type CustomerIOController struct {
	client *customerio.CustomerIO
}

func NewCustomerIOController(settings *config.Settings, logger *zerolog.Logger) CustomerIOController {
	return CustomerIOController{
		client: customerio.NewTrackClient(
			settings.CustomerIOSiteID,
			settings.CustomerIOApiKey,
			customerio.WithRegion(customerio.RegionUS),
		),
	}
}

func (d *CustomerIOController) Track(c *fiber.Ctx) error {
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
	if err := d.client.Track(getUserID(c), name, req.Params); err != nil {
		return errorResponseHandler(c, fmt.Errorf("failed Customer.io request"), fiber.StatusInternalServerError)
	}
	return c.JSON(fiber.Map{"success": true})
}

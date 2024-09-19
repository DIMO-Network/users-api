package controllers

import (
	"github.com/DIMO-Network/users-api/models"
	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"
)

// GetUser godoc
// @Summary Get attributes for the authenticated user. If multiple records for the same user, gets the one with the email confirmed.
// @Produce json
// @Param checkEmailRequest body controllers.CheckEmailRequest true "Specify the email to check."
// @Success 200 {object} controllers.CheckEmailResponse
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 500 {object} controllers.ErrorResponse
// @Router /v1/check-email [get]
func (d *UserController) CheckEmail(c *fiber.Ctx) error {
	var cer CheckEmailRequest

	if err := c.BodyParser(&cer); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Couldn't parse body.")
	}

	exists, err := models.Users(
		models.UserWhere.EmailAddress.EQ(null.StringFrom(cer.Address)),
		models.UserWhere.EmailConfirmed.EQ(true),
	).Exists(c.Context(), d.dbs.DBS().Reader)
	if err != nil {
		return err
	}

	return c.JSON(CheckEmailResponse{EmailInUse: exists})
}

type CheckEmailRequest struct {
	// Address is the email address to check. Must be confirmed.
	Address string `json:"address"`
}

type CheckEmailResponse struct {
	// EmailInUse specifies whether the email is attached to a DIMO user.
	EmailInUse bool `json:"emailInUse"`
}

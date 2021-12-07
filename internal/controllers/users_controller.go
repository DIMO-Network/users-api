package controllers

import (
	"context"
	"database/sql"
	"fmt"
	"net/smtp"

	"github.com/DIMO-INC/users-api/internal/config"
	"github.com/DIMO-INC/users-api/internal/database"
	"github.com/DIMO-INC/users-api/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type UsersController struct {
	Settings *config.Settings
	DBS      func() *database.DBReaderWriter
	log      *zerolog.Logger
}

func NewUsersController(settings *config.Settings, dbs func() *database.DBReaderWriter, logger *zerolog.Logger) UsersController {
	return UsersController{
		Settings: settings,
		DBS:      dbs,
		log:      logger,
	}
}

type userResponse struct {
	ID            string      `json:"id"`
	Email         null.String `json:"email"`
	EmailVerified bool        `json:"email_verified"`
}

type userRequest struct {
	Email null.String `json:"email"`
}

func (d *UsersController) getOrCreateUser(userID string, ctx context.Context) (user *models.User, err error) {
	tx, err := d.DBS().Writer.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	user, err = models.FindUser(ctx, tx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			var newUser models.User
			newUser.ID = userID
			err = newUser.Insert(ctx, tx, boil.Infer())
			if err != nil {
				tx.Rollback()
				return
			}
			tx.Commit()
			return &newUser, nil
		}
		tx.Rollback()
		return
	}
	tx.Commit()
	return
}

func (d *UsersController) GetUser(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)

	user, err := d.getOrCreateUser(userID, c.Context())
	if err != nil {
		panic(err)
	}
	return c.JSON(formatUser(user))
}

func formatUser(user *models.User) *userResponse {
	return &userResponse{ID: user.ID, Email: user.Email, EmailVerified: user.Verified}
}

func (d *UsersController) UpdateUser(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)

	user, err := d.getOrCreateUser(userID, c.Context())
	if err != nil {
		return err
	}

	var body userRequest
	c.BodyParser(&body)

	if body.Email != user.Email {
		user.Email = body.Email
		user.Verified = false
		user.Update(c.Context(), d.DBS().Writer, boil.Infer())
	}

	return c.JSON(formatUser(user))
}

func (d *UsersController) SendVerificationEmail(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)

	user, err := d.getOrCreateUser(userID, c.Context())
	auth := smtp.PlainAuth("", d.Settings.EmailUsername, d.Settings.EmailPassword, d.Settings.EmailFrom)
	addr := fmt.Sprintf("%s:%s", d.Settings.EmailHost, d.Settings.EmailPort)
	err = smtp.SendMail(addr, auth, d.Settings.EmailFrom, []string{user.Email.String}, []byte{})
	if err != nil {
		return err
	}
	return nil
}

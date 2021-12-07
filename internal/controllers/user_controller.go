package controllers

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net/smtp"
	"time"

	"github.com/DIMO-INC/users-api/internal/config"
	"github.com/DIMO-INC/users-api/internal/database"
	"github.com/DIMO-INC/users-api/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type UserController struct {
	Settings *config.Settings
	DBS      func() *database.DBReaderWriter
	log      *zerolog.Logger
}

func NewUserController(settings *config.Settings, dbs func() *database.DBReaderWriter, logger *zerolog.Logger) UserController {
	return UserController{
		Settings: settings,
		DBS:      dbs,
		log:      logger,
	}
}

type userResponse struct {
	ID             string      `json:"id"`
	Email          null.String `json:"email"`
	EmailConfirmed bool        `json:"email_verified"`
}

type userRequest struct {
	Email null.String `json:"email"`
}

func (d *UserController) getOrCreateUser(userID string, ctx context.Context) (user *models.User, err error) {
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

func (d *UserController) GetUser(c *fiber.Ctx) error {
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
	return &userResponse{ID: user.ID, Email: user.Email, EmailConfirmed: user.EmailConfirmed}
}

func (d *UserController) UpdateUser(c *fiber.Ctx) error {
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
		user.EmailConfirmed = false
		user.Update(c.Context(), d.DBS().Writer, boil.Infer())
	}

	return c.JSON(formatUser(user))
}

var digits = []rune("0123456789")

func generateConfirmationKey() string {
	o := make([]rune, 8)
	for i := range o {
		o[i] = digits[rand.Intn(10)]
	}
	return string(o)
}

func (d *UserController) SendConfirmationEmail(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)

	user, err := models.FindUser(c.Context(), d.DBS().Reader, userID)
	if err != nil {
		return nil
	}
	if !user.Email.Valid {
		return nil
	}

	if user.EmailConfirmed {
		return nil
	}

	key := generateConfirmationKey()
	user.EmailConfirmationKey = null.StringFrom(key)
	user.EmailConfirmationSent = null.TimeFrom(time.Now())
	user.Update(c.Context(), d.DBS().Writer, boil.Infer())

	auth := smtp.PlainAuth("", d.Settings.EmailUsername, d.Settings.EmailPassword, d.Settings.EmailFrom)
	addr := fmt.Sprintf("%s:%s", d.Settings.EmailHost, d.Settings.EmailPort)
	msg := []byte("From: DIMO Mailer <mailer@dimo.zone>\r\n" +
		"To: " + user.Email.String + "\r\n" +
		"Subject: DIMO email confirmation\r\n" +
		"\r\n" +
		"Your email confirmation code is\r\n" +
		"\r\n" +
		key + "\r\n")
	err = smtp.SendMail(addr, auth, d.Settings.EmailFrom, []string{user.Email.String}, msg)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	var err error
	allowedLateness, err = time.ParseDuration("15m")
	if err != nil {
		panic(err)
	}
}

var allowedLateness time.Duration

func (d *UserController) VerifyConfirmationEmail(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)

	user, err := models.FindUser(c.Context(), d.DBS().Reader, userID)
	if err != nil {
		return nil
	}
	if user.EmailConfirmed {
		return nil
	}
	if !user.EmailConfirmationKey.Valid || !user.EmailConfirmationSent.Valid {
		return nil
	}
	if time.Since(user.EmailConfirmationSent.Time) > allowedLateness {
		return nil
	}

	var confirmationBody struct {
		Key string `json:"key"`
	}
	c.BodyParser(&confirmationBody)

	if confirmationBody.Key == user.EmailConfirmationKey.String {
		user.EmailConfirmed = true
		user.Update(c.Context(), d.DBS().Writer, boil.Infer())
	}

	return nil
}

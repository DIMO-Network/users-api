package controllers

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/smtp"
	"regexp"
	"sort"
	"time"

	"github.com/DIMO-INC/users-api/internal/config"
	"github.com/DIMO-INC/users-api/internal/database"
	"github.com/DIMO-INC/users-api/models"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

// Sorted JSON array of valid ISO 3116-1 apha-3 codes
//go:embed country_codes.json
var rawCountryCodes []byte

type UserController struct {
	Settings        *config.Settings
	DBS             func() *database.DBReaderWriter
	log             *zerolog.Logger
	allowedLateness time.Duration
	countryCodes    []string
}

func NewUserController(settings *config.Settings, dbs func() *database.DBReaderWriter, logger *zerolog.Logger) UserController {
	var countryCodes []string
	if err := json.Unmarshal(rawCountryCodes, &countryCodes); err != nil {
		panic(err)
	}
	return UserController{
		Settings:        settings,
		DBS:             dbs,
		log:             logger,
		allowedLateness: 15 * time.Minute,
		countryCodes:    countryCodes,
	}
}

type userResponse struct {
	ID             string      `json:"id"`
	EmailAddress   null.String `json:"email_address"`
	EmailConfirmed bool        `json:"email_verified"`
	CreatedAt      time.Time   `json:"created_at"`
	CountryCode    null.String `json:"country_code"`
}

func formatUser(user *models.User) *userResponse {
	return &userResponse{user.ID, user.EmailAddress, user.EmailConfirmed, user.CreatedAt, user.CountryCode}
}

func (d *UserController) getOrCreateUser(ctx context.Context, userID string) (user *models.User, err error) {
	tx, err := d.DBS().Writer.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint

	user, err = models.FindUser(ctx, tx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// New user, insert a mostly-empty record
			user = &models.User{ID: userID}
			if err := user.Insert(ctx, tx, boil.Infer()); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return user, nil
}

func (d *UserController) GetUser(c *fiber.Ctx) error {
	userID := getUserID(c)

	user, err := d.getOrCreateUser(c.Context(), userID)
	if err != nil {
		panic(err)
	}
	return c.JSON(formatUser(user))
}

func inSorted(v []string, x string) bool {
	i := sort.SearchStrings(v, x)
	return i < len(v) && v[i] == x
}

func (d *UserController) UpdateUser(c *fiber.Ctx) error {
	userID := getUserID(c)

	user, err := d.getOrCreateUser(c.Context(), userID)
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	var body struct {
		EmailAddress null.String `json:"email_address"`
		CountryCode  null.String `json:"country_code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return errorResponseHandler(c, err, fiber.StatusBadRequest)
	}
	if body.CountryCode.Valid && !inSorted(d.countryCodes, body.CountryCode.String) {
		return errorResponseHandler(c, fmt.Errorf("invalid country code"), fiber.StatusBadRequest)
	}
	user.CountryCode = body.CountryCode

	if body.EmailAddress != user.EmailAddress {
		if body.EmailAddress.Valid {
			if !emailPattern.MatchString(body.EmailAddress.String) {
				return errorResponseHandler(c, fmt.Errorf("invalid email"), fiber.StatusBadRequest)
			}
		}
		user.EmailAddress = body.EmailAddress
		user.EmailConfirmed = false
		user.EmailConfirmationKey = null.StringFromPtr(nil)
		user.EmailConfirmationSent = null.TimeFromPtr(nil)
	}

	if _, err := user.Update(c.Context(), d.DBS().Writer, boil.Infer()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
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
	userID := getUserID(c)

	user, err := d.getOrCreateUser(c.Context(), userID)
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	if !user.EmailAddress.Valid {
		return errorResponseHandler(c, fmt.Errorf("user has not provided an email"), fiber.StatusBadRequest)
	}
	if user.EmailConfirmed {
		return errorResponseHandler(c, fmt.Errorf("email already confirmed"), fiber.StatusBadRequest)
	}

	key := generateConfirmationKey()
	user.EmailConfirmationKey = null.StringFrom(key)
	user.EmailConfirmationSent = null.TimeFrom(time.Now())
	if _, err := user.Update(c.Context(), d.DBS().Writer, boil.Infer()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	auth := smtp.PlainAuth("", d.Settings.EmailUsername, d.Settings.EmailPassword, d.Settings.EmailHost)
	addr := fmt.Sprintf("%s:%s", d.Settings.EmailHost, d.Settings.EmailPort)
	msg := []byte("From: DIMO Mailer <mailer@dimo.zone>\r\n" +
		"To: " + user.EmailAddress.String + "\r\n" +
		"Subject: DIMO email confirmation\r\n" +
		"\r\n" +
		"Your email confirmation code is\r\n" +
		"\r\n" +
		key + "\r\n")
	err = smtp.SendMail(addr, auth, d.Settings.EmailFrom, []string{user.EmailAddress.String}, msg)
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	return nil
}

// https://html.spec.whatwg.org/multipage/input.html#valid-e-mail-address
var emailPattern = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func (d *UserController) ConfirmEmail(c *fiber.Ctx) error {
	userID := getUserID(c)

	user, err := d.getOrCreateUser(c.Context(), userID)
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	if user.EmailConfirmed {
		return errorResponseHandler(c, fmt.Errorf("email already confirmed"), fiber.StatusBadRequest)
	}
	if !user.EmailConfirmationKey.Valid || !user.EmailConfirmationSent.Valid {
		return errorResponseHandler(c, fmt.Errorf("email confirmation never sent"), fiber.StatusBadRequest)
	}
	if time.Since(user.EmailConfirmationSent.Time) > d.allowedLateness {
		return errorResponseHandler(c, fmt.Errorf("email confirmation message expired"), fiber.StatusBadRequest)
	}

	var confirmationBody struct {
		Key string `json:"key"`
	}
	if err := c.BodyParser(&confirmationBody); err != nil {
		return errorResponseHandler(c, err, fiber.StatusBadRequest)
	}

	if confirmationBody.Key == user.EmailConfirmationKey.String {
		user.EmailConfirmed = true
		if _, err := user.Update(c.Context(), d.DBS().Writer, boil.Infer()); err != nil {
			return errorResponseHandler(c, err, fiber.StatusInternalServerError)
		}
		return nil
	}

	return errorResponseHandler(c, fmt.Errorf("email confirmation code invalid"), fiber.StatusBadRequest)
}

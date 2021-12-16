package controllers

import (
	"bytes"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"math/rand"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"regexp"
	"sort"
	"time"

	"github.com/DIMO-INC/users-api/internal"
	"github.com/DIMO-INC/users-api/internal/config"
	"github.com/DIMO-INC/users-api/internal/database"
	"github.com/DIMO-INC/users-api/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"google.golang.org/protobuf/proto"
)

// Sorted JSON array of valid ISO 3116-1 apha-3 codes
//go:embed country_codes.json
var rawCountryCodes []byte

//go:embed confirmation_email.html
var rawConfirmationEmail string

type UserController struct {
	Settings        *config.Settings
	DBS             func() *database.DBReaderWriter
	log             *zerolog.Logger
	allowedLateness time.Duration
	countryCodes    []string
	emailTemplate   *template.Template
}

func NewUserController(settings *config.Settings, dbs func() *database.DBReaderWriter, logger *zerolog.Logger) UserController {
	var countryCodes []string
	if err := json.Unmarshal(rawCountryCodes, &countryCodes); err != nil {
		panic(err)
	}
	t := template.Must(template.New("confirmation_email").Parse(rawConfirmationEmail))
	return UserController{
		Settings:        settings,
		DBS:             dbs,
		log:             logger,
		allowedLateness: 5 * time.Minute,
		countryCodes:    countryCodes,
		emailTemplate:   t,
	}
}

type UserResponse struct {
	// ID is the user's DIMO-internal ID
	ID string `json:"id" example:"ChFrb2JsaXR6QGRpbW8uem9uZRIGZ29vZ2xl"`
	// EmailAddress is the email address coming from a user's login or manual election
	EmailAddress null.String `json:"emailAddress" swaggertype:"string" example:"koblitz@dimo.zone"`
	// EmailConfirmed indicates whether DIMO has confirmed the user's ownership of
	// EmailAddress
	EmailConfirmed bool `json:"emailVerified" example:"false"`
	// EmailConfirmationSentAt is the time when we last sent the user an email
	// confirmation message, and is only present if such an email has been sent but
	// confirmation has not yet occurred
	EmailConfirmationSentAt null.Time `json:"emailConfirmationSentAt" swaggertype:"string" example:"2021-12-01T09:01:12Z"`
	// CreatedAt is when the user first logged in
	CreatedAt time.Time `json:"createdAt" swaggertype:"string" example:"2021-12-01T09:00:00Z"`
	// CountryCode, if present, is a valid ISO 3166-1 alpha-3 country code
	CountryCode null.String `json:"countryCode" swaggertype:"string" example:"USA"`
	// EthereumAddress is the Ethereum address used to log in, if the user did use Web3
	EthereumAddress null.String `json:"ethereumAddress" swaggertype:"string" example:"0x142e0C7A098622Ea98E5D67034251C4dFA746B5d"`
	// ReferralCode is the short code used in a user's share link
	ReferralCode string `json:"referralCode" example:"bUkZuSL7"`
}

func formatUser(user *models.User) *UserResponse {
	return &UserResponse{
		ID:                      user.ID,
		EmailAddress:            user.EmailAddress,
		EmailConfirmed:          user.EmailConfirmed,
		EmailConfirmationSentAt: user.EmailConfirmationSentAt,
		CreatedAt:               user.CreatedAt,
		CountryCode:             user.CountryCode,
		EthereumAddress:         user.EthereumAddress,
		ReferralCode:            user.ReferralCode,
	}
}

func getBooleanClaim(claims jwt.MapClaims, key string) (value, ok bool) {
	if rawValue, ok := claims[key]; ok {
		if value, ok := rawValue.(bool); ok {
			return value, true
		}
	}
	return false, false
}

func getStringClaim(claims jwt.MapClaims, key string) (value string, ok bool) {
	if rawValue, ok := claims[key]; ok {
		if value, ok := rawValue.(string); ok {
			return value, true
		}
	}
	return "", false
}

func (d *UserController) getOrCreateUser(c *fiber.Ctx, userID string) (user *models.User, err error) {
	tx, err := d.DBS().Writer.BeginTx(c.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint

	user, err = models.FindUser(c.Context(), tx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			user = &models.User{ID: userID, ReferralCode: generateReferralCode()}
			// New user, insert a mostly-empty record

			token := c.Locals("user").(*jwt.Token)
			claims := token.Claims.(jwt.MapClaims)

			if emailVerified, ok := getBooleanClaim(claims, "email_verified"); ok && emailVerified {
				if email, ok := getStringClaim(claims, "email"); ok {
					user.EmailAddress = null.StringFrom(email)
					user.EmailConfirmed = true
				}
			}

			if ethereumAddress, ok := getStringClaim(claims, "ethereum_address"); ok && ethereumAddress != "" {
				user.EthereumAddress = null.StringFrom(ethereumAddress)
			}

			if err := user.Insert(c.Context(), tx, boil.Infer()); err != nil {
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

// GetUser godoc
// @Summary Get attributes for the authenticated user
// @Produce json
// @Success 200 {object} controllers.UserResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Router /v1/user [get]
func (d *UserController) GetUser(c *fiber.Ctx) error {
	userID := getUserID(c)

	user, err := d.getOrCreateUser(c, userID)
	if err != nil {
		panic(err)
	}
	return c.JSON(formatUser(user))
}

func inSorted(v []string, x string) bool {
	i := sort.SearchStrings(v, x)
	return i < len(v) && v[i] == x
}

type optionalString struct {
	Defined bool
	Value   null.String
}

func (o *optionalString) UnmarshalJSON(data []byte) error {
	o.Defined = true
	return json.Unmarshal(data, &o.Value)
}

// UserUpdateRequest describes a user's request to modify or delete certain fields
type UserUpdateRequest struct {
	// EmailAddress, if specified, should be a valid email address. Note when this field
	// is modified the user's verification status will reset.
	EmailAddress optionalString `json:"emailAddress" swaggertype:"string" example:"neal@dimo.zone"`
	// CountryCode, if specified, should be a valid ISO 3166-1 alpha-3 country code
	CountryCode optionalString `json:"countryCode" swaggertype:"string" example:"USA"`
}

// UpdateUser godoc
// @Summary Modify attributes for the authenticated user
// @Accept json
// @Produce json
// @Param userUpdateRequest body controllers.UserUpdateRequest true "New field values"
// @Success 200 {object} controllers.UserResponse
// @Success 400 {object} controllers.ErrorResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Router /v1/user [put]
func (d *UserController) UpdateUser(c *fiber.Ctx) error {
	userID := getUserID(c)

	user, err := d.getOrCreateUser(c, userID)
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	var body UserUpdateRequest
	if err := c.BodyParser(&body); err != nil {
		return errorResponseHandler(c, err, fiber.StatusBadRequest)
	}

	if body.CountryCode.Defined {
		if body.CountryCode.Value.Valid && !inSorted(d.countryCodes, body.CountryCode.Value.String) {
			return errorResponseHandler(c, fmt.Errorf("invalid country code"), fiber.StatusBadRequest)
		}
		user.CountryCode = body.CountryCode.Value
	}

	if body.EmailAddress.Defined && body.EmailAddress.Value != user.EmailAddress {
		if body.EmailAddress.Value.Valid {
			if !emailPattern.MatchString(body.EmailAddress.Value.String) {
				return errorResponseHandler(c, fmt.Errorf("invalid email"), fiber.StatusBadRequest)
			}
		}
		user.EmailAddress = body.EmailAddress.Value
		user.EmailConfirmed = false
		user.EmailConfirmationKey = null.StringFromPtr(nil)
		user.EmailConfirmationSentAt = null.TimeFromPtr(nil)
	}

	if _, err := user.Update(c.Context(), d.DBS().Writer, boil.Infer()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	return c.JSON(formatUser(user))
}

// DeleteUser godoc
// @Summary Delete the authenticated user
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Router /v1/user [delete]
func (d *UserController) DeleteUser(c *fiber.Ctx) error {
	userID := getUserID(c)

	tx, err := d.DBS().Writer.BeginTx(c.Context(), nil)
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	defer tx.Rollback() //nolint

	user, err := models.FindUser(c.Context(), tx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errorResponseHandler(c, err, fiber.StatusBadRequest)
		}
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	if _, err := user.Delete(c.Context(), d.DBS().Writer); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	if err := tx.Commit(); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

var digits = []rune("0123456789")

func generateConfirmationKey() string {
	o := make([]rune, 6)
	for i := range o {
		o[i] = digits[rand.Intn(10)]
	}
	return string(o)
}

var validChars = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateReferralCode() string {
	o := make([]rune, 8)
	for i := range o {
		o[i] = validChars[rand.Intn(len(validChars))]
	}
	return string(o)
}

// SendConfirmationEmail godoc
// @Summary Send a confirmation email to the authenticated user
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Failure 500 {object} controllers.ErrorResponse
// @Router /v1/user/send-confirmation-email [post]
func (d *UserController) SendConfirmationEmail(c *fiber.Ctx) error {
	userID := getUserID(c)

	user, err := d.getOrCreateUser(c, userID)
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
	user.EmailConfirmationSentAt = null.TimeFrom(time.Now())

	auth := smtp.PlainAuth("", d.Settings.EmailUsername, d.Settings.EmailPassword, d.Settings.EmailHost)
	addr := fmt.Sprintf("%s:%s", d.Settings.EmailHost, d.Settings.EmailPort)

	var partsBuffer bytes.Buffer
	w := multipart.NewWriter(&partsBuffer)
	defer w.Close() //nolint

	p, err := w.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain"}, "Content-Transfer-Encoding": {"quoted-printable"}})
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	pw := quotedprintable.NewWriter(p)
	if _, err := pw.Write([]byte("Hi,\r\n\r\nYour email verification code is: " + key + "\r\n")); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	pw.Close()

	h, err := w.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/html"}, "Content-Transfer-Encoding": {"quoted-printable"}})
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	hw := quotedprintable.NewWriter(h)
	if err := d.emailTemplate.Execute(hw, struct{ Key string }{key}); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	hw.Close()

	var buffer bytes.Buffer
	buffer.WriteString("From: DIMO <" + d.Settings.EmailFrom + ">\r\n" +
		"To: " + user.EmailAddress.String + "\r\n" +
		"Subject: DIMO email confirmation\r\n" +
		"Content-Type: multipart/alternative; boundary=\"" + w.Boundary() + "\"\r\n" +
		"\r\n")
	if _, err := partsBuffer.WriteTo(&buffer); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	if err := smtp.SendMail(addr, auth, d.Settings.EmailFrom, []string{user.EmailAddress.String}, buffer.Bytes()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	if _, err := user.Update(c.Context(), d.DBS().Writer, boil.Infer()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// https://html.spec.whatwg.org/multipage/input.html#valid-e-mail-address
var emailPattern = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

type ConfirmEmailRequest struct {
	// Key is the 6-digit number from the confirmation email
	Key string `json:"key" example:"010990"`
}

// ConfirmEmail godoc
// @Summary Submit an email confirmation key
// @Accept json
// @Param confirmEmailRequest body controllers.ConfirmEmailRequest true "Specifies the key from the email"
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Router /v1/user/confirm-email [post]
func (d *UserController) ConfirmEmail(c *fiber.Ctx) error {
	userID := getUserID(c)

	user, err := d.getOrCreateUser(c, userID)
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	if user.EmailConfirmed {
		return errorResponseHandler(c, fmt.Errorf("email already confirmed"), fiber.StatusBadRequest)
	}
	if !user.EmailConfirmationKey.Valid || !user.EmailConfirmationSentAt.Valid {
		return errorResponseHandler(c, fmt.Errorf("email confirmation never sent"), fiber.StatusBadRequest)
	}
	if time.Since(user.EmailConfirmationSentAt.Time) > d.allowedLateness {
		return errorResponseHandler(c, fmt.Errorf("email confirmation message expired"), fiber.StatusBadRequest)
	}

	var confirmationBody struct {
		Key string `json:"key"`
	}
	if err := c.BodyParser(&confirmationBody); err != nil {
		return errorResponseHandler(c, err, fiber.StatusBadRequest)
	}

	if confirmationBody.Key != user.EmailConfirmationKey.String {
		return errorResponseHandler(c, fmt.Errorf("email confirmation code invalid"), fiber.StatusBadRequest)
	}

	user.EmailConfirmed = true
	user.EmailConfirmationKey = null.StringFromPtr(nil)
	user.EmailConfirmationSentAt = null.TimeFromPtr(nil)
	if _, err := user.Update(c.Context(), d.DBS().Writer, boil.Infer()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

var addressRegex = regexp.MustCompile("^0x[a-fA-F0-9]{40}$")

func (d *UserController) AdminCreateUser(c *fiber.Ctx) error {
	var body struct {
		NewID      string  `json:"new_id"`
		Email      string  `json:"email"`
		ReferralID string  `json:"referral_id"`
		CreatedAt  float64 `json:"created_at"`
		Region     string  `json:"region"`
		EthAddress string  `json:"eth_address"`
		GoogleID   string  `json:"google_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return errorResponseHandler(c, err, fiber.StatusBadRequest)
	}
	var user models.User
	if body.NewID == "" {
		return errorResponseHandler(c, fmt.Errorf("ID required"), fiber.StatusBadRequest)
	}
	user.ID = body.NewID

	if !emailPattern.MatchString(body.Email) {
		return errorResponseHandler(c, fmt.Errorf("invalid email"), fiber.StatusBadRequest)
	}
	user.EmailAddress = null.StringFrom(body.Email)
	user.EmailConfirmed = true

	if body.ReferralID == "" || len(body.ReferralID) > 12 {
		return errorResponseHandler(c, fmt.Errorf("invalid referral code"), fiber.StatusBadRequest)
	}
	user.ReferralCode = body.ReferralID

	if body.Region != "" {
		if !inSorted(d.countryCodes, body.Region) {
			return errorResponseHandler(c, fmt.Errorf("invalid country code"), fiber.StatusBadRequest)
		}
		user.CountryCode = null.StringFrom(body.Region)
	}

	if body.CreatedAt == 0 {
		return errorResponseHandler(c, fmt.Errorf("invalid creation time"), fiber.StatusBadRequest)
	}
	user.CreatedAt = time.UnixMicro(int64(1e6 * body.CreatedAt))

	if body.EthAddress != "" {
		if !addressRegex.MatchString(body.EthAddress) {
			return errorResponseHandler(c, fmt.Errorf("invalid Ethereum address"), fiber.StatusBadRequest)
		}
		user.EthereumAddress = null.StringFrom(body.EthAddress)
	}

	// One last sanity check
	var userSomething internal.IDTokenSubject
	data, err := base64.RawURLEncoding.DecodeString(body.NewID)
	if err != nil {
		return errorResponseHandler(c, fmt.Errorf("invalid ID: could not decode as base64"), fiber.StatusBadRequest)
	}

	if err := proto.Unmarshal(data, &userSomething); err != nil {
		return errorResponseHandler(c, fmt.Errorf("invalid ID: could not deserialize into protobuf"), fiber.StatusBadRequest)
	}

	if user.EthereumAddress.Valid {
		if userSomething.ConnId != "web3" {
			return errorResponseHandler(c, fmt.Errorf("invalid ID: Eth address given but connector not web3"), fiber.StatusBadRequest)
		}
		if userSomething.UserId != user.EthereumAddress.String {
			return errorResponseHandler(c, fmt.Errorf("invalid ID: Eth address in body and ID don't match"), fiber.StatusBadRequest)
		}
	} else {
		if userSomething.ConnId != "google" {
			return errorResponseHandler(c, fmt.Errorf("invalid ID: No Eth address given but connector not google"), fiber.StatusBadRequest)
		}
		if body.GoogleID == "" {
			return errorResponseHandler(c, fmt.Errorf("invalid ID: No Eth address or Google ID given"), fiber.StatusBadRequest)
		}
		if userSomething.UserId != body.GoogleID {
			return errorResponseHandler(c, fmt.Errorf("invalid ID: Google ID in body and ID don't match"), fiber.StatusBadRequest)
		}
	}

	if err := user.Insert(c.Context(), d.DBS().Writer, boil.Infer()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	return c.JSON(
		formatUser(&user),
	)
}

func (d *UserController) AdminViewUsers(c *fiber.Ctx) error {
	users, err := models.Users().All(c.Context(), d.DBS().Reader)
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusBadRequest)
	}
	return c.JSON(users)
}

func (d *UserController) AdminDeleteUser(c *fiber.Ctx) error {
	user, err := models.FindUser(c.Context(), d.DBS().Writer, c.Params("userID"))
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusBadRequest)
	}
	_, err = user.Delete(c.Context(), d.DBS().Writer)
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	return c.JSON(fiber.Map{"status": "overwhelming_success"})
}

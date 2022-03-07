package controllers

import (
	"bytes"
	"database/sql"
	_ "embed"
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

	"github.com/DIMO-INC/users-api/internal/config"
	"github.com/DIMO-INC/users-api/internal/database"
	"github.com/DIMO-INC/users-api/internal/services"
	"github.com/DIMO-INC/users-api/models"
	"github.com/customerio/go-customerio/v3"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
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
	cioClient       *customerio.CustomerIO
	eventService    *services.EventService
}

func NewUserController(settings *config.Settings, dbs func() *database.DBReaderWriter, eventService *services.EventService, logger *zerolog.Logger) UserController {
	rand.Seed(time.Now().UnixNano())
	var countryCodes []string
	if err := json.Unmarshal(rawCountryCodes, &countryCodes); err != nil {
		panic(err)
	}
	t := template.Must(template.New("confirmation_email").Parse(rawConfirmationEmail))
	var cioClient *customerio.CustomerIO
	if settings.CIOSiteID != "" && settings.CIOApiKey != "" {
		cioClient = customerio.NewTrackClient(
			settings.CIOSiteID,
			settings.CIOApiKey,
			customerio.WithRegion(customerio.RegionUS),
		)
	}
	return UserController{
		Settings:        settings,
		DBS:             dbs,
		log:             logger,
		allowedLateness: 5 * time.Minute,
		countryCodes:    countryCodes,
		emailTemplate:   t,
		cioClient:       cioClient,
		eventService:    eventService,
	}
}

type UserResponseEmail struct {
	// Address is the email address for the user.
	Address null.String `json:"address" swaggertype:"string" example:"koblitz@dimo.zone"`
	// Confirmed indicates whether the user has confirmed the address by entering a code.
	Confirmed bool `json:"confirmed" example:"false"`
	// ConfirmationSentAt is the time at which we last sent a confirmation email. This will only
	// be present if we've sent an email but the code has not been sent back to us.
	ConfirmationSentAt null.Time `json:"confirmationSentAt" swaggertype:"string" example:"2021-12-01T09:01:12Z"`
}

type UserResponseWeb3 struct {
	// Address is the Ethereum address associated with the user.
	Address null.String `json:"address" swaggertype:"string" example:"0x142e0C7A098622Ea98E5D67034251C4dFA746B5d"`
}

type UserResponse struct {
	// ID is the user's DIMO-internal ID.
	ID string `json:"id" example:"ChFrb2JsaXR6QGRpbW8uem9uZRIGZ29vZ2xl"`
	// Email describes the user's email and the state of its confirmation.
	Email UserResponseEmail `json:"email"`
	// Web3 describes the user's blockchain account.
	Web3 UserResponseWeb3 `json:"web3"`
	// CreatedAt is when the user first logged in.
	CreatedAt time.Time `json:"createdAt" swaggertype:"string" example:"2021-12-01T09:00:00Z"`
	// CountryCode, if present, is a valid ISO 3166-1 alpha-3 country code.
	CountryCode null.String `json:"countryCode" swaggertype:"string" example:"USA"`
	// ReferralCode is the short code used in a user's share link.
	ReferralCode string `json:"referralCode" example:"bUkZuSL7"`
	// ReferredBy is the referral code of the person who referred this user to the site.
	ReferredBy null.String `json:"referredBy" swaggertype:"string" example:"k9H7RoTG"`
	// AgreedTosAt is the time at which the user last agreed to the terms of service.
	AgreedTOSAt null.Time `json:"agreedTosAt" swaggertype:"string" example:"2021-12-01T09:00:41Z"`
	// ReferralsMade is the number of completed referrals made by the user
	ReferralsMade int `json:"referralsMade" example:"1"`
}

func formatUser(user *models.User) *UserResponse {
	refferedBy := func(user *models.User) null.String {
		if user.R != nil && user.R.Referrer != nil {
			return null.StringFrom(user.R.Referrer.ReferralCode)
		}
		return null.StringFromPtr(nil)
	}
	referralsMade := 0
	if user.R != nil && user.R.Referrals != nil {
		referralsMade = len(user.R.Referrals)
	}
	return &UserResponse{
		ID: user.ID,
		Email: UserResponseEmail{
			Address:            user.EmailAddress,
			Confirmed:          user.EmailConfirmed,
			ConfirmationSentAt: user.EmailConfirmationSentAt,
		},
		Web3: UserResponseWeb3{
			Address: user.EthereumAddress,
		},
		CreatedAt:     user.CreatedAt,
		CountryCode:   user.CountryCode,
		ReferralCode:  user.ReferralCode,
		ReferredBy:    refferedBy(user),
		AgreedTOSAt:   user.AgreedTosAt,
		ReferralsMade: referralsMade,
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

const UserCreationEventType = "com.dimo.zone.user.create"

type UserCreationEventData struct {
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"userId"`
	Method    string    `json:"method"`
}

func (d *UserController) getOrCreateUser(c *fiber.Ctx, userID string) (user *models.User, err error) {
	tx, err := d.DBS().Writer.BeginTx(c.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint

	newUser := false
	var providerID string

	user, err = models.Users(
		models.UserWhere.ID.EQ(userID),
		qm.Load(models.UserRels.Referrals),
	).One(c.Context(), tx)
	if err != nil {
		if err == sql.ErrNoRows {
			newUser = true
			user = &models.User{ID: userID, ReferralCode: generateReferralCode()}
			// New user, insert a mostly-empty record

			token := c.Locals("user").(*jwt.Token)
			claims := token.Claims.(jwt.MapClaims)

			// Some outstanding tokens may not have this field set. In the future, we would like to
			// reject such tokens.
			var providerClaim bool
			providerID, providerClaim = getStringClaim(claims, "provider_id")

			if emailVerified, ok := getBooleanClaim(claims, "email_verified"); ok && emailVerified {
				if email, ok := getStringClaim(claims, "email"); ok {
					user.EmailAddress = null.StringFrom(email)
					user.EmailConfirmed = true
					if !providerClaim {
						providerID = "google"
					}
				}
			}

			if ethereumAddress, ok := getStringClaim(claims, "ethereum_address"); ok && ethereumAddress != "" {
				user.EthereumAddress = null.StringFrom(ethereumAddress)
				if !providerClaim {
					providerID = "web3"
				}
				if d.cioClient != nil {
					go func() {
						if err := d.cioClient.Track(userID, "walletAdded", map[string]interface{}{}); err != nil {
							d.log.Error().Err(err).Msg("")
						}
					}()
				}
			}

			user.AuthProviderID = providerID

			d.log.Info().Msgf("Creating new user with id %s, provider %s", userID, providerID)

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

	if newUser {
		msg := UserCreationEventData{
			Timestamp: time.Now(),
			UserID:    userID,
			Method:    providerID,
		}
		err = d.eventService.Emit(&services.Event{
			Type:    UserCreationEventType,
			Subject: userID,
			Source:  "users-api",
			Data:    msg,
		})
		if err != nil {
			d.log.Err(err).Msg("Failed sending user creation event")
		}
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
	Email struct {
		// Address, if present, should be a valid email address. Note when this field
		// is modified the user's verification status will reset.
		Address optionalString `json:"address" swaggertype:"string" example:"neal@dimo.zone"`
	} `json:"email"`
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

	if body.Email.Address.Defined && body.Email.Address.Value != user.EmailAddress {
		if user.AuthProviderID == "google" || user.AuthProviderID == "apple" {
			return errorResponseHandler(c, fmt.Errorf("cannot change email address for Google or Apple accounts"), fiber.StatusBadRequest)
		}
		if body.Email.Address.Value.Valid {
			if !emailPattern.MatchString(body.Email.Address.Value.String) {
				return errorResponseHandler(c, fmt.Errorf("invalid email"), fiber.StatusBadRequest)
			}
		}
		user.EmailAddress = body.Email.Address.Value
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

// AgreeTOS godoc
// @Summary Agree to the current terms of service
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Router /v1/user/agree-tos [post]
func (d *UserController) AgreeTOS(c *fiber.Ctx) error {
	userID := getUserID(c)

	user, err := d.getOrCreateUser(c, userID)
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	user.AgreedTosAt = null.TimeFrom(time.Now())

	if _, err := user.Update(c.Context(), d.DBS().Writer, boil.Infer()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	return c.SendStatus(fiber.StatusNoContent)
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
		"Subject: [DIMO] Verification Code\r\n" +
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

package controllers

import (
	"bytes"
	"context"
	crypto_rand "crypto/rand"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"math/big"
	"math/rand"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"regexp"
	"sort"
	"strings"
	"time"

	pb "github.com/DIMO-Network/devices-api/pkg/grpc"
	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/users-api/internal/config"
	"github.com/DIMO-Network/users-api/internal/services"
	"github.com/DIMO-Network/users-api/models"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/volatiletech/null/v8"

	analytics "github.com/customerio/cdp-analytics-go"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Sorted JSON array of valid ISO 3116-1 apha-3 codes
//
//go:embed country_codes.json
var rawCountryCodes []byte

//go:embed confirmation_email.html
var rawConfirmationEmail string

var referralCodeRegex = regexp.MustCompile(`^[A-Z0-9]{6}$`)

type UserController struct {
	Settings        *config.Settings
	dbs             db.Store
	log             *zerolog.Logger
	allowedLateness time.Duration
	countryCodes    []string
	emailTemplate   *template.Template
	eventService    services.EventService
	devicesClient   DevicesAPI
	amClient        pb.AftermarketDeviceServiceClient
	cioClient       analytics.Client
}

type DevicesAPI interface {
	ListUserDevicesForUser(ctx context.Context, in *pb.ListUserDevicesForUserRequest, opts ...grpc.CallOption) (*pb.ListUserDevicesForUserResponse, error)
}

func NewUserController(settings *config.Settings, dbs db.Store, eventService services.EventService, logger *zerolog.Logger) UserController {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	var countryCodes []string
	if err := json.Unmarshal(rawCountryCodes, &countryCodes); err != nil {
		panic(err)
	}
	t := template.Must(template.New("confirmation_email").Parse(rawConfirmationEmail))

	gc, err := grpc.NewClient(settings.DevicesAPIGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	cioClient, err := analytics.NewWithConfig(settings.CustomerIOAPIKey, analytics.Config{})
	if err != nil {
		panic(err)
	}

	dc := pb.NewUserDeviceServiceClient(gc)

	amc := pb.NewAftermarketDeviceServiceClient(gc)

	return UserController{
		Settings:        settings,
		dbs:             dbs,
		log:             logger,
		allowedLateness: 5 * time.Minute,
		countryCodes:    countryCodes,
		emailTemplate:   t,
		eventService:    eventService,
		devicesClient:   dc,
		amClient:        amc,
		cioClient:       cioClient,
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
	// Confirmed indicates whether the user has confirmed the address by signing a challenge
	// message.
	Confirmed bool `json:"confirmed" example:"false"`
	// Used indicates whether the user has used this address to perform any on-chain
	// actions like minting, claiming, or pairing.
	Used bool `json:"used" example:"false"`
	// InApp indicates whether this is an in-app wallet, managed by the DIMO app.
	InApp bool `json:"inApp" example:"false"`
	// ChallengeSentAt is the time at which we last generated a challenge message for the user to
	// sign. This will only be present if we've generated such a message but a signature has not
	// been sent back to us.
	ChallengeSentAt null.Time `json:"challengeSentAt" swaggertype:"string" example:"2021-12-01T09:01:12Z"`
}

type UserResponse struct {
	// ID is the user's DIMO-internal ID.
	ID string `json:"id" example:"ChFrb2JsaXR6QGRpbW8uem9uZRIGZ29vZ2xl"`
	// Email describes the user's email and the state of its confirmation.
	Email UserResponseEmail `json:"email"`
	// Web3 describes the user's blockchain account.
	Web3       UserResponseWeb3 `json:"web3"`
	MigratedAt *time.Time       `json:"migratedAt" example:"2024-09-17T09:00:00Z"`
	// CreatedAt is when the user first logged in.
	CreatedAt time.Time `json:"createdAt" swaggertype:"string" example:"2021-12-01T09:00:00Z"`
	// CountryCode, if present, is a valid ISO 3166-1 alpha-3 country code.
	CountryCode null.String `json:"countryCode" swaggertype:"string" example:"USA"`
	// AgreedTosAt is the time at which the user last agreed to the terms of service.
	AgreedTOSAt null.Time `json:"agreedTosAt" swaggertype:"string" example:"2021-12-01T09:00:41Z"`
	// ReferralCode is the user's referral code to be given to others. It is an 8 alphanumeric code,
	// only present if the account has a confirmed Ethereum address.
	ReferralCode null.String `json:"referralCode" swaggertype:"string" example:"ANB95N"`
	ReferredBy   null.String `json:"referredBy" swaggertype:"string" example:"0x3497B704a954789BC39999262510DE9B09Ff1366"`
	ReferredAt   null.Time   `json:"referredAt" swaggertype:"string" example:"2021-12-01T09:00:41Z"`
}

type SubmitReferralCodeRequest struct {
	// ReferralCode is the 6-digit, alphanumeric referral code from another user.
	ReferralCode string `json:"referralCode" example:"ANB95N"`
}

type SubmitReferralCodeResponse struct {
	Message string `json:"message"`
}

func formatUser(user *models.User) *UserResponse {
	var referralCode null.String
	if user.EthereumConfirmed {
		referralCode = user.ReferralCode
	}

	var referrer null.String
	if user.R != nil && user.R.ReferringUser != nil && user.R.ReferringUser.EthereumConfirmed {
		referrer = null.StringFrom(common.BytesToAddress(user.R.ReferringUser.EthereumAddress.Bytes).Hex())
	}

	return &UserResponse{
		ID: user.ID,
		Email: UserResponseEmail{
			Address:            user.EmailAddress,
			Confirmed:          user.EmailConfirmed,
			ConfirmationSentAt: user.EmailConfirmationSentAt,
		},
		Web3: UserResponseWeb3{
			Address:         null.StringFrom(common.BytesToAddress(user.EthereumAddress.Bytes).Hex()),
			Confirmed:       user.EthereumConfirmed,
			ChallengeSentAt: user.EthereumChallengeSent,
			InApp:           user.InAppWallet,
		},
		CreatedAt:    user.CreatedAt,
		CountryCode:  user.CountryCode,
		AgreedTOSAt:  user.AgreedTosAt,
		ReferralCode: referralCode,
		ReferredBy:   referrer,
		ReferredAt:   user.ReferredAt,
		MigratedAt:   user.MigratedAt.Ptr(),
	}
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
	tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint

	user, err = models.Users(
		models.UserWhere.ID.EQ(userID),
		qm.Load(models.UserRels.ReferringUser),
	).One(c.Context(), tx)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}

		// New user, generate a record
		token := c.Locals("user").(*jwt.Token)
		claims := token.Claims.(jwt.MapClaims)

		providerID, ok := getStringClaim(claims, "provider_id")
		if !ok {
			return nil, errors.New("no provider_id claim in ID token")
		}

		user = &models.User{ID: userID, AuthProviderID: providerID}

		switch providerID {
		case "apple", "google":
			email, ok := getStringClaim(claims, "email")
			if !ok {
				return nil, fmt.Errorf("provider %s but no email claim in ID token", providerID)
			}
			if !emailPattern.MatchString(email) {
				return nil, fmt.Errorf("invalid email address %s", email)
			}
			user.EmailAddress = null.StringFrom(email)
			user.EmailConfirmed = true
			user.EthereumConfirmed = false
		case "web3":
			ethereum, ok := getStringClaim(claims, "ethereum_address")
			if !ok {
				return nil, fmt.Errorf("provider %s but no ethereum_address claim in ID token", providerID)
			}

			mixAddr, err := common.NewMixedcaseAddressFromString(ethereum)
			if err != nil {
				return nil, fmt.Errorf("invalid ethereum_address %s", ethereum)
			}
			if !mixAddr.ValidChecksum() {
				d.log.Warn().Msgf("ethereum_address %s in ID token is not checksummed", ethereum)
			}

			referralCode, err := d.GenerateReferralCode(c.Context())
			if err != nil {
				d.log.Error().Err(err).Msg("error occurred creating referral code for user")
				return nil, errors.New("internal error")
			}

			user.ReferralCode = null.StringFrom(referralCode)
			user.EthereumAddress = null.BytesFrom(mixAddr.Address().Bytes())
			user.EthereumConfirmed = true
		default:
			return nil, fmt.Errorf("unrecognized provider_id %s", providerID)
		}

		d.log.Info().Msgf("Creating new user with id %s, provider %s", userID, providerID)

		if err := user.Insert(c.Context(), tx, boil.Infer()); err != nil {
			return nil, err
		}

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

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return user, nil
}

// getUserByEth gets the users from the db with matching eth addr, but selects the one with email confirmed if more than one
func (d *UserController) getUserByEth(ctx context.Context, ethAddr common.Address) (user *models.User, err error) {
	user, err = models.Users(
		models.UserWhere.EthereumAddress.EQ(null.BytesFrom(ethAddr.Bytes())),
		qm.Load(models.UserRels.ReferringUser),
		qm.OrderBy("email_confirmed desc"),
	).One(ctx, d.dbs.DBS().Reader)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserV2 godoc
// @Summary Get attributes for the authenticated user. If multiple records for the same user, gets the one with the email confirmed.
// @Produce json
// @Success 200 {object} controllers.UserResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Security BearerAuth
// @Router /v2/user [get]
func (d *UserController) GetUserV2(c *fiber.Ctx) error {
	userID := getUserID(c)

	user, err := models.Users(
		models.UserWhere.ID.EQ(userID),
		qm.Load(models.UserRels.ReferringUser),
	).One(c.Context(), d.dbs.DBS().Reader)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("No user with id %s.", userID))
		}
		return err
	}

	// TODO(elffjs): The stuff below makes me incredibly nervous. Commenting it out for now. Talk to James.
	// many users have multiple entries for the same eth_addr, but we want to use the one with verified email
	// get users by eth addr, if it exists, order by email_confirmed desc, if userId different, use the better user but just replace the user_id
	// ethAddr := getUserEthAddr(c)
	// if ethAddr != nil {
	// 	userBetter, err := d.getUserByEth(c.Context(), *ethAddr)
	// 	if err == nil {
	// 		if user.ID != userBetter.ID {
	// 			// use the user with better information but preserve the original ID of the claim so not to potentially break stuff
	// 			user = userBetter
	// 			user.ID = userID
	// 		}
	// 	}
	// }

	out := formatUser(user)

	out.Web3.Used, err = d.computeWeb3Used(c.Context(), user)
	if err != nil {
		d.log.Err(err).Str("userId", userID).Msg("Failed to determine whether user owns any NFTs.")
	}

	return c.JSON(out)
}

// GetUser godoc
// @Summary Get attributes for the authenticated user. If multiple records for the same user, gets the one with the email confirmed.
// @Produce json
// @Success 200 {object} controllers.UserResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Security BearerAuth
// @Router /v1/user [get]
func (d *UserController) GetUser(c *fiber.Ctx) error {
	userID := getUserID(c)
	user, err := d.getOrCreateUser(c, userID)
	if err != nil {
		return err
	}
	// many users have multiple entries for the same eth_addr, but we want to use the one with verified email
	// get users by eth addr, if it exists, order by email_confirmed desc, if userId different, use the better user but just replace the user_id
	ethAddr := getUserEthAddr(c)
	if ethAddr != nil {
		userBetter, err := d.getUserByEth(c.Context(), *ethAddr)
		if err == nil {
			if user.ID != userBetter.ID {
				// use the user with better information but preserve the original ID of the claim so not to potentially break stuff
				user = userBetter
				user.ID = userID
			}
		}
	}

	out := formatUser(user)

	out.Web3.Used, err = d.computeWeb3Used(c.Context(), user)
	if err != nil {
		d.log.Err(err).Str("userId", userID).Msg("Failed to determine whether user owns any NFTs.")
	}

	return c.JSON(out)
}

func (d *UserController) computeWeb3Used(ctx context.Context, user *models.User) (bool, error) {
	if user.AuthProviderID == "web3" {
		return true, nil
	}

	if !user.EthereumConfirmed {
		return false, nil
	}

	devices, err := d.devicesClient.ListUserDevicesForUser(ctx, &pb.ListUserDevicesForUserRequest{UserId: user.ID})
	if err != nil {
		return false, fmt.Errorf("couldn't retrieve user's vehicles: %w", err)
	}

	for _, amd := range devices.UserDevices {
		if amd.TokenId != nil {
			return true, nil
		}
	}

	ams, err := d.amClient.ListAftermarketDevicesForUser(ctx, &pb.ListAftermarketDevicesForUserRequest{UserId: user.ID})
	if err != nil {
		return false, fmt.Errorf("couldn't retrieve user's aftermarket devices: %w", err)
	}

	for _, am := range ams.AftermarketDevices {
		if len(am.OwnerAddress) == 20 {
			return true, nil
		}
	}

	return false, nil
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
	Web3 struct {
		// Address, if present, should be a valid ethereum address. Note when this field
		// is modified the user's address verification status will reset.
		Address optionalString `json:"address" swaggertype:"string" example:"0x71C7656EC7ab88b098defB751B7401B5f6d8976F"`
		// InApp, if true, indicates that the address above corresponds to an in-app wallet.
		// You can only set this when setting a new wallet. It defaults to false.
		InApp bool `json:"inApp" example:"true"`
	} `json:"web3"`
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

	if body.Web3.Address.Defined && body.Web3.Address.Value != null.StringFrom(common.BytesToAddress(user.EthereumAddress.Bytes).Hex()) {
		ethereum := body.Web3.Address.Value
		if ethereum.Valid {
			mixAddr, err := common.NewMixedcaseAddressFromString(ethereum.String)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Invalid Ethereum address %s.", mixAddr))
			}
			ethereum = null.StringFrom(mixAddr.Address().Hex())
		}

		ethAddress := common.HexToAddress(ethereum.String)

		user.EthereumAddress = null.BytesFrom(ethAddress.Bytes())
		user.EthereumConfirmed = false
		user.InAppWallet = body.Web3.InApp
		user.EthereumChallengeSent = null.Time{}
		user.EthereumChallenge = null.String{}
	}

	if _, err := user.Update(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	return c.JSON(formatUser(user))
}

// DeleteUser godoc
// @Summary Delete the authenticated user. Fails if the user has any devices.
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Failure 409 {object} controllers.ErrorResponse "Returned if the user still has devices."
// @Router /v1/user [delete]
func (d *UserController) DeleteUser(c *fiber.Ctx) error {
	userID := getUserID(c)

	tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), nil)
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

	dr, err := d.devicesClient.ListUserDevicesForUser(c.Context(), &pb.ListUserDevicesForUserRequest{UserId: userID})
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	if l := len(dr.UserDevices); l > 0 {
		return errorResponseHandler(c, fmt.Errorf("user must delete %d devices first", l), fiber.StatusConflict)
	}

	if _, err := user.Delete(c.Context(), d.dbs.DBS().Writer); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	if err := tx.Commit(); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	d.log.Info().Str("userId", userID).Msg("Deleted user.")

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

func generateNonce() (string, error) {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	alphabetSize := big.NewInt(int64(len(alphabet)))
	b := make([]byte, 30)
	for i := range b {
		c, err := crypto_rand.Int(crypto_rand.Reader, alphabetSize)
		if err != nil {
			return "", err
		}
		b[i] = alphabet[c.Int64()]
	}
	return string(b), nil
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

	if _, err := user.Update(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// SetMigrated godoc
// @Summary Sets the migration timestamp.
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Router /v1/user/set-migrated [post]
func (d *UserController) SetMigrated(c *fiber.Ctx) error {
	userID := getUserID(c)

	user, err := d.getOrCreateUser(c, userID)
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	if d.Settings.Environment == "dev" && c.Query("clear") == "true" {
		user.MigratedAt = null.TimeFromPtr(nil)
	} else {
		user.MigratedAt = null.TimeFrom(time.Now())
	}

	if _, err := user.Update(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
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

	logger := d.log.With().Str("userId", userID).Logger()

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
	if user.EmailConfirmationSentAt.Valid && time.Since(user.EmailConfirmationSentAt.Time) < d.allowedLateness {
		logger.Error().Msgf("Rejecting confirmation email request: sent one at %s.", user.EmailConfirmationSentAt.Time)
		return errorResponseHandler(c, errors.New("email confirmation sent recently, please wait"), fiber.StatusConflict)
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

	if _, err := user.Update(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

type ChallengeResponse struct {
	// Challenge is the message to be signed.
	Challenge string `json:"challenge"`
	// ExpiresAt is the time at which the signed challenge will no longer be accepted.
	ExpiresAt time.Time `json:"expiresAt"`
}

var opaqueInternalError = fiber.NewError(fiber.StatusInternalServerError, "Internal error.")

// GenerateEthereumChallenge godoc
// @Summary Generate a challenge message for the user to sign.
// @Success 200 {object} controllers.ChallengeResponse
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 500 {object} controllers.ErrorResponse
// @Router /v1/user/web3/challenge/generate [post]
func (d *UserController) GenerateEthereumChallenge(c *fiber.Ctx) error {
	userID := getUserID(c)

	user, err := d.getOrCreateUser(c, userID)
	if err != nil {
		// TODO: Distinguish between bad tokens and server faults.
		d.log.Err(err).Str("userId", userID).Msg("Failed to get or create user.")
		return opaqueInternalError
	}

	nonce, err := generateNonce()
	if err != nil {
		d.log.Err(err).Str("userId", userID).Msg("Failed to generate nonce.")
		return opaqueInternalError
	}

	if user.EthereumConfirmed {
		return fiber.NewError(fiber.StatusBadRequest, "Ethereum address already confirmed.")
	}

	if !user.EthereumAddress.Valid {
		return fiber.NewError(fiber.StatusBadRequest, "No ethereum address to confirm.")
	}

	challenge := fmt.Sprintf("%s is asking you to please verify ownership of the address %s by signing this random string: %s", c.Hostname(), common.BytesToAddress(user.EthereumAddress.Bytes).Hex(), nonce)

	now := time.Now()
	user.EthereumChallengeSent = null.TimeFrom(now)
	user.EthereumChallenge = null.StringFrom(challenge)

	if _, err := user.Update(c.Context(), d.dbs.DBS().Reader, boil.Infer()); err != nil {
		d.log.Err(err).Str("userId", userID).Msg("Failed to update database record with new challenge.")
		return opaqueInternalError
	}

	return c.JSON(
		ChallengeResponse{
			Challenge: challenge,
			ExpiresAt: now.Add(d.allowedLateness),
		},
	)
}

type ConfirmEthereumRequest struct {
	// Signature is the result of signing the provided challenge message using the address in
	// question.
	Signature string `json:"signature"`
}

var randInt = func(n *big.Int) (*big.Int, error) {
	return crypto_rand.Int(crypto_rand.Reader, n)
}

func (d *UserController) GenerateReferralCode(ctx context.Context) (string, error) {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	alphabetSize := big.NewInt(int64(len(alphabet)))

	getCode := func() (string, error) {
		res := make([]byte, 6)
		for i := 0; i < 6; i++ {
			d, err := randInt(alphabetSize)
			if err != nil {
				return "", err
			}
			res[i] = alphabet[d.Int64()]
		}
		return string(res), nil
	}

	for {
		code, err := getCode()
		if err != nil {
			return "", err
		}

		exists, err := models.Users(models.UserWhere.ReferralCode.EQ(null.StringFrom(code))).Exists(ctx, d.dbs.DBS().Reader)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
}

// SubmitEthereumChallenge godoc
// @Summary Confirm ownership of an ethereum address by submitting a signature
// @Param confirmEthereumRequest body controllers.ConfirmEthereumRequest true "Signed challenge message"
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 500 {object} controllers.ErrorResponse
// @Router /v1/user/web3/challenge/submit [post]
func (d *UserController) SubmitEthereumChallenge(c *fiber.Ctx) error {
	userID := getUserID(c)

	user, err := d.getOrCreateUser(c, userID)
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	if user.EthereumConfirmed {
		return fiber.NewError(fiber.StatusBadRequest, "ethereum address already confirmed")
	}

	if !user.EthereumChallengeSent.Valid || !user.EthereumChallenge.Valid {
		return fiber.NewError(fiber.StatusBadRequest, "ethereum challenge never generated")
	}

	if time.Since(user.EthereumChallengeSent.Time) > d.allowedLateness {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("ethereum challenge expired at %s", user.EthereumChallengeSent.Time))
	}

	submitBody := new(ConfirmEthereumRequest)

	if err := c.BodyParser(submitBody); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	addrb := common.BytesToAddress(user.EthereumAddress.Bytes)

	signb, err := hexutil.Decode(submitBody.Signature)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not decode hex signature")
	}

	// This is the v parameter in the signature. Per the yellow paper, 27 means even and 28
	// means odd; it is our responsibility to shift it before passing it to crypto functions.
	switch signb[64] {
	case 0, 1:
		// This is not standard, but it seems to be what Ledger does.
		break
	case 27, 28:
		signb[64] -= 27
	default:
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("invalid v parameter %d", signb[64]))
	}

	pubKey, err := crypto.SigToPub(signHash([]byte(user.EthereumChallenge.String)), signb)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not recover public key from signature")
	}

	// TODO(elffjs): Why can't we just use crypto.Ecrecover?
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	// These are byte arrays, not slices, so this is okay to do.
	if recoveredAddr != addrb {
		return fiber.NewError(fiber.StatusBadRequest, "given address and recovered address do not match")
	}

	referralCode, err := d.GenerateReferralCode(c.Context())
	if err != nil {
		d.log.Error().Err(err).Msg("error occurred creating referral code for user")
		return fiber.NewError(fiber.StatusInternalServerError, "internal error")
	}

	user.ReferralCode = null.StringFrom(referralCode)
	user.EthereumConfirmed = true
	user.EthereumChallengeSent = null.Time{}
	user.EthereumChallenge = null.String{}
	if _, err := user.Update(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal error")
	}

	if user.EmailConfirmed && user.EmailAddress.Valid {
		// This would be quite alarming, if it's not valid.
		typ := "eoa"
		if user.InAppWallet {
			typ = "in_app"
		}

		err := d.cioClient.Enqueue(
			analytics.Identify{
				UserId: user.EmailAddress.String,
				Traits: analytics.NewTraits().Set("ethereum_wallet_address", addrb.Hex()).Set("legacy_wallet_address", addrb.Hex()).Set("legacy_wallet_address_type", typ),
			},
		)
		if err != nil {
			d.log.Err(err).Str("userId", user.ID).Msg("Error sending CIO event on wallet confirmation.")
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
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

	confirmationBody := new(ConfirmEmailRequest)
	if err := c.BodyParser(confirmationBody); err != nil {
		return errorResponseHandler(c, err, fiber.StatusBadRequest)
	}

	if confirmationBody.Key != user.EmailConfirmationKey.String {
		return errorResponseHandler(c, fmt.Errorf("email confirmation code invalid"), fiber.StatusBadRequest)
	}

	user.EmailConfirmed = true
	user.EmailConfirmationKey = null.StringFromPtr(nil)
	user.EmailConfirmationSentAt = null.TimeFromPtr(nil)
	if _, err := user.Update(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	if user.EthereumConfirmed && user.EthereumAddress.Valid {
		addr := common.BytesToAddress(user.EthereumAddress.Bytes)
		// This would be quite alarming to not end up here.
		typ := "eoa"
		if user.InAppWallet {
			typ = "in_app"
		}

		err := d.cioClient.Enqueue(
			analytics.Identify{
				UserId: user.EmailAddress.String,
				Traits: analytics.NewTraits().Set("ethereum_wallet_address", addr.Hex()).Set("legacy_wallet_address", addr.Hex()).Set("legacy_wallet_address_type", typ),
			},
		)
		if err != nil {
			d.log.Err(err).Str("userId", user.ID).Msg("Error sending CIO event on email confirmation.")
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// SubmitReferralCode godoc
// @Summary Takes the referral code, validates and stores it
// @Param submitReferralCodeRequest body controllers.SubmitReferralCodeRequest true "ReferralCode is the 6-digit, alphanumeric referral code from another user."
// @Success 200 {object} controllers.SubmitReferralCodeResponse
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 500 {object} controllers.ErrorResponse
// @Router /v1/user/submit-referral-code [post]
func (d *UserController) SubmitReferralCode(c *fiber.Ctx) error {
	userID := getUserID(c)
	user, err := d.getOrCreateUser(c, userID)
	if err != nil {
		return err
	}

	// TODO(elffjs): Some of these should probably be conflicts.
	if user.ReferredAt.Valid {
		return fiber.NewError(fiber.StatusBadRequest, "Already entered a referral code.")
	}

	dr, err := d.devicesClient.ListUserDevicesForUser(c.Context(), &pb.ListUserDevicesForUserRequest{UserId: userID})
	if err != nil {
		return err
	}

	if len(dr.UserDevices) != 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Can't enter a referral code after adding vehicles.")
	}

	var body SubmitReferralCodeRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Couldn't parse request body.")
	}

	d.log.Info().Str("userId", userID).Msgf("Got referral code %q.", body.ReferralCode)

	referralCode := strings.ToUpper(strings.TrimSpace(body.ReferralCode))

	if !referralCodeRegex.MatchString(referralCode) {
		return fiber.NewError(fiber.StatusBadRequest, "Referral code must be 6 characters and consist of digits and upper-case letters.")
	}

	if user.ReferralCode.Valid && user.ReferralCode.String == referralCode {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot refer self.")
	}

	referrer, err := models.Users(models.UserWhere.ReferralCode.EQ(null.StringFrom(referralCode))).One(c.Context(), d.dbs.DBS().Reader)
	if err != nil {
		if err == sql.ErrNoRows {
			return fiber.NewError(fiber.StatusBadRequest, "No user with that referral code found.")
		}
		return err
	}

	if user.EthereumAddress.Valid && common.BytesToAddress(user.EthereumAddress.Bytes) == common.BytesToAddress(referrer.EthereumAddress.Bytes) {
		return fiber.NewError(fiber.StatusBadRequest, "User and referrer have the same Ethereum address.")
	}

	// No circular referrals.
	if referrer.ReferringUserID.Valid && referrer.ReferringUserID.String == user.ID {
		return fiber.NewError(fiber.StatusBadRequest, "Referrer was referred by this user.")
	}

	user.ReferringUserID = null.StringFrom(referrer.ID)
	user.ReferredAt = null.TimeFrom(time.Now())
	if _, err := user.Update(c.Context(), d.dbs.DBS().Writer, boil.Whitelist(models.UserColumns.ReferringUserID, models.UserColumns.ReferredAt)); err != nil {
		return err
	}

	return c.JSON(SubmitReferralCodeResponse{
		Message: "Referral code used.",
	})
}

type AltAccount struct {
	// Type is the authentication provider, one of "web3", "apple", "google".
	Type string `json:"type"`
	// Login is the login username for the provider, either an email address
	// or an EIP-55-compliant ethereum address.
	Login string `json:"login"`
}

type AlternateAccountsResponse struct {
	// OtherAccounts is a list of any other accounts that share email or
	// ethereum address with the provided token.
	OtherAccounts []*AltAccount `json:"otherAccounts"`
}

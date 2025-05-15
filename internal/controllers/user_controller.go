package controllers

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	pb "github.com/DIMO-Network/devices-api/pkg/grpc"
	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/users-api/internal/config"
	"github.com/DIMO-Network/users-api/models"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/volatiletech/null/v8"

	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserController struct {
	Settings        *config.Settings
	dbs             db.Store
	log             *zerolog.Logger
	allowedLateness time.Duration
	devicesClient   DevicesAPI
	amClient        pb.AftermarketDeviceServiceClient
}

type DevicesAPI interface {
	ListUserDevicesForUser(ctx context.Context, in *pb.ListUserDevicesForUserRequest, opts ...grpc.CallOption) (*pb.ListUserDevicesForUserResponse, error)
}

func NewUserController(settings *config.Settings, dbs db.Store, logger *zerolog.Logger) UserController {
	gc, err := grpc.NewClient(settings.DevicesAPIGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		devicesClient:   dc,
		amClient:        amc,
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

func (d *UserController) getOrCreateUser(c *fiber.Ctx, userID string) (user *models.User, err error) {
	user, err = models.Users(
		models.UserWhere.ID.EQ(userID),
		qm.Load(models.UserRels.ReferringUser),
	).One(c.Context(), d.dbs.DBS().Reader)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}

		return nil, fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("No user with id %q found. This API is deprecated and new users cannot be created.", userID))
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

// TODO(elffjs): Really need to get rid of this. At least hit Identity or something.
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

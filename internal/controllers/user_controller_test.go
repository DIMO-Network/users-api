package controllers

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	pb "github.com/DIMO-Network/devices-api/pkg/grpc"
	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/users-api/internal/database"
	"github.com/DIMO-Network/users-api/internal/services"
	"github.com/DIMO-Network/users-api/models"
	analytics "github.com/customerio/cdp-analytics-go"
	"github.com/docker/go-connections/nat"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"google.golang.org/grpc"
)

type UserControllerTestSuite struct {
	suite.Suite
	dbcont testcontainers.Container
	dbs    db.Store
	logger *zerolog.Logger
}

func TestUserControllerSuite(t *testing.T) {
	suite.Run(t, &UserControllerTestSuite{})
}

func (s *UserControllerTestSuite) SetupSuite() {
	ctx := context.Background()

	logger := zerolog.Nop()
	s.logger = &logger

	port := 5432
	nport := fmt.Sprintf("%d/tcp", port)

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16.6-alpine",
		ExposedPorts: []string{nport},
		AutoRemove:   true,
		Env: map[string]string{
			"POSTGRES_DB":       "users_api",
			"POSTGRES_PASSWORD": "postgres",
		},
		WaitingFor: wait.ForListeningPort(nat.Port(nport)),
	}
	dbcont, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	s.Require().NoError(err)
	s.dbcont = dbcont

	host, err := dbcont.Host(ctx)
	s.Require().NoError(err)

	mport, err := dbcont.MappedPort(ctx, nat.Port(nport))
	s.Require().NoError(err)

	dbset := db.Settings{
		User:               "postgres",
		Password:           "postgres",
		Port:               mport.Port(),
		Host:               host,
		Name:               "users_api",
		MaxOpenConnections: 10,
		MaxIdleConnections: 10,
	}

	err = database.MigrateDatabase(ctx, logger, &dbset, "", "../../migrations")
	s.Require().NoError(err)

	dbs := db.NewDbConnectionFromSettings(ctx, &dbset, true)
	dbs.WaitForDB(logger)

	s.dbs = dbs
}

func (s *UserControllerTestSuite) TearDownSuite() {
	s.Require().NoError(s.dbcont.Terminate(context.Background()))
}

func (s *UserControllerTestSuite) TearDownTest() {
	_, err := models.Users().DeleteAll(context.Background(), s.dbs.DBS().Writer)
	s.Require().NoError(err)
}

type es struct{}

func (e *es) Emit(*services.Event) error {
	return nil
}

type udsc struct {
	store map[string][]*pb.UserDevice
}

func (c *udsc) ListUserDevicesForUser(_ context.Context, in *pb.ListUserDevicesForUserRequest, _ ...grpc.CallOption) (*pb.ListUserDevicesForUserResponse, error) {
	return &pb.ListUserDevicesForUserResponse{UserDevices: c.store[in.UserId]}, nil
}

type adsc struct{}

func (c *adsc) ListAftermarketDevicesForUser(_ context.Context, _ *pb.ListAftermarketDevicesForUserRequest, _ ...grpc.CallOption) (*pb.ListAftermarketDevicesForUserResponse, error) {
	return &pb.ListAftermarketDevicesForUserResponse{AftermarketDevices: []*pb.AftermarketDevice{}}, nil
}

type dummyCIO struct{}

func (c *dummyCIO) Close() error {
	return nil
}

func (c *dummyCIO) Enqueue(analytics.Message) error {
	return nil
}

func (s *UserControllerTestSuite) TestGetUser_OnlyUserID() {
	ctx := context.Background()

	uc := UserController{
		dbs:             s.dbs,
		log:             s.logger,
		allowedLateness: 5 * time.Minute,
		countryCodes:    []string{"USA", "CAN"},
		emailTemplate:   nil,
		eventService:    &es{},
		devicesClient:   &udsc{},
		amClient:        &adsc{},
	}

	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &jwt.Token{Claims: jwt.MapClaims{
			"provider_id": "apple",
			"sub":         "Cwbs",
			"email":       "steve@apple.com",
		}})
		return c.Next()
	})

	pk, err := crypto.GenerateKey()
	s.Require().NoError(err)

	addr := crypto.PubkeyToAddress(pk.PublicKey)

	nu := models.User{
		ID:                "SomeID",
		EmailConfirmed:    true,
		CreatedAt:         time.Now(),
		ReferralCode:      null.StringFrom("123456"),
		EthereumAddress:   null.BytesFrom(addr.Bytes()),
		EthereumConfirmed: true,
	}

	nu2 := models.User{
		ID:                "Cwbs",
		EmailConfirmed:    true,
		CreatedAt:         time.Now(),
		ReferralCode:      null.StringFrom("789abx"),
		EthereumAddress:   null.BytesFrom(addr.Bytes()),
		EthereumConfirmed: true,
		ReferringUserID:   null.StringFrom(nu.ID),
	}

	err = nu.Insert(ctx, uc.dbs.DBS().Writer, boil.Infer())
	s.Require().NoError(err)

	err = nu2.Insert(ctx, uc.dbs.DBS().Writer, boil.Infer())
	s.Require().NoError(err)

	app.Get("/", uc.GetUser)

	r := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	eResp := UserResponse{}
	err = json.NewDecoder(resp.Body).Decode(&eResp)
	s.Require().NoError(err)

	s.Require().Equal(200, resp.StatusCode)
	s.Require().Equal(eResp.ReferredBy, null.StringFrom(common.BytesToAddress(nu.EthereumAddress.Bytes).Hex()))
}

func (s *UserControllerTestSuite) TestGetUser_EthAddr() {
	ctx := context.Background()

	uc := UserController{
		dbs:             s.dbs,
		log:             s.logger,
		allowedLateness: 5 * time.Minute,
		countryCodes:    []string{"USA", "CAN"},
		emailTemplate:   nil,
		eventService:    &es{},
		devicesClient:   &udsc{},
		amClient:        &adsc{},
	}

	pk, err := crypto.GenerateKey()
	s.Require().NoError(err)

	addr := crypto.PubkeyToAddress(pk.PublicKey)

	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &jwt.Token{Claims: jwt.MapClaims{
			"provider_id":      "apple",
			"sub":              "SomeID",
			"ethereum_address": addr.Hex(),
		}})
		return c.Next()
	})
	// both users with same eth addr
	nu := models.User{
		ID:                "SomeID",
		EmailConfirmed:    false, // we don't want this user to be returned
		CreatedAt:         time.Now(),
		ReferralCode:      null.StringFrom("123456"),
		EthereumAddress:   null.BytesFrom(addr.Bytes()),
		EthereumConfirmed: true,
	}
	nu2 := models.User{
		ID:                "Cwbs",
		EmailConfirmed:    true,
		EmailAddress:      null.StringFrom("steve@crapple.com"),
		CreatedAt:         time.Now(),
		ReferralCode:      null.StringFrom("789abx"),
		EthereumAddress:   null.BytesFrom(addr.Bytes()),
		EthereumConfirmed: true,
	}

	err = nu.Insert(ctx, uc.dbs.DBS().Writer, boil.Infer())
	s.Require().NoError(err)

	err = nu2.Insert(ctx, uc.dbs.DBS().Writer, boil.Infer())
	s.Require().NoError(err)

	app.Get("/", uc.GetUser)

	r := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	eResp := UserResponse{}
	err = json.NewDecoder(resp.Body).Decode(&eResp)
	s.Require().NoError(err)

	s.Assert().Equal(200, resp.StatusCode)
	s.Require().Equal(eResp.ID, nu.ID)                                     // use the original ID
	s.Require().Equal(eResp.Email.Address.String, nu2.EmailAddress.String) // but the confirmed account email address
}

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	pb "github.com/DIMO-Network/shared/api/devices"
	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/users-api/internal/database"
	"github.com/DIMO-Network/users-api/internal/services"
	"github.com/DIMO-Network/users-api/models"
	"github.com/docker/go-connections/nat"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
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
		Image:        "postgres:12.11-alpine",
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

	err = database.MigrateDatabase(logger, &dbset, "", "../../migrations")
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

type udsc struct{}

func (c *udsc) GetUserDevice(ctx context.Context, in *pb.GetUserDeviceRequest, opts ...grpc.CallOption) (*pb.UserDevice, error) {
	return nil, nil
}

func (c *udsc) ListUserDevicesForUser(ctx context.Context, in *pb.ListUserDevicesForUserRequest, opts ...grpc.CallOption) (*pb.ListUserDevicesForUserResponse, error) {
	return &pb.ListUserDevicesForUserResponse{UserDevices: []*pb.UserDevice{}}, nil
}

type adsc struct{}

func (c *adsc) ListAftermarketDevicesForUser(ctx context.Context, in *pb.ListAftermarketDevicesForUserRequest, opts ...grpc.CallOption) (*pb.ListAftermarketDevicesForUserResponse, error) {
	return &pb.ListAftermarketDevicesForUserResponse{AftermarketDevices: []*pb.AftermarketDevice{}}, nil
}

func (s *UserControllerTestSuite) TestSubmitChallenge() {
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

	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	app.Get("/", uc.GetUser)
	app.Put("/", uc.UpdateUser)
	app.Post("/generate-challenge", uc.GenerateEthereumChallenge)
	app.Post("/submit-challenge", uc.SubmitEthereumChallenge)

	req := fmt.Sprintf(`{"web3": {"address": %q}}`, address)

	r := httptest.NewRequest("PUT", "/", strings.NewReader(req))
	r.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(r, -1)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Require().Equal(200, resp.StatusCode)

	r = httptest.NewRequest("POST", "/generate-challenge", nil)

	resp, err = app.Test(r, -1)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Require().Equal(200, resp.StatusCode)

	var chall struct{ Challenge string }
	err = json.NewDecoder(resp.Body).Decode(&chall)
	s.Require().NoError(err)

	toSign := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(chall.Challenge), chall.Challenge)

	hash := crypto.Keccak256([]byte(toSign))
	sig, err := crypto.Sign(hash, privateKey)
	s.Require().NoError(err)

	sig[64] += 27

	req = fmt.Sprintf(`{"signature": %q}`, hexutil.Encode(sig))

	r = httptest.NewRequest("POST", "/submit-challenge", strings.NewReader(req))
	r.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	s.Require().Equal(204, resp.StatusCode)

	r = httptest.NewRequest("GET", "/", nil)
	resp, err = app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()
}

func (s *UserControllerTestSuite) TestGenerateReferralCode() {
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

	code, err := uc.GenerateReferralCode(ctx)
	s.NoError(err)

	s.Regexp(referralCodeRegex, code)
}

func (s *UserControllerTestSuite) TestConfirmingAddressGeneratesReferralCode() {
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

	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	app.Get("/", uc.GetUser)
	app.Put("/", uc.UpdateUser)
	app.Post("/generate-challenge", uc.GenerateEthereumChallenge)
	app.Post("/submit-challenge", uc.SubmitEthereumChallenge)

	req := fmt.Sprintf(`{"web3": {"address": %q}}`, address)

	r := httptest.NewRequest("PUT", "/", strings.NewReader(req))
	r.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	s.Require().Equal(200, resp.StatusCode)

	r = httptest.NewRequest("POST", "/generate-challenge", nil)

	resp, err = app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	s.Require().Equal(200, resp.StatusCode)

	var chall struct{ Challenge string }
	err = json.NewDecoder(resp.Body).Decode(&chall)
	s.Require().NoError(err)

	toSign := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(chall.Challenge), chall.Challenge)

	hash := crypto.Keccak256([]byte(toSign))
	sig, err := crypto.Sign(hash, privateKey)
	s.Require().NoError(err)

	sig[64] += 27

	req = fmt.Sprintf(`{"signature": %q}`, hexutil.Encode(sig))

	r = httptest.NewRequest("POST", "/submit-challenge", strings.NewReader(req))
	r.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	s.Require().Equal(204, resp.StatusCode)

	r = httptest.NewRequest("GET", "/", nil)
	resp, err = app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	var user UserResponse
	err = json.NewDecoder(resp.Body).Decode(&user)
	s.Require().NoError(err)

	s.Regexp(referralCodeRegex, user.ReferralCode.String)
}

func (s *UserControllerTestSuite) TestNoReferralCodeWithoutEthereumAddress() {
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
			"provider_id": "google",
			"sub":         "Cwbs",
			"email":       "steve@gmail.com",
		}})
		return c.Next()
	})

	app.Get("/", uc.GetUser)

	r := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	var user UserResponse
	err = json.NewDecoder(resp.Body).Decode(&user)
	s.Require().NoError(err)

	s.Empty(user.ReferralCode)
}

func (s *UserControllerTestSuite) TestReferralCodeGeneratedOnWeb3Provider() {
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

	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &jwt.Token{Claims: jwt.MapClaims{
			"provider_id":      "web3",
			"sub":              "Cwbss",
			"email":            "steve@web3.com",
			"ethereum_address": address.Hex(),
		}})
		return c.Next()
	})

	app.Get("/", uc.GetUser)

	r := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	var user UserResponse
	err = json.NewDecoder(resp.Body).Decode(&user)
	s.Require().NoError(err)

	s.Regexp(referralCodeRegex, user.ReferralCode.String)
}

func (s *UserControllerTestSuite) TestSubmitReferralCode() {
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
			"provider_id": "google",
			"sub":         "Cwbss",
			"email":       "steve@web3.com",
		}})
		return c.Next()
	})

	app.Post("/submit-referral-code", uc.SubmitReferralCode)

	pk, err := crypto.GenerateKey()
	s.Require().NoError(err)

	addr := crypto.PubkeyToAddress(pk.PublicKey)
	nu := models.User{
		ID:              "SomeID",
		EmailConfirmed:  true,
		CreatedAt:       time.Now(),
		ReferralCode:    null.StringFrom("123456"),
		EthereumAddress: null.StringFrom(addr.Hex()),
	}

	err = nu.Insert(ctx, uc.dbs.DBS().Writer, boil.Infer())
	s.Require().NoError(err)

	req := `{"referralCode": "123456"}`

	r := httptest.NewRequest("POST", "/submit-referral-code", strings.NewReader(req))
	r.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	eResp := SubmitReferralCodeResponse{}
	err = json.NewDecoder(resp.Body).Decode(&eResp)
	s.Require().NoError(err)

	s.Require().Equal(200, resp.StatusCode)

	user, err := models.FindUser(ctx, uc.dbs.DBS().Reader, "Cwbss")
	s.Require().NoError(err)

	s.Require().Equal(null.StringFrom("123456"), user.ReferredBy)
}

func (s *UserControllerTestSuite) TestUserCannotRecommendSelf() {
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

	mockRefCode := "123456"

	nu := models.User{
		ID:             "Cwbss",
		EmailAddress:   null.StringFrom("steve@web3.com"),
		EmailConfirmed: true,
		CreatedAt:      time.Now(),
		ReferralCode:   null.StringFrom(mockRefCode),
	}

	err := nu.Insert(ctx, uc.dbs.DBS().Writer, boil.Infer())
	s.Require().NoError(err)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &jwt.Token{Claims: jwt.MapClaims{
			"provider_id": "google",
			"sub":         "Cwbss",
			"email":       "steve@web3.com",
		}})
		return c.Next()
	})

	app.Post("/submit-referral-code", uc.SubmitReferralCode)

	req := fmt.Sprintf(`{"referralCode": %q}`, mockRefCode)

	r := httptest.NewRequest("POST", "/submit-referral-code", strings.NewReader(req))
	r.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	eResp := SubmitReferralCodeResponse{}
	err = json.NewDecoder(resp.Body).Decode(&eResp)
	s.Require().Error(err)

	s.Require().Equal(400, resp.StatusCode)

	user, err := models.FindUser(ctx, uc.dbs.DBS().Reader, "Cwbss")
	s.Require().NoError(err)

	s.Require().Empty(user.ReferredBy)
}

func (s *UserControllerTestSuite) TestFailureOnReferralCodeNotExist() {
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

	mockRefCode := "123456"

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &jwt.Token{Claims: jwt.MapClaims{
			"provider_id": "google",
			"sub":         "Cwbss",
			"email":       "steve@web3.com",
		}})
		return c.Next()
	})

	app.Post("/submit-referral-code", uc.SubmitReferralCode)

	req := fmt.Sprintf(`{"referralCode": %q}`, mockRefCode)

	r := httptest.NewRequest("POST", "/submit-referral-code", strings.NewReader(req))
	r.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	eResp := SubmitReferralCodeResponse{}
	err = json.NewDecoder(resp.Body).Decode(&eResp)
	s.Require().Error(err)

	s.Require().Equal(400, resp.StatusCode)

	user, err := models.FindUser(ctx, uc.dbs.DBS().Reader, "Cwbss")
	s.Require().NoError(err)

	s.Require().Empty(user.ReferredBy)
}

func (s *UserControllerTestSuite) TestFailureOnInvalidReferralCode() {
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

	mockRefCode := "1234"

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &jwt.Token{Claims: jwt.MapClaims{
			"provider_id": "google",
			"sub":         "Cwbss",
			"email":       "steve@web3.com",
		}})
		return c.Next()
	})

	app.Post("/submit-referral-code", uc.SubmitReferralCode)

	req := fmt.Sprintf(`{"referralCode": %q}`, mockRefCode)

	r := httptest.NewRequest("POST", "/submit-referral-code", strings.NewReader(req))
	r.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	eResp := SubmitReferralCodeResponse{}
	err = json.NewDecoder(resp.Body).Decode(&eResp)
	s.Require().Error(err)

	s.Require().Equal(400, resp.StatusCode)

	user, err := models.FindUser(ctx, uc.dbs.DBS().Reader, "Cwbss")
	s.Require().NoError(err)

	s.Require().Empty(user.ReferredBy)
}

func (s *UserControllerTestSuite) TestFailureOnUserAlreadyReferred() {
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

	mockRefCode := "789102"

	nu := models.User{
		ID:             "Cwbss",
		EmailAddress:   null.StringFrom("steve@web3.com"),
		EmailConfirmed: true,
		CreatedAt:      time.Now(),
		ReferredBy:     null.StringFrom(mockRefCode),
	}

	err := nu.Insert(ctx, uc.dbs.DBS().Writer, boil.Infer())
	s.Require().NoError(err)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &jwt.Token{Claims: jwt.MapClaims{
			"provider_id": "google",
			"sub":         "Cwbss",
			"email":       "steve@web3.com",
		}})
		return c.Next()
	})

	app.Post("/submit-referral-code", uc.SubmitReferralCode)

	req := fmt.Sprintf(`{"referralCode": %q}`, "123456")

	r := httptest.NewRequest("POST", "/submit-referral-code", strings.NewReader(req))
	r.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(r, -1)
	s.Require().NoError(err)

	defer resp.Body.Close()

	eResp := SubmitReferralCodeResponse{}
	err = json.NewDecoder(resp.Body).Decode(&eResp)
	s.Require().Error(err)

	s.Require().Equal(400, resp.StatusCode)

	user, err := models.FindUser(ctx, uc.dbs.DBS().Reader, "Cwbss")
	s.Require().NoError(err)

	s.Require().Equal(user.ReferredBy, null.StringFrom(mockRefCode))
}

func (s *UserControllerTestSuite) TestFailureOnSameEthereumAddressForReferrerAndReferred() {

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

	mockRefCode := "789102"

	pk, err := crypto.GenerateKey()
	s.Require().NoError(err)

	addr := crypto.PubkeyToAddress(pk.PublicKey)

	nu := models.User{
		ID:              "Cwbss",
		EmailAddress:    null.StringFrom("steve@web3.com"),
		EmailConfirmed:  true,
		CreatedAt:       time.Now(),
		EthereumAddress: null.StringFrom(addr.Hex()),
	}

	err = nu.Insert(ctx, uc.dbs.DBS().Writer, boil.Infer())
	s.Require().NoError(err)

	nu2 := models.User{
		ID:              "Xwbzz",
		EmailAddress:    null.StringFrom("steve2@web3.com"),
		EmailConfirmed:  true,
		CreatedAt:       time.Now(),
		EthereumAddress: null.StringFrom(addr.Hex()),
		ReferralCode:    null.StringFrom(mockRefCode),
	}

	err = nu2.Insert(ctx, uc.dbs.DBS().Writer, boil.Infer())
	s.Require().NoError(err)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &jwt.Token{Claims: jwt.MapClaims{
			"provider_id": "google",
			"sub":         "Cwbss",
			"email":       "steve@web3.com",
		}})
		return c.Next()
	})

	app.Post("/submit-referral-code", uc.SubmitReferralCode)

	req := fmt.Sprintf(`{"referralCode": %q}`, mockRefCode)

	r := httptest.NewRequest("POST", "/submit-referral-code", strings.NewReader(req))
	r.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(r, -1)
	s.Require().NoError(err)

	body, _ := io.ReadAll(resp.Body)

	defer resp.Body.Close()

	s.Require().Equal(400, resp.StatusCode)
	s.Require().Equal("invalid referral code, user cannot refer self", string(body))
}

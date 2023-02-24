package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	pb "github.com/DIMO-Network/shared/api/devices"
	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/users-api/internal/database"
	"github.com/DIMO-Network/users-api/internal/services"
	"github.com/docker/go-connections/nat"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
)

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

func TestSubmitChallenge(t *testing.T) {
	ctx := context.Background()

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
	cont, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}

	defer cont.Terminate(ctx) //nolint

	logger := zerolog.Nop()

	host, err := cont.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mport, err := cont.MappedPort(ctx, nat.Port(nport))
	if err != nil {
		t.Fatal(err)
	}

	dbset := db.Settings{
		User:               "postgres",
		Password:           "postgres",
		Port:               mport.Port(),
		Host:               host,
		Name:               "users_api",
		MaxOpenConnections: 10,
		MaxIdleConnections: 10,
	}

	if err := database.MigrateDatabase(logger, &dbset, "", "../../migrations"); err != nil {
		t.Fatal(err)
	}

	dbs := db.NewDbConnectionFromSettings(ctx, &dbset, true)
	dbs.WaitForDB(logger)

	uc := UserController{
		dbs:             dbs,
		log:             &logger,
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
	if err != nil {
		t.Fatal(err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	app.Get("/", uc.GetUser)
	app.Put("/", uc.UpdateUser)
	app.Post("/generate-challenge", uc.GenerateEthereumChallenge)
	app.Post("/submit-challenge", uc.SubmitEthereumChallenge)

	s := fmt.Sprintf(`{"web3": {"address": %q}}`, address)

	r := httptest.NewRequest("PUT", "/", strings.NewReader(s))
	r.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(r, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatal(err)
	}

	r = httptest.NewRequest("POST", "/generate-challenge", nil)

	resp, err = app.Test(r, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Got status code %d when generating challenge", resp.StatusCode)
	}

	var chall struct{ Challenge string }
	if err := json.NewDecoder(resp.Body).Decode(&chall); err != nil {
		t.Fatal(err)
	}

	toSign := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(chall.Challenge), chall.Challenge)

	hash := crypto.Keccak256([]byte(toSign))
	sig, err := crypto.Sign(hash, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	sig[64] += 27

	s = fmt.Sprintf(`{"signature": %q}`, hexutil.Encode(sig))

	r = httptest.NewRequest("POST", "/submit-challenge", strings.NewReader(s))
	r.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(r, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		t.Fatal("BAD")
	}

	r = httptest.NewRequest("GET", "/", nil)
	resp, err = app.Test(r, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Assert things about the body.
}

type testDependencies struct {
	dbs       db.Store
	logger    *zerolog.Logger
	ctx       context.Context
	assert    *assert.Assertions
	container testcontainers.Container
}

func TestReferralCodeIsGenerated(t *testing.T) {
	ctx := context.Background()

	c := prepareTestDependencies(t, ctx)

	defer c.container.Terminate(ctx) //nolint

	uc := UserController{
		dbs:             c.dbs,
		log:             c.logger,
		allowedLateness: 5 * time.Minute,
		countryCodes:    []string{"USA", "CAN"},
		emailTemplate:   nil,
		eventService:    &es{},
		devicesClient:   &udsc{},
		amClient:        &adsc{},
	}

	code, err := uc.generateReferralCode(ctx, nil)
	c.assert.NoError(err)

	c.assert.NotEmpty(code)
}

func TestReferralCodeIsGeneratedCorrectlyOnConstraint(t *testing.T) {
	// We can limit max to 2
	// Which will only generate btw 1 and 2
	// and save all other combinations except 1, then check if that 1 is generated
}

func prepareTestDependencies(t *testing.T, ctx context.Context) testDependencies {
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
	cont, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// defer cont.Terminate(ctx) //nolint

	logger := zerolog.Nop()

	host, err := cont.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mport, err := cont.MappedPort(ctx, nat.Port(nport))
	if err != nil {
		t.Fatal(err)
	}

	dbset := db.Settings{
		User:               "postgres",
		Password:           "postgres",
		Port:               mport.Port(),
		Host:               host,
		Name:               "users_api",
		MaxOpenConnections: 10,
		MaxIdleConnections: 10,
	}

	if err := database.MigrateDatabase(logger, &dbset, "", "../../migrations"); err != nil {
		t.Fatal(err)
	}

	dbs := db.NewDbConnectionFromSettings(ctx, &dbset, true)
	dbs.WaitForDB(logger)

	assert := assert.New(t)

	return testDependencies{
		dbs:       dbs,
		logger:    &logger,
		ctx:       ctx,
		assert:    assert,
		container: cont,
	}
}

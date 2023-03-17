package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/users-api/internal/config"
	"github.com/DIMO-Network/users-api/internal/database"
	"github.com/DIMO-Network/users-api/models"
	"github.com/docker/go-connections/nat"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type PopulateReferralCodeSuite struct {
	suite.Suite
	dbcont   testcontainers.Container
	dbs      db.Store
	logger   *zerolog.Logger
	settings *config.Settings
}

func TestPopulateReferralCodeSuite(t *testing.T) {
	suite.Run(t, &PopulateReferralCodeSuite{})
}

func (s *PopulateReferralCodeSuite) SetupSuite() {
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

	s.settings = &config.Settings{}
}

func (s *PopulateReferralCodeSuite) TearDownSuite() {
	s.Require().NoError(s.dbcont.Terminate(context.Background()))
}

func (s *PopulateReferralCodeSuite) TearDownTest() {
	_, err := models.Users().DeleteAll(context.Background(), s.dbs.DBS().Writer)
	s.Require().NoError(err)
}

func (s *PopulateReferralCodeSuite) TestGenerateReferralCodeForUsers() {
	ctx := context.Background()

	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	nu := models.User{
		ID:              "SomeID",
		EmailConfirmed:  true,
		CreatedAt:       time.Now(),
		EthereumAddress: null.StringFrom(address.Hex()),
	}

	nu.Insert(ctx, s.dbs.DBS().Writer, boil.Infer())

	pp := &populateReferralCodeCmd{
		dbs:      s.dbs,
		log:      s.logger,
		ctx:      ctx,
		Settings: s.settings,
	}

	pp.Execute()

	res, err := models.Users().One(ctx, s.dbs.DBS().Reader)
	s.NoError(err)

	s.NotEmpty(res.ReferralCode)
}

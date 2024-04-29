package api

import (
	"context"
	"testing"

	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/users-api/internal/database"
	"github.com/DIMO-Network/users-api/models"
	userpb "github.com/DIMO-Network/users-api/pkg/grpc"
	"github.com/docker/go-connections/nat"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type UserServiceTestSuite struct {
	suite.Suite
	dbcont testcontainers.Container
	dbs    db.Store
	logger *zerolog.Logger
}

func TestUserServiceSuite(t *testing.T) {
	suite.Run(t, &UserServiceTestSuite{})
}

func (s *UserServiceTestSuite) SetupSuite() {
	ctx := context.Background()

	logger := zerolog.Nop()
	s.logger = &logger

	port := "5432/tcp"
	req := testcontainers.ContainerRequest{
		Image:        "postgres:12.11-alpine",
		ExposedPorts: []string{port},
		Env: map[string]string{
			"POSTGRES_DB":       "users_api_test",
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "password",
		},
		WaitingFor: wait.ForListeningPort(nat.Port(port)),
	}
	dbcont, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	s.Require().NoError(err)
	s.dbcont = dbcont

	host, err := dbcont.Host(ctx)
	s.Require().NoError(err)

	mport, err := dbcont.MappedPort(ctx, nat.Port(port))
	s.Require().NoError(err)

	dbset := db.Settings{
		User:               "postgres",
		Password:           "password",
		Port:               mport.Port(),
		Host:               host,
		Name:               "users_api_test",
		MaxOpenConnections: 10,
		MaxIdleConnections: 10,
	}

	s.dbs = db.NewDbConnectionFromSettings(ctx, &dbset, true)
	s.dbs.WaitForDB(logger)

	err = database.MigrateDatabase(ctx, logger, &dbset, "", "../../migrations")
	s.Require().NoError(err)
}

func (s *UserServiceTestSuite) TearDownSuite() {
	s.Require().NoError(s.dbcont.Terminate(context.Background()))
}

func (s *UserServiceTestSuite) TestGetUserByEthAddr() {
	ctx := context.Background()
	userSvc := NewUserService(s.dbs, s.logger)

	ethAddr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	testUser := models.User{
		ID:                "TestUserID",
		EthereumAddress:   null.BytesFrom(ethAddr.Bytes()),
		EmailConfirmed:    true,
		EmailAddress:      null.StringFrom("testuser@example.com"),
		EthereumConfirmed: true,
	}

	err := testUser.Insert(ctx, s.dbs.DBS().Writer, boil.Infer())
	s.Require().NoError(err)

	req := &userpb.GetUserByEthRequest{
		EthAddr: ethAddr.Bytes(),
	}

	resultUser, err := userSvc.GetUserByEthAddr(ctx, req)
	s.Require().NoError(err)

	s.Require().NotNil(resultUser)
	s.Require().Equal("TestUserID", resultUser.Id)
	s.Require().Equal("testuser@example.com", *resultUser.EmailAddress)
	s.Require().Equal(ethAddr.Hex(), *resultUser.EthereumAddress)

	_, err = models.Users(models.UserWhere.ID.EQ("TestUserID")).DeleteAll(ctx, s.dbs.DBS().Writer)
	s.Require().NoError(err)
}

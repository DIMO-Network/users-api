package api

import (
	"context"
	"database/sql"
	"errors"

	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/users-api/models"
	pb "github.com/DIMO-Network/users-api/pkg/grpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewUserService(dbs db.Store, logger *zerolog.Logger) pb.UserServiceServer {
	return &userService{dbs: dbs, logger: logger}
}

type userService struct {
	pb.UnimplementedUserServiceServer
	dbs    db.Store
	logger *zerolog.Logger
}

func (s *userService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	dbUser, err := models.Users(
		models.UserWhere.ID.EQ(req.Id),
		qm.Load(models.UserRels.ReferringUser),
	).One(ctx, s.dbs.DBS().Reader)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "No user with that ID found.")
		}
		s.logger.Err(err).Str("userId", req.Id).Msg("Database failure retrieving user.")
		return nil, status.Error(codes.Internal, "Internal error.")
	}

	return formatUser(dbUser), nil
}

func formatUser(user *models.User) *pb.User {
	out := pb.User{
		Id: user.ID,
	}

	if user.EthereumConfirmed {
		hexAddress := common.BytesToAddress(user.EthereumAddress.Bytes).Hex()
		out.EthereumAddress = &hexAddress
		out.EthereumAddressBytes = user.EthereumAddress.Bytes
	}

	if user.EmailConfirmed {
		out.EmailAddress = user.EmailAddress.Ptr()
	}

	if user.ReferredAt.Valid {
		var pbRef pb.UserReferrer

		if ref := user.R.ReferringUser; ref != nil && ref.EthereumConfirmed {
			pbRef.ReferrerValid = true
			pbRef.EthereumAddress = ref.EthereumAddress.Bytes
			pbRef.Id = ref.ID
		}

		out.ReferredBy = &pbRef
	}

	return &out
}

func (s *userService) GetUserByEthAddr(ctx context.Context, req *pb.GetUserByEthRequest) (*pb.User, error) {
	dbUser, err := models.Users(
		models.UserWhere.EthereumAddress.EQ(null.BytesFrom(req.EthAddr)),
		models.UserWhere.EthereumConfirmed.EQ(true),
		qm.Load(models.UserRels.ReferringUser),
	).One(ctx, s.dbs.DBS().Reader)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "No user with that ID found.")
		}
		s.logger.Err(err).Str("ethAddr", common.BytesToAddress(req.EthAddr).Hex()).Msg("Database failure retrieving user.")
		return nil, status.Error(codes.Internal, "Internal error.")
	}

	return formatUser(dbUser), nil
}

func (s *userService) GetUsersByEthereumAddress(ctx context.Context, in *pb.GetUsersByEthereumAddressRequest) (*pb.GetUsersByEthereumAddressResponse, error) {
	users, err := models.Users(
		models.UserWhere.EthereumConfirmed.EQ(true),
		models.UserWhere.EthereumAddress.EQ(null.BytesFrom(in.EthereumAddress)),
		qm.Load(models.UserRels.ReferringUser),
	).All(ctx, s.dbs.DBS().Reader)
	if err != nil {
		return nil, err
	}

	var out pb.GetUsersByEthereumAddressResponse

	for _, u := range users {
		out.Users = append(out.Users, formatUser(u))
	}

	return &out, nil
}

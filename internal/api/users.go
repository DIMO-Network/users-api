package api

import (
	"context"
	"database/sql"

	"errors"
	"github.com/volatiletech/null/v8"

	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/users-api/models"
	pb "github.com/DIMO-Network/users-api/pkg/grpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
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

	pbUser := &pb.User{
		Id: dbUser.ID,
	}

	if dbUser.EthereumConfirmed {
		hexAddress := common.BytesToAddress(dbUser.EthereumAddress.Bytes).Hex()
		pbUser.EthereumAddress = &hexAddress
	}

	if dbUser.EmailConfirmed {
		pbUser.EmailAddress = dbUser.EmailAddress.Ptr()
	}

	if dbUser.ReferredAt.Valid {
		var pbRef pb.UserReferrer

		if ref := dbUser.R.ReferringUser; ref != nil && ref.EthereumConfirmed {
			pbRef.ReferrerValid = true
			pbRef.EthereumAddress = ref.EthereumAddress.Bytes
			pbRef.Id = ref.ID
		}

		pbUser.ReferredBy = &pbRef
	}

	return pbUser, nil
}

func (s *userService) GetUserByEthAddr(ctx context.Context, req *pb.GetUserByEthRequest) (*pb.User, error) {
	dbUser, err := models.Users(
		models.UserWhere.EthereumAddress.EQ(null.BytesFrom(req.EthAddr)),
		qm.Load(models.UserRels.ReferringUser),
	).One(ctx, s.dbs.DBS().Reader)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "No user with that ID found.")
		}
		s.logger.Err(err).Str("ethAddr", common.BytesToAddress(req.EthAddr).Hex()).Msg("Database failure retrieving user.")
		return nil, status.Error(codes.Internal, "Internal error.")
	}

	hexAddr := common.BytesToAddress(dbUser.EthereumAddress.Bytes).Hex()
	pbUser := &pb.User{
		EthereumAddress: &hexAddr,
		Id:              dbUser.ID, //should this eventually be deprecated?
	}

	if dbUser.EmailConfirmed {
		pbUser.EmailAddress = dbUser.EmailAddress.Ptr()
	}

	if dbUser.ReferredAt.Valid {
		var pbRef pb.UserReferrer

		if ref := dbUser.R.ReferringUser; ref != nil && ref.EthereumConfirmed {
			pbRef.ReferrerValid = true
			pbRef.EthereumAddress = ref.EthereumAddress.Bytes
			pbRef.Id = ref.ID
		}

		pbUser.ReferredBy = &pbRef
	}

	return pbUser, nil
}

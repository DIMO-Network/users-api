package api

import (
	"context"
	"database/sql"
	"errors"

	pb "github.com/DIMO-Network/shared/api/users"
	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/users-api/models"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/volatiletech/null/v8"
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
	dbUser, err := models.FindUser(ctx, s.dbs.DBS().Reader, req.Id)
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
		pbUser.EthereumAddress = dbUser.EthereumAddress.Ptr()
	}

	if dbUser.EmailConfirmed {
		pbUser.EmailAddress = dbUser.EmailAddress.Ptr()
	}

	if dbUser.ReferredBy.Valid {
		referrer, err := models.Users(models.UserWhere.ReferralCode.EQ(null.StringFrom(dbUser.ReferredBy.String))).One(ctx, s.dbs.DBS().Reader)
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, status.Error(codes.Internal, "Internal error.")
			}
		}

		pbUser.ReferredBy = &pb.UserReferrer{
			EthereumAddress: common.FromHex(referrer.EthereumAddress.String),
		}

		if referrer.EthereumConfirmed {
			pbUser.ReferredBy.ReferrerValid = true
		}

	}

	return pbUser, nil
}

package api

import (
	"context"
	"database/sql"
	"errors"

	pb "github.com/DIMO-Network/shared/api/users"
	"github.com/DIMO-Network/users-api/internal/database"
	"github.com/DIMO-Network/users-api/models"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewUserService(dbs func() *database.DBReaderWriter, logger *zerolog.Logger) pb.UserServiceServer {
	return &userService{dbs: dbs, logger: logger}
}

type userService struct {
	pb.UnimplementedUserServiceServer
	dbs    func() *database.DBReaderWriter
	logger *zerolog.Logger
}

func (s *userService) GetUserDevice(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	dbUser, err := models.FindUser(ctx, s.dbs().Reader, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "No device with that ID found.")
		}
		s.logger.Err(err).Str("userId", req.Id).Msg("Database failure retrieving user.")
		return nil, status.Error(codes.Internal, "Internal error.")
	}
	pbDevice := &pb.User{
		Id: dbUser.ID,
	}
	if dbUser.EthereumConfirmed {
		pbDevice.EthereumAddress = dbUser.EthereumAddress.Ptr()
	}
	return pbDevice, nil
}

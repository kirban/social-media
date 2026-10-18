// Package identityserver adapts internal/service's UserServiceInterface to
// the generated identityv1.IdentityServiceServer gRPC contract. All identity
// business logic stays in internal/service/user.go, untouched — this is a
// pure translation layer between model.* and the proto types.
package identityserver

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "github.com/kirban/social-media/api/proto/gen/identity/v1"
	"github.com/kirban/social-media/internal/model"
	"github.com/kirban/social-media/internal/service"
)

type Server struct {
	identityv1.UnimplementedIdentityServiceServer
	svc service.UserServiceInterface
}

func New(svc service.UserServiceInterface) *Server {
	return &Server{svc: svc}
}

func (s *Server) Register(ctx context.Context, req *identityv1.RegisterRequest) (*identityv1.RegisterResponse, error) {
	id, err := s.svc.Register(ctx, req.GetFirstName(), req.GetSecondName(), req.GetPassword(), req.Birthdate, req.GetBiography(), req.GetCity())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &identityv1.RegisterResponse{Id: id}, nil
}

func (s *Server) Login(ctx context.Context, req *identityv1.LoginRequest) (*identityv1.LoginResponse, error) {
	token, err := s.svc.Login(ctx, req.GetId(), req.GetPassword())
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &identityv1.LoginResponse{Token: token}, nil
}

func (s *Server) GetUser(ctx context.Context, req *identityv1.GetUserRequest) (*identityv1.GetUserResponse, error) {
	u, err := s.svc.GetByID(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &identityv1.GetUserResponse{User: toProtoUser(u)}, nil
}

func (s *Server) SearchUsers(ctx context.Context, req *identityv1.SearchUsersRequest) (*identityv1.SearchUsersResponse, error) {
	users, err := s.svc.SearchByNames(ctx, req.GetFirstName(), req.GetSecondName())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &identityv1.SearchUsersResponse{}
	if users != nil {
		resp.Users = make([]*identityv1.User, len(*users))
		for i := range *users {
			resp.Users[i] = toProtoUser(&(*users)[i])
		}
	}
	return resp, nil
}

func toProtoUser(u *model.User) *identityv1.User {
	return &identityv1.User{
		Id:         u.ID,
		FirstName:  u.FirstName,
		SecondName: u.SecondName,
		Birthdate:  u.Birthdate,
		Biography:  u.Biography,
		City:       u.City,
	}
}

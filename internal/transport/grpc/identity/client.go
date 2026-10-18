package identity_grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "github.com/kirban/social-media/api/proto/gen/identity/v1"
	"github.com/kirban/social-media/internal/model"
	"github.com/kirban/social-media/internal/service"
)

// UserGRPCClient implements UserServiceInterface by calling the extracted
// identity-service over gRPC, mapping its status codes back onto the same
// sentinel errors (ErrNotFound, ErrUnauthorized) the in-process UserService
// returns, so errors.Is checks in internal/api behave identically either way.
type UserGRPCClient struct {
	client identityv1.IdentityServiceClient
}

// NewUserGRPCClient wraps an already-constructed identityv1.IdentityServiceClient.
// The caller dials the underlying connection (with the request-id-propagating
// client interceptor from internal/grpcinterceptor attached) and builds the
// client with identityv1.NewIdentityServiceClient.
func NewUserGRPCClient(client identityv1.IdentityServiceClient) *UserGRPCClient {
	return &UserGRPCClient{client: client}
}

var _ service.UserServiceInterface = (*UserGRPCClient)(nil)

func (c *UserGRPCClient) Register(ctx context.Context, firstName, secondName, password string, birthdate *string, biography, city string) (string, error) {
	resp, err := c.client.Register(ctx, &identityv1.RegisterRequest{
		FirstName:  firstName,
		SecondName: secondName,
		Password:   password,
		Birthdate:  birthdate,
		Biography:  biography,
		City:       city,
	})
	if err != nil {
		return "", fromGRPCStatus(err)
	}
	return resp.GetId(), nil
}

func (c *UserGRPCClient) Login(ctx context.Context, id, password string) (string, error) {
	resp, err := c.client.Login(ctx, &identityv1.LoginRequest{Id: id, Password: password})
	if err != nil {
		return "", fromGRPCStatus(err)
	}
	return resp.GetToken(), nil
}

func (c *UserGRPCClient) GetByID(ctx context.Context, id string) (*model.User, error) {
	resp, err := c.client.GetUser(ctx, &identityv1.GetUserRequest{Id: id})
	if err != nil {
		return nil, fromGRPCStatus(err)
	}
	return userFromProto(resp.GetUser()), nil
}

func (c *UserGRPCClient) SearchByNames(ctx context.Context, fname, lname string) (*[]model.User, error) {
	resp, err := c.client.SearchUsers(ctx, &identityv1.SearchUsersRequest{FirstName: fname, SecondName: lname})
	if err != nil {
		return nil, fromGRPCStatus(err)
	}

	users := make([]model.User, len(resp.GetUsers()))
	for i, u := range resp.GetUsers() {
		users[i] = *userFromProto(u)
	}
	return &users, nil
}

func userFromProto(u *identityv1.User) *model.User {
	return &model.User{
		ID:         u.GetId(),
		FirstName:  u.GetFirstName(),
		SecondName: u.GetSecondName(),
		Birthdate:  u.Birthdate,
		Biography:  u.GetBiography(),
		City:       u.GetCity(),
	}
}

// fromGRPCStatus maps a gRPC status error back onto the sentinel errors
// UserService returns in-process, so errors.Is checks in internal/api
// behave identically whether UserSvc is local or this gRPC adapter.
func fromGRPCStatus(err error) error {
	switch status.Code(err) {
	case codes.NotFound:
		return service.ErrNotFound
	case codes.Unauthenticated:
		return service.ErrUnauthorized
	default:
		return err
	}
}

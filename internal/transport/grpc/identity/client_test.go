package identity_grpc

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "github.com/kirban/social-media/api/proto/gen/identity/v1"
	"github.com/kirban/social-media/internal/service"
)

// fakeIdentityServiceClient implements identityv1.IdentityServiceClient with
// canned responses/errors, so UserGRPCClient's field mapping and error
// translation can be tested without a real gRPC connection.
type fakeIdentityServiceClient struct {
	registerResp *identityv1.RegisterResponse
	loginResp    *identityv1.LoginResponse
	loginErr     error
	getUserResp  *identityv1.GetUserResponse
	getUserErr   error
	searchResp   *identityv1.SearchUsersResponse

	lastRegisterReq *identityv1.RegisterRequest
	lastSearchReq   *identityv1.SearchUsersRequest
}

func (f *fakeIdentityServiceClient) Register(ctx context.Context, in *identityv1.RegisterRequest, opts ...grpc.CallOption) (*identityv1.RegisterResponse, error) {
	f.lastRegisterReq = in
	return f.registerResp, nil
}

func (f *fakeIdentityServiceClient) Login(ctx context.Context, in *identityv1.LoginRequest, opts ...grpc.CallOption) (*identityv1.LoginResponse, error) {
	return f.loginResp, f.loginErr
}

func (f *fakeIdentityServiceClient) GetUser(ctx context.Context, in *identityv1.GetUserRequest, opts ...grpc.CallOption) (*identityv1.GetUserResponse, error) {
	return f.getUserResp, f.getUserErr
}

func (f *fakeIdentityServiceClient) SearchUsers(ctx context.Context, in *identityv1.SearchUsersRequest, opts ...grpc.CallOption) (*identityv1.SearchUsersResponse, error) {
	f.lastSearchReq = in
	return f.searchResp, nil
}

func TestUserGRPCClient_Register_BirthdateNilability(t *testing.T) {
	fake := &fakeIdentityServiceClient{registerResp: &identityv1.RegisterResponse{Id: "u1"}}
	c := NewUserGRPCClient(fake)

	id, err := c.Register(context.Background(), "Jane", "Doe", "pw", nil, "bio", "city")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "u1" {
		t.Fatalf("id = %q, want %q", id, "u1")
	}
	if fake.lastRegisterReq.Birthdate != nil {
		t.Fatalf("Birthdate = %v, want nil", fake.lastRegisterReq.Birthdate)
	}

	bday := "1990-01-01"
	_, err = c.Register(context.Background(), "Jane", "Doe", "pw", &bday, "bio", "city")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.lastRegisterReq.Birthdate == nil || *fake.lastRegisterReq.Birthdate != bday {
		t.Fatalf("Birthdate = %v, want %q", fake.lastRegisterReq.Birthdate, bday)
	}
}

func TestUserGRPCClient_Login_MapsUnauthenticatedToErrUnauthorized(t *testing.T) {
	fake := &fakeIdentityServiceClient{loginErr: status.Error(codes.Unauthenticated, "unauthorized")}
	c := NewUserGRPCClient(fake)

	_, err := c.Login(context.Background(), "u1", "wrong")
	if !errors.Is(err, service.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestUserGRPCClient_GetByID_MapsNotFoundToErrNotFound(t *testing.T) {
	fake := &fakeIdentityServiceClient{getUserErr: status.Error(codes.NotFound, "not found")}
	c := NewUserGRPCClient(fake)

	_, err := c.GetByID(context.Background(), "missing")
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestUserGRPCClient_GetByID_PreservesBirthdateNilability(t *testing.T) {
	fake := &fakeIdentityServiceClient{getUserResp: &identityv1.GetUserResponse{
		User: &identityv1.User{Id: "u1", FirstName: "Jane", SecondName: "Doe", Biography: "bio", City: "city"},
	}}
	c := NewUserGRPCClient(fake)

	u, err := c.GetByID(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Birthdate != nil {
		t.Fatalf("Birthdate = %v, want nil", u.Birthdate)
	}
}

func TestUserGRPCClient_SearchByNames(t *testing.T) {
	fake := &fakeIdentityServiceClient{searchResp: &identityv1.SearchUsersResponse{
		Users: []*identityv1.User{
			{Id: "u1", FirstName: "Jane", SecondName: "Doe"},
			{Id: "u2", FirstName: "Jane", SecondName: "Roe"},
		},
	}}
	c := NewUserGRPCClient(fake)

	users, err := c.SearchByNames(context.Background(), "Jane", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(*users))
	}
	if fake.lastSearchReq.GetFirstName() != "Jane" {
		t.Fatalf("FirstName = %q, want %q", fake.lastSearchReq.GetFirstName(), "Jane")
	}
}

func TestUserGRPCClient_OtherErrorsPassThrough(t *testing.T) {
	fake := &fakeIdentityServiceClient{getUserErr: status.Error(codes.Internal, "boom")}
	c := NewUserGRPCClient(fake)

	_, err := c.GetByID(context.Background(), "u1")
	if errors.Is(err, service.ErrNotFound) || errors.Is(err, service.ErrUnauthorized) {
		t.Fatalf("err = %v, want neither sentinel mapped", err)
	}
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
}

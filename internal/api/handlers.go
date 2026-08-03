package api

import (
	"github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/service"
)

//go:generate go tool oapi-codegen -config ../../oapi-codegen.yaml -o api.gen.go ../../docs/openapi.json

// Handlers implements ServerInterface. Embed Unimplemented so adding new
// endpoints to the spec doesn't break the build until they are wired up.
//
// The flip side: renaming an operation in the spec silently falls back to
// Unimplemented's 501 rather than failing the build, because the old method
// still compiles as dead code. After regenerating, check that every route you
// expect to work still has a handler here — the compiler will not tell you.
//
// Never hand-edit api.gen.go. Operation names come from the spec's operationId
// (falling back to a path-derived name), so an edit there is lost on the next
// regeneration; set operationId instead.
type Handlers struct {
	Unimplemented
	Logger     *logger.AppLogger
	UserSvc    *service.UserService
	PostSvc    *service.PostsService
	FriendsSvc *service.FriendsService
	DialogSvc  *service.DialogService
}

var _ ServerInterface = (*Handlers)(nil)

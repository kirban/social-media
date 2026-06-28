package api

import (
	"github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/service"
)

// Handlers implements ServerInterface. Embed Unimplemented so adding new
// endpoints to the spec doesn't break the build until they are wired up.
type Handlers struct {
	Unimplemented
	Logger  *logger.AppLogger
	UserSvc *service.UserService
}

var _ ServerInterface = (*Handlers)(nil)

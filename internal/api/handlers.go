package api

import (
	"github.com/kirban/social-media/internal/db"
	"github.com/kirban/social-media/internal/logger"
)

// Handlers implements ServerInterface. Embed Unimplemented so adding new
// endpoints to the spec doesn't break the build until they are wired up.
type Handlers struct {
	Unimplemented // temporary stub for unimplemented methods
	Db            *db.DB
	Logger        *logger.AppLogger
}

var _ ServerInterface = (*Handlers)(nil)

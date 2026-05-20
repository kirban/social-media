package api

// Handlers implements ServerInterface. Embed Unimplemented so adding new
// endpoints to the spec doesn't break the build until they are wired up.
type Handlers struct {
	Unimplemented
}

var _ ServerInterface = (*Handlers)(nil)

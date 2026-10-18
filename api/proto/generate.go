// Package proto holds the gRPC service contracts (api/proto/*/v1/*.proto)
// for the extracted Dialog and Identity services, and the Go code generated
// from them under api/proto/gen/. Regenerate with `go generate ./api/proto/`
// after editing a .proto file.
//
//go:generate ./generate.sh
package proto

#!/usr/bin/env bash
# Regenerates api/proto/gen/** from the .proto files in this tree.
#
# protoc-gen-go and protoc-gen-go-grpc are declared as `tool` dependencies in
# go.mod (same pattern as oapi-codegen/goose), but protoc invokes codegen
# plugins as plain executables, not via `go tool`, so we build them once into
# a throwaway bin dir and point protoc at them with --plugin.
set -euo pipefail
cd "$(dirname "$0")/../.."

bin="$(mktemp -d)"
trap 'rm -rf "$bin"' EXIT

go build -o "$bin/protoc-gen-go" google.golang.org/protobuf/cmd/protoc-gen-go
go build -o "$bin/protoc-gen-go-grpc" google.golang.org/grpc/cmd/protoc-gen-go-grpc

protoc \
  --plugin=protoc-gen-go="$bin/protoc-gen-go" \
  --plugin=protoc-gen-go-grpc="$bin/protoc-gen-go-grpc" \
  --go_out=. --go_opt=module=github.com/kirban/social-media \
  --go-grpc_out=. --go-grpc_opt=module=github.com/kirban/social-media \
  api/proto/dialog/v1/dialog.proto \
  api/proto/identity/v1/identity.proto

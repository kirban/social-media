# Gateway: the remaining monolith (Friends, Post/Feed in-process; Dialog and
# Identity forwarded over gRPC when SERVICE_MODE env vars say so — see
# internal/config.ServicesConfig). Same two-stage shape as
# build/package/Dockerfile, which other compose variants still use unchanged.
FROM golang:1.26.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/gateway ./cmd/server

FROM alpine:3.20

ARG CFG_PATH=configs/gateway-config-docker.yaml
ENV CFG_PATH=${CFG_PATH}

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/gateway .
COPY ${CFG_PATH} ./configs/app-config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/gateway"]

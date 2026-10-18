# identity-service: the extracted User/Identity domain as a standalone gRPC
# server (cmd/identity-service). Same two-stage shape as build/package/Dockerfile.
FROM golang:1.26.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/identity-service ./cmd/identity-service

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/identity-service .
COPY configs/identity-service-config-docker.yaml ./configs/app-config.yaml

EXPOSE 50062

ENTRYPOINT ["/app/identity-service"]

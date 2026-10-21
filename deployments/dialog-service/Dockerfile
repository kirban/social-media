# dialog-service: the extracted Dialog domain as a standalone gRPC server
# (cmd/dialog-service). Same two-stage shape as build/package/Dockerfile.
FROM golang:1.26.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/dialog-service ./cmd/dialog-service

FROM alpine:3.20

ARG CFG_PATH=configs/dialog-service-config-docker.yaml
ENV CFG_PATH=${CFG_PATH}

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/dialog-service .
COPY ${CFG_PATH} ./configs/app-config.yaml

EXPOSE 50061

ENTRYPOINT ["/app/dialog-service"]

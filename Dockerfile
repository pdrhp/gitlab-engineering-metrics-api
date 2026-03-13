# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/gitlab-engineering-metrics-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

COPY --from=builder /out/gitlab-engineering-metrics-api /app/gitlab-engineering-metrics-api

ENV SERVER_ADDR=:8080

EXPOSE 8080

ENTRYPOINT ["/app/gitlab-engineering-metrics-api"]

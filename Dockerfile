# syntax=docker/dockerfile:1.21

ARG GO_VERSION="1.26"

# =============================================================================
FROM golang:${GO_VERSION}-alpine AS builder

SHELL ["/bin/sh", "-euc"]

WORKDIR /app

RUN --mount=type=bind,source=./go.mod,target=./go.mod \
    --mount=type=bind,source=./go.sum,target=./go.sum \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

ENV CGO_ENABLED=0

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    <<'EOD'
go build \
    -ldflags="-s -w" \
    -trimpath \
    -o statuspage-exporter \
    .
EOD

# =============================================================================
FROM docker.io/dier/distroless-static-debian13:nonroot-flatten AS release
COPY --from=builder --chown=nonroot:nonroot /app/statuspage-exporter /usr/local/bin/statuspage-exporter
USER nonroot:nonroot
ENTRYPOINT ["statuspage-exporter"]

# syntax=docker/dockerfile
# Install golang

# Build Stage
FROM golang:1.26-alpine3.23 AS builder

ARG VERSION=dev
ENV CGO_ENABLED=0 \
    GOFLAGS="-trimpath" \
    LDFLAGS="-s -w -X main.version=${VERSION}" \
    WRK_DIR=/app

# Set working directory
WORKDIR $WRK_DIR

# Better layer caching for deps
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy the contents to /app
COPY . $WRK_DIR

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags "${LDFLAGS}" -o $WRK_DIR/appbin $WRK_DIR/cmd

COPY docker /app/docker

#
# Run Stage
#
FROM alpine:3.24

LABEL org.opencontainers.image.authors="CZERTAINLY <support@czertainly.com>"

# add non root user czertainly
RUN apk upgrade --no-cache \
    && addgroup --system --gid 10001 czertainly \
    && adduser --system --home /opt/czertainly --uid 10001 --ingroup czertainly czertainly

COPY --from=builder /app/docker /
COPY --from=builder /app /opt/czertainly

WORKDIR /opt/czertainly

ENV SERVER_PORT=8080
ENV LOG_LEVEL=INFO

USER 10001

ENTRYPOINT ["/opt/czertainly/entry.sh"]

# syntax=docker/dockerfile:1

# ==============================
# Build
# ==============================

FROM golang:1.26-alpine AS build

WORKDIR /src

# Download dependencies in their own layer so code changes reuse the cache.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binaries: no cgo, so they run on a plain Alpine image.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# ==============================
# Runtime
# ==============================

FROM alpine:3.22

# CA certificates for TLS to Resend and to managed PostgreSQL. The timezone
# database is embedded in the binaries (time/tzdata), so tzdata is not needed.
RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 10001 app

WORKDIR /app

COPY --from=build /out/api /out/migrate ./

USER app

# Default port; hosts such as Render override it through PORT.
ENV PORT=8080
EXPOSE 8080

# Migrations run on every start, before the API, since hosts without a
# pre-deploy step (such as Render's free tier) have nowhere else to run them.
# They are idempotent and guarded by a database lock. exec makes the API
# PID 1, so it receives SIGTERM and shuts down gracefully.
CMD ["sh", "-c", "./migrate && exec ./api"]

# Multi-stage Dockerfile for Permission Slip.
# Produces a minimal (~30MB) runtime image containing only the static binary.
#
# Build: docker build --build-arg VITE_SUPABASE_URL=... --build-arg VITE_SUPABASE_PUBLISHABLE_KEY=... -t permission-slip .
# Run:   docker run -p 8080:8080 -e DATABASE_URL=... -e SUPABASE_URL=... permission-slip

# ── Stage 1: Build frontend ──────────────────────────────────────────────────
FROM node:25-alpine AS frontend

WORKDIR /app/frontend

# Install dependencies first (layer cache)
COPY frontend/package.json frontend/package-lock.json ./

# The OpenAPI spec is needed by the postinstall hook (openapi-typescript)
COPY spec/ /app/spec/

RUN npm ci

# Vite inlines VITE_* env vars into the JS bundle at build time.
# Pass these as Docker build args (--build-arg) or via [build.args] in fly.toml.
ARG VITE_SUPABASE_URL
ARG VITE_SUPABASE_PUBLISHABLE_KEY
ARG VITE_SENTRY_DSN
ARG VITE_POSTHOG_KEY
ARG VITE_POSTHOG_HOST
ARG VITE_STRIPE_PUBLISHABLE_KEY
ARG VITE_PERMISSION_SLIP_SAAS

# Sentry source-map upload (optional — skip if not set).
# @sentry/vite-plugin reads these during the Vite build to upload maps.
ARG SENTRY_AUTH_TOKEN
ARG SENTRY_ORG
ARG SENTRY_PROJECT

# Copy shared validation config (imported by frontend/src/lib/validation.ts)
COPY shared/ /app/shared/

# Copy config directory (frontend/src/config/plans.ts imports @config/plans.json)
COPY config/plans.json /app/config/plans.json

# Copy frontend source and build
COPY frontend/ ./
RUN npm run build

# ── Stage 2: Build Go binary ────────────────────────────────────────────────
# Pin to the exact patch version from go.mod's toolchain directive to avoid
# automatic toolchain downloads during build (which can fail in restricted envs).
FROM golang:1.26.2-alpine AS backend

RUN apk add --no-cache git

WORKDIR /app

# Download Go modules first (layer cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend into the embed path
COPY --from=frontend /app/frontend/dist ./frontend/dist

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /server .

# ── Stage 3: Minimal runtime ────────────────────────────────────────────────
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata

# Run as non-root
RUN adduser -D -h /app appuser
USER appuser
WORKDIR /app

COPY --from=backend /server ./server

EXPOSE 8080

# Alpine includes wget but not curl; use wget for the health check.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/health || exit 1

ENTRYPOINT ["./server"]

.PHONY: dev dev-backend dev-frontend build run install setup \
       test test-backend test-frontend test-integration typecheck \
       mobile-install mobile-start mobile-test mobile-typecheck \
       mobile-build-dev mobile-build-preview mobile-build-prod \
       mobile-build-dev-ios mobile-build-dev-android \
       mobile-build-preview-ios mobile-build-preview-android \
       mobile-submit mobile-update \
       generate-frontend generate-mobile \
       migrate-up migrate-down migrate-create db-setup seed \
       bundle generate generate-vapid-keys install-connectors \
       audit audit-backend audit-frontend audit-mobile \
       docker-build deploy

# Install all dependencies (frontend + backend + mobile)
install:
	cd frontend && npm install
	cd mobile && npm install
	go mod download

# Full setup: install deps + generate API client
setup: install generate

# Run both servers for development (Vite HMR + Go API)
# Installs custom connectors first if custom-connectors.json exists.
dev:
	@if [ -f custom-connectors.json ]; then $(MAKE) install-connectors; fi
	$(MAKE) dev-backend &
	$(MAKE) dev-frontend &
	wait

dev-backend:
	MODE=development go run .

dev-frontend:
	cd frontend && npm run dev

# ---------- Code Generation ----------

# Bundle the multi-file OpenAPI spec into a single file (version-pinned)
bundle:
	npx @redocly/cli@2.19.1 bundle spec/openapi/openapi.yaml -o spec/openapi/openapi.bundle.yaml

# Generate typed API clients from the bundled OpenAPI spec
generate: generate-frontend generate-mobile

generate-frontend: bundle
	cd frontend && npm run generate:api

generate-mobile: bundle
	cd mobile && npm run generate:api

# Type-check frontend (generates API client first, then runs tsc --noEmit)
typecheck: generate-frontend
	cd frontend && npx tsc --noEmit

# Build for production (generates API client first, then compiles)
# Embeds the git SHA as the Sentry release version via -ldflags.
GIT_SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
build: generate
	cd frontend && npm run build
	touch frontend/dist/.gitkeep
	go build -ldflags "-X main.version=$(GIT_SHA)" -o bin/server .

# Run the production binary
run:
	./bin/server

# ---------- Deployment ----------

# Build Docker image locally for testing before deploying.
# Requires VITE_SUPABASE_URL and VITE_SUPABASE_PUBLISHABLE_KEY in the environment
# (Vite inlines these into the JS bundle at build time).
# Usage: VITE_SUPABASE_URL=https://xxx.supabase.co VITE_SUPABASE_PUBLISHABLE_KEY=yyy make docker-build
docker-build:
	docker build \
		--build-arg VITE_SUPABASE_URL=$${VITE_SUPABASE_URL} \
		--build-arg VITE_SUPABASE_PUBLISHABLE_KEY=$${VITE_SUPABASE_PUBLISHABLE_KEY} \
		-t permission-slip-web .

# Deploy to Fly.io. Reads Supabase build args from the environment.
# Alternatively, configure [build.args] in fly.toml and just run: fly deploy
# Usage: VITE_SUPABASE_URL=https://xxx.supabase.co VITE_SUPABASE_PUBLISHABLE_KEY=yyy make deploy
deploy:
	fly deploy \
		--build-arg VITE_SUPABASE_URL=$${VITE_SUPABASE_URL} \
		--build-arg VITE_SUPABASE_PUBLISHABLE_KEY=$${VITE_SUPABASE_PUBLISHABLE_KEY}

# ---------- Testing ----------

test: test-backend test-frontend mobile-test

test-backend:
	@echo "=== Step 1: go build ==="
	go build ./... > /tmp/gobuild.log 2>&1; if [ $$? -ne 0 ]; then echo "::error::BUILD FAILED:"; cat /tmp/gobuild.log; cat /tmp/gobuild.log | head -50 | while IFS= read -r line; do echo "::error::$$line"; done; exit 2; fi
	@echo "=== Step 2: go vet ==="
	go vet ./... > /tmp/govet.log 2>&1; if [ $$? -ne 0 ]; then echo "::error::VET FAILED:"; cat /tmp/govet.log; cat /tmp/govet.log | head -50 | while IFS= read -r line; do echo "::error::$$line"; done; exit 2; fi
	@echo "=== Step 3: go test ==="
	go test -vet=off ./...
	@if curl -sf http://127.0.0.1:54321/auth/v1/health > /dev/null 2>&1; then \
		echo "Supabase detected — also running integration tests..."; \
		DATABASE_URL=postgresql://postgres:postgres@127.0.0.1:54322/postgres \
		go test -tags integration -v ./...; \
	else \
		echo "Supabase not detected — skipping integration tests (run 'supabase start' to include them)."; \
	fi

test-frontend:
	cd frontend && npm test

# Explicit integration test target — errors if Supabase is not running.
# (test-backend also runs these automatically when Supabase is detected.)
test-integration:
	@echo "Checking Supabase is running..."
	@curl -sf http://127.0.0.1:54321/auth/v1/health > /dev/null || \
		(echo "Error: Supabase is not running. Run 'supabase start' first." && exit 1)
	DATABASE_URL=postgresql://postgres:postgres@127.0.0.1:54322/postgres \
	go test -tags integration -v ./...

# ---------- Mobile (Expo) ----------

# Install mobile app dependencies
mobile-install:
	cd mobile && npm install

# Start Expo development server
mobile-start:
	cd mobile && npm start

# Run mobile tests (--ci for deterministic output in CI)
mobile-test:
	cd mobile && npm test -- --ci

# Type-check mobile app (no emit, just validate)
mobile-typecheck:
	cd mobile && npx tsc --noEmit

# EAS Build: development (simulator/internal distribution)
mobile-build-dev:
	cd mobile && npx eas-cli build --profile development --platform all
mobile-build-dev-ios:
	cd mobile && npx eas-cli build --profile development --platform ios
mobile-build-dev-android:
	cd mobile && npx eas-cli build --profile development --platform android

# EAS Build: preview (internal distribution for testers)
mobile-build-preview:
	cd mobile && npx eas-cli build --profile preview --platform all
mobile-build-preview-ios:
	cd mobile && npx eas-cli build --profile preview --platform ios
mobile-build-preview-android:
	cd mobile && npx eas-cli build --profile preview --platform android

# EAS Build: production (App Store / Google Play)
mobile-build-prod:
	cd mobile && npx eas-cli build --profile production --platform all

# EAS Submit: upload latest production build to app stores
mobile-submit:
	cd mobile && npx eas-cli submit --profile production --platform all

# EAS Update: publish OTA update to the production channel
mobile-update:
	cd mobile && npx eas-cli update --channel production

# ---------- Dependency Audit ----------

# Run all dependency audits
audit: audit-backend audit-frontend audit-mobile

# Audit Go modules for known vulnerabilities (requires govulncheck)
audit-backend:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# Audit npm packages for known vulnerabilities
audit-frontend:
	cd frontend && npm audit

# Audit mobile npm packages for known vulnerabilities
audit-mobile:
	cd mobile && npm audit

# ---------- Database ----------

# Create test database (requires standalone local Postgres for CI/tests)
db-setup:
	createdb permission_slip_test 2>/dev/null || true

# Run migrations against DATABASE_URL (defaults to Supabase local Postgres)
migrate-up:
	DATABASE_URL=$${DATABASE_URL:-postgresql://postgres:postgres@127.0.0.1:54322/postgres} \
	go run ./cmd/migrate up

migrate-down:
	DATABASE_URL=$${DATABASE_URL:-postgresql://postgres:postgres@127.0.0.1:54322/postgres} \
	go run ./cmd/migrate down

# Seed development database with test data (always cleans and re-seeds)
seed:
	DATABASE_URL=$${DATABASE_URL:-postgresql://postgres:postgres@127.0.0.1:54322/postgres} \
	go run ./cmd/seed

# ---------- Custom Connectors ----------

# Install custom connectors from custom-connectors.json
install-connectors:
	go run ./cmd/install-connectors

# Generate a VAPID key pair for Web Push (required to enable Web Push in production)
generate-vapid-keys:
	go run ./cmd/generate-vapid-keys

# Create a new migration file: make migrate-create NAME=add_users_table
migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=my_migration"; exit 1; fi
	@TIMESTAMP=$$(date +%Y%m%d%H%M%S); \
	FILE="db/migrations/$${TIMESTAMP}_$(NAME).sql"; \
	printf -- '-- +goose Up\n\n-- +goose Down\n' > $$FILE; \
	echo "Created $$FILE"

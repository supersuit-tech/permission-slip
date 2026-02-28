#!/usr/bin/env bash
# .openclaw/verify-setup.sh
# Run by the verify-pr skill before tests. Ensures all dependencies are
# installed and the test database is ready.
#
# Exit 0 = ready to test
# Exit 1 = something is missing (error message will explain)

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "=== Permission Slip verify-setup ==="

# ── 1. Frontend deps ─────────────────────────────────────────────────────────
echo ""
echo "▶ Checking frontend dependencies..."

FRONTEND_DIR="$REPO_ROOT/frontend"
if [ ! -d "$FRONTEND_DIR/node_modules" ]; then
  echo "  node_modules missing — running npm install..."
  cd "$FRONTEND_DIR" && npm install --silent
  cd "$REPO_ROOT"
  echo "  ✅ Frontend dependencies installed"
else
  # Check if package.json is newer than node_modules (deps may have changed)
  if [ "$FRONTEND_DIR/package.json" -nt "$FRONTEND_DIR/node_modules/.package-lock.json" ] 2>/dev/null; then
    echo "  package.json changed — running npm install..."
    cd "$FRONTEND_DIR" && npm install --silent
    cd "$REPO_ROOT"
    echo "  ✅ Frontend dependencies updated"
  else
    echo "  ✅ Frontend dependencies up to date"
  fi
fi

# ── 2. Go deps ───────────────────────────────────────────────────────────────
echo ""
echo "▶ Checking Go dependencies..."
if ! go mod download 2>&1; then
  echo "  ❌ go mod download failed"
  exit 1
fi
echo "  ✅ Go dependencies ready"

# ── 3. .env setup ────────────────────────────────────────────────────────────
echo ""
echo "▶ Checking .env..."
if [ ! -f "$REPO_ROOT/.env" ]; then
  if [ -f "$REPO_ROOT/.env.example" ]; then
    cp "$REPO_ROOT/.env.example" "$REPO_ROOT/.env"
    echo "  ✅ .env created from .env.example"
  else
    echo "  ⚠️  No .env or .env.example found — continuing without it"
  fi
else
  echo "  ✅ .env exists"
fi

# Load .env so DATABASE_URL_TEST is available if set there
if [ -f "$REPO_ROOT/.env" ]; then
  set -o allexport
  # shellcheck disable=SC1091
  source "$REPO_ROOT/.env"
  set +o allexport
fi

# ── 4. PostgreSQL check ───────────────────────────────────────────────────────
echo ""
echo "▶ Checking PostgreSQL..."

# find_pg_binary <name>
# Search Homebrew pg@16 and system install paths for a PostgreSQL binary.
# Prints the first match found; prints nothing if not found.
find_pg_binary() {
  local name="$1"
  for candidate in \
    /opt/homebrew/Cellar/postgresql@16/*/bin/"$name" \
    /usr/local/Cellar/postgresql@16/*/bin/"$name" \
    /opt/homebrew/bin/"$name" \
    /usr/local/bin/"$name" \
    "$name"; do
    if command -v "$candidate" &>/dev/null || [ -x "$candidate" ]; then
      echo "$candidate"
      return
    fi
  done
}

PSQL="$(find_pg_binary psql)"
CREATEDB="$(find_pg_binary createdb)"

if [ -z "$PSQL" ]; then
  echo "  ❌ psql not found."
  echo "     Install PostgreSQL 16: brew install postgresql@16"
  echo "     Then start it:         brew services start postgresql@16"
  exit 1
fi

if [ -z "$CREATEDB" ]; then
  echo "  ❌ createdb not found."
  echo "     Install PostgreSQL 16: brew install postgresql@16"
  exit 1
fi

# Check if Postgres is accepting connections
if ! "$PSQL" -U "$(whoami)" -d postgres -c '\q' &>/dev/null; then
  echo "  ❌ PostgreSQL is not running or not accepting connections."
  echo "     Start it with: brew services start postgresql@16"
  exit 1
fi
echo "  ✅ PostgreSQL is running"

# ── 5. Test database ──────────────────────────────────────────────────────────
echo ""
echo "▶ Checking test database..."

# Parse DB name from DATABASE_URL_TEST (fallback to default)
DB_URL="${DATABASE_URL_TEST:-postgres://localhost:5432/permission_slip_test?sslmode=disable}"
DB_NAME=$(echo "$DB_URL" | sed 's|.*://[^/]*/||' | sed 's/?.*$//')

if ! "$PSQL" -U "$(whoami)" -lqt 2>/dev/null | cut -d\| -f1 | grep -qw "$DB_NAME"; then
  echo "  Test database '$DB_NAME' not found — creating..."
  "$CREATEDB" -U "$(whoami)" "$DB_NAME"
  echo "  ✅ Test database '$DB_NAME' created"
else
  echo "  ✅ Test database '$DB_NAME' exists"
fi

# ── Done ──────────────────────────────────────────────────────────────────────
echo ""
echo "✅ Setup complete — ready to run tests"

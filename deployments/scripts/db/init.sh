#!/usr/bin/env bash
# ============================================================================
# Argus Platform — Database Initialization Script
# Runs all SQL migration files in order against the target PostgreSQL database.
#
# Usage:
#   ./init.sh                          # uses ARGUS_DB_DSN or default
#   ARGUS_DB_DSN="postgres://..." ./init.sh
#   ./init.sh --seed                   # include seed data (003_seed.sql)
#   ./init.sh --dry-run                # print what would be executed
# ============================================================================

set -euo pipefail

# --------------------------------------------------------------------------
# Configuration
# --------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DB_DSN="${ARGUS_DB_DSN:-postgres://argus:argus@localhost:5432/argus}"

# Parse flags
INCLUDE_SEED=false
DRY_RUN=false

for arg in "$@"; do
    case "$arg" in
        --seed)
            INCLUDE_SEED=true
            ;;
        --dry-run)
            DRY_RUN=true
            ;;
        --help|-h)
            echo "Usage: $0 [--seed] [--dry-run] [--help]"
            echo ""
            echo "Flags:"
            echo "  --seed      Include seed data (003_seed.sql) — only for dev environments"
            echo "  --dry-run   Print the SQL files that would be executed without running them"
            echo "  --help      Show this help message"
            echo ""
            echo "Environment:"
            echo "  ARGUS_DB_DSN   PostgreSQL connection string"
            echo "                 Default: postgres://argus:argus@localhost:5432/argus"
            exit 0
            ;;
        *)
            echo "Error: unknown argument '$arg'. Use --help for usage."
            exit 1
            ;;
    esac
done

# --------------------------------------------------------------------------
# Helpers
# --------------------------------------------------------------------------
log() {
    echo "[argus-db-init] $(date '+%Y-%m-%d %H:%M:%S') $*"
}

run_sql_file() {
    local file="$1"
    local filename
    filename="$(basename "$file")"

    if [ "$DRY_RUN" = true ]; then
        log "[DRY-RUN] Would execute: $filename"
        return 0
    fi

    log "Executing: $filename ..."
    if psql "$DB_DSN" -v ON_ERROR_STOP=1 -f "$file"; then
        log "  -> $filename applied successfully."
    else
        log "  -> ERROR: $filename failed. Aborting."
        exit 1
    fi
}

# --------------------------------------------------------------------------
# Pre-flight checks
# --------------------------------------------------------------------------
if ! command -v psql &>/dev/null; then
    log "Error: 'psql' command not found. Please install PostgreSQL client tools."
    exit 1
fi

# Verify connectivity (unless dry-run)
if [ "$DRY_RUN" = false ]; then
    log "Verifying database connectivity..."
    if ! psql "$DB_DSN" -c "SELECT 1;" &>/dev/null; then
        log "Error: Cannot connect to database at: $DB_DSN"
        log "Ensure PostgreSQL is running and the connection string is correct."
        exit 1
    fi
    log "Database connection verified."
fi

# --------------------------------------------------------------------------
# Run migrations
# --------------------------------------------------------------------------
log "============================================"
log "  Argus Database Initialization"
log "  DSN: ${DB_DSN%%:*}:****@${DB_DSN#*@}"
log "  Seed data: $INCLUDE_SEED"
log "  Dry run: $DRY_RUN"
log "============================================"

# Schema
run_sql_file "$SCRIPT_DIR/001_schema.sql"

# Row-Level Security
run_sql_file "$SCRIPT_DIR/002_rls.sql"

# Seed data (only if requested)
if [ "$INCLUDE_SEED" = true ]; then
    run_sql_file "$SCRIPT_DIR/003_seed.sql"
else
    log "Skipping seed data (use --seed flag to include)."
fi

log "============================================"
log "  Database initialization complete."
log "============================================"

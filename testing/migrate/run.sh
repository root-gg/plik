#!/bin/bash

set -e
cd "$(dirname "$0")"

BACKEND="migrate"
CMD=$1
TEST=$2

source ../utils.sh

ROOT=$(realpath ../..)

# Build plikd if not already built
if [[ ! -x "$ROOT/server/plikd" ]]; then
    echo "Building plikd..."
    ( cd "$ROOT" && make server )
fi

PLIKD="$ROOT/server/plikd"
WORKDIR=$(mktemp -d)

trap "rm -rf $WORKDIR" EXIT

SRC_DB="$WORKDIR/src.db"
DST_DB="$WORKDIR/dst.db"
SRC_CFG="$WORKDIR/src.cfg"
DST_CFG="$WORKDIR/dst.cfg"

function count_records {
    local db="$1"
    local table="$2"
    sqlite3 "$db" "SELECT COUNT(*) FROM $table;" 2>/dev/null || echo "0"
}

function start {
    echo "nothing to start for migrate tests"
}

function stop {
    echo "nothing to stop for migrate tests"
}

function status {
    true
}

function run_tests {
    BACKEND="$1"
    TEST="$2"

    echo ""
    echo "=== Migration E2E Test ==="
    echo ""

    # --- Step 1: Generate a small fake database ---
    echo " - Generating fake source database..."
    "$PLIKD" fakedb \
        --users 5 \
        --tokens 2 \
        --uploads 3 \
        --files 2 \
        --anon-uploads 3 \
        --output "$SRC_DB"

    echo ""

    # Count source records
    SRC_USERS=$(count_records "$SRC_DB" "users")
    SRC_TOKENS=$(count_records "$SRC_DB" "tokens")
    SRC_UPLOADS=$(count_records "$SRC_DB" "uploads")
    SRC_FILES=$(count_records "$SRC_DB" "files")
    SRC_SETTINGS=$(count_records "$SRC_DB" "settings")

    echo " - Source database contains:"
    echo "     users:   $SRC_USERS"
    echo "     tokens:  $SRC_TOKENS"
    echo "     uploads: $SRC_UPLOADS"
    echo "     files:   $SRC_FILES"
    echo "     settings:$SRC_SETTINGS"
    echo ""

    # Sanity — source must not be empty
    if [[ "$SRC_USERS" -eq 0 ]] || [[ "$SRC_UPLOADS" -eq 0 ]] || [[ "$SRC_FILES" -eq 0 ]]; then
        echo "FAIL: source database is empty after fakedb"
        exit 1
    fi

    # --- Step 2: Create config files ---
    cat > "$SRC_CFG" <<EOF
Debug = true
ListenPort = 8080
ListenAddress = "0.0.0.0"

DataBackend = "file"
[DataBackendConfig]
    Directory = "$WORKDIR/src_files"

[MetadataBackendConfig]
    Driver = "sqlite3"
    ConnectionString = "$SRC_DB"
EOF

    cat > "$DST_CFG" <<EOF
[MetadataBackendConfig]
    Driver = "sqlite3"
    ConnectionString = "$DST_DB"
EOF

    # --- Step 3: Run metadata-only migration ---
    echo " - Running: plikd migrate --metadata-only"
    "$PLIKD" --config "$SRC_CFG" migrate --to "$DST_CFG" --metadata-only
    echo ""

    # --- Step 4: Verify destination record counts match source ---
    DST_USERS=$(count_records "$DST_DB" "users")
    DST_TOKENS=$(count_records "$DST_DB" "tokens")
    DST_UPLOADS=$(count_records "$DST_DB" "uploads")
    DST_FILES=$(count_records "$DST_DB" "files")
    DST_SETTINGS=$(count_records "$DST_DB" "settings")

    echo " - Destination database contains:"
    echo "     users:   $DST_USERS"
    echo "     tokens:  $DST_TOKENS"
    echo "     uploads: $DST_UPLOADS"
    echo "     files:   $DST_FILES"
    echo "     settings:$DST_SETTINGS"
    echo ""

    FAIL=0
    if [[ "$SRC_USERS" -ne "$DST_USERS" ]]; then
        echo "FAIL: users count mismatch: src=$SRC_USERS dst=$DST_USERS"
        FAIL=1
    fi
    if [[ "$SRC_TOKENS" -ne "$DST_TOKENS" ]]; then
        echo "FAIL: tokens count mismatch: src=$SRC_TOKENS dst=$DST_TOKENS"
        FAIL=1
    fi
    if [[ "$SRC_UPLOADS" -ne "$DST_UPLOADS" ]]; then
        echo "FAIL: uploads count mismatch: src=$SRC_UPLOADS dst=$DST_UPLOADS"
        FAIL=1
    fi
    if [[ "$SRC_FILES" -ne "$DST_FILES" ]]; then
        echo "FAIL: files count mismatch: src=$SRC_FILES dst=$DST_FILES"
        FAIL=1
    fi

    if [[ "$FAIL" -eq 1 ]]; then
        echo ""
        echo "=== MIGRATION TEST FAILED ==="
        exit 1
    fi

    echo " ✓ All record counts match"
    echo ""

    # --- Step 5: Dry-run test ---
    echo " - Running: plikd migrate --dry-run (full)"

    # Reset dst for a clean dry-run
    rm -f "$DST_DB"

    DRY_OUTPUT=$("$PLIKD" --config "$SRC_CFG" migrate --to "$DST_CFG" --dry-run 2>&1)
    echo "$DRY_OUTPUT"
    echo ""

    # Destination should be empty after dry-run (only schema, no data)
    DST_UPLOADS_DRY=$(count_records "$DST_DB" "uploads")
    if [[ "$DST_UPLOADS_DRY" -ne 0 ]]; then
        echo "FAIL: dry-run wrote data to destination (uploads=$DST_UPLOADS_DRY)"
        exit 1
    fi
    echo " ✓ Dry-run did not write any data"
    echo ""

    # --- Step 6: Re-run migration with --ignore-errors (idempotency) ---
    echo " - Running: plikd migrate --metadata-only --ignore-errors (re-run on existing destination)"

    # Migrate again into the non-empty dst from step 3
    rm -f "$DST_DB"
    "$PLIKD" --config "$SRC_CFG" migrate --to "$DST_CFG" --metadata-only
    RERUN_OUTPUT=$("$PLIKD" --config "$SRC_CFG" migrate --to "$DST_CFG" --metadata-only --ignore-errors 2>&1)
    echo "$RERUN_OUTPUT"
    echo ""

    # After re-run with --ignore-errors, counts should still match
    DST_USERS_2=$(count_records "$DST_DB" "users")
    DST_UPLOADS_2=$(count_records "$DST_DB" "uploads")
    if [[ "$SRC_USERS" -ne "$DST_USERS_2" ]] || [[ "$SRC_UPLOADS" -ne "$DST_UPLOADS_2" ]]; then
        echo "FAIL: re-run counts mismatch after --ignore-errors"
        exit 1
    fi
    echo " ✓ Re-run with --ignore-errors preserved existing data"
    echo ""

    echo "=== ALL MIGRATION TESTS PASSED ==="
}

run_cmd

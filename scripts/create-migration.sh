#!/bin/bash
# Script to create new migration files
# Usage: ./scripts/create-migration.sh <migration_name>

set -e

if [ -z "$1" ]; then
    echo "Usage: ./scripts/create-migration.sh <migration_name>"
    echo "Example: ./scripts/create-migration.sh create_users_table"
    exit 1
fi

MIGRATION_NAME=$1
MIGRATIONS_DIR="db/migrations"

# Get the next migration number
LAST_MIGRATION=$(ls -1 "$MIGRATIONS_DIR" | grep -E '^[0-9]+' | sort -r | head -1 | cut -d_ -f1)
if [ -z "$LAST_MIGRATION" ]; then
    NEXT_NUM="000001"
else
    NEXT_NUM=$(printf "%06d" $((10#$LAST_MIGRATION + 1)))
fi

UP_FILE="${MIGRATIONS_DIR}/${NEXT_NUM}_${MIGRATION_NAME}.up.sql"
DOWN_FILE="${MIGRATIONS_DIR}/${NEXT_NUM}_${MIGRATION_NAME}.down.sql"

# Create the up migration file
cat > "$UP_FILE" <<EOF
-- Migration: ${MIGRATION_NAME}
-- Add your up migration here

EOF

# Create the down migration file
cat > "$DOWN_FILE" <<EOF
-- Migration: ${MIGRATION_NAME}
-- Add your down migration here

EOF

echo "✅ Created migration files:"
echo "   - $UP_FILE"
echo "   - $DOWN_FILE"
echo ""
echo "Next steps:"
echo "   1. Edit the migration files with your SQL"
echo "   2. Run migrations: make migrate-up"
echo "   3. Test rollback: make migrate-down"

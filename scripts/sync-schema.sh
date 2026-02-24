#!/bin/bash
# sync-schema.sh
# Dumps schema from Supabase production database
# Usage: ./scripts/sync-schema.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
OUTPUT_FILE="$PROJECT_ROOT/internal/db/queries/schema.sql"
ENV_FILE="$PROJECT_ROOT/.env"

# Load environment variables
if [ -f "$ENV_FILE" ]; then
    export $(grep -v '^#' "$ENV_FILE" | xargs)
fi

# Check DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "Error: DATABASE_URL not found in $ENV_FILE"
    exit 1
fi

echo "Dumping schema from Supabase production database..."

# Use Docker to run pg_dump (postgres 17 for compatibility with Supabase PG17)
# Note: If IPv6 fails, try using the pooler connection string instead:
# DATABASE_URL="postgresql://postgres.<project-ref>:<password>@aws-0-<region>.pooler.supabase.com:6543/postgres"

docker run --rm \
    postgres:17-alpine \
    pg_dump "$DATABASE_URL" \
    --schema=public \
    --no-owner \
    --no-privileges \
    --schema-only \
    --table=recipes \
    --table=recipe_ingredients \
    --table=recipe_instructions \
    --table=recipe_nutrition \
    --table=recipe_import_jobs \
    --table=social_media_owners \
    --table=profiles \
    --table=stored_images \
    --table=recipe_images \
    > "$OUTPUT_FILE"

echo "Schema dumped to $OUTPUT_FILE"
echo ""
echo "Next steps:"
echo "  1. Review the generated schema.sql"
echo "  2. Run: make sqlc-generate"
echo "  3. Fix any Go code that uses old field names"
echo "  4. Run: go build ./..."

#!/bin/bash

# Generate JWT token for Bruno testing
# Usage: ./generate-jwt.sh [dev|production]
#   dev       - uses DEV_SUPABASE_JWT_SECRET, DEV_SUPABASE_URL, DEV_USER_ID
#   production - uses SUPABASE_JWT_SECRET, SUPABASE_URL (user id from SUPABASE_USER_ID or default)

# Load environment variables
if [ -f .env.bruno ]; then
    export $(grep -v '^#' .env.bruno | xargs)
fi

ENV="${1:-dev}"

if [ "$ENV" = "dev" ]; then
    JWT_SECRET="$DEV_SUPABASE_JWT_SECRET"
    SUPABASE_BASE_URL="$DEV_SUPABASE_URL"
    USER_ID="$DEV_USER_ID"
    if [ -z "$JWT_SECRET" ] || [ -z "$SUPABASE_BASE_URL" ] || [ -z "$USER_ID" ]; then
        echo "Error: DEV_SUPABASE_JWT_SECRET, DEV_SUPABASE_URL and DEV_USER_ID must be set in .env.bruno for dev"
        exit 1
    fi
elif [ "$ENV" = "production" ]; then
    JWT_SECRET="$SUPABASE_JWT_SECRET"
    SUPABASE_BASE_URL="$SUPABASE_URL"
    USER_ID="${SUPABASE_USER_ID:-60399ada-5092-4665-b002-e0fc0345cb1b}"
    if [ -z "$JWT_SECRET" ] || [ -z "$SUPABASE_BASE_URL" ]; then
        echo "Error: SUPABASE_JWT_SECRET and SUPABASE_URL must be set in .env.bruno for production"
        exit 1
    fi
else
    echo "Error: Invalid env. Use 'dev' or 'production'"
    echo "Usage: ./generate-jwt.sh [dev|production]"
    exit 1
fi

# Create JWT header
HEADER=$(echo -n '{"alg":"HS256","typ":"JWT"}' | base64 | tr '+/' '-_' | tr -d '=')

# Create JWT payload (expires in 1 hour)
EXP=$(($(date +%s) + 3600))
PAYLOAD=$(echo -n "{\"sub\":\"$USER_ID\",\"iss\":\"$SUPABASE_BASE_URL/auth/v1\",\"exp\":$EXP}" | base64 | tr '+/' '-_' | tr -d '=')

# Create signature
SIGNATURE=$(echo -n "$HEADER.$PAYLOAD" | openssl dgst -sha256 -hmac "$JWT_SECRET" -binary | base64 | tr '+/' '-_' | tr -d '=')

# Combine to form JWT
JWT="$HEADER.$PAYLOAD.$SIGNATURE"

echo "Generated JWT Token ($ENV):"
echo "$JWT"
echo ""
echo "Add this to your bruno/environments/local.bru:"
echo ""
echo "vars {"
echo "  baseUrl: http://localhost:8080"
echo "  testUserId: $USER_ID"
echo "  jwtToken: $JWT"
echo "}"

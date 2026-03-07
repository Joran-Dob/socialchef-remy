#!/bin/bash

# Generate JWT token for Bruno testing
# Usage: ./generate-jwt.sh

# Load environment variables
if [ -f .env.bruno ]; then
    export $(grep -v '^#' .env.bruno | xargs)
fi

if [ -z "$SUPABASE_JWT_SECRET" ] || [ -z "$SUPABASE_URL" ]; then
    echo "Error: SUPABASE_JWT_SECRET and SUPABASE_URL must be set in .env.bruno"
    exit 1
fi

# Create JWT header
HEADER=$(echo -n '{"alg":"HS256","typ":"JWT"}' | base64 | tr '+/' '-_' | tr -d '=')

# Create JWT payload (expires in 1 hour)
EXP=$(($(date +%s) + 3600))
PAYLOAD=$(echo -n "{\"sub\":\"60399ada-5092-4665-b002-e0fc0345cb1b\",\"iss\":\"$SUPABASE_URL/auth/v1\",\"exp\":$EXP}" | base64 | tr '+/' '-_' | tr -d '=')

# Create signature
SIGNATURE=$(echo -n "$HEADER.$PAYLOAD" | openssl dgst -sha256 -hmac "$SUPABASE_JWT_SECRET" -binary | base64 | tr '+/' '-_' | tr -d '=')

# Combine to form JWT
JWT="$HEADER.$PAYLOAD.$SIGNATURE"

echo "Generated JWT Token:"
echo "$JWT"
echo ""
echo "Add this to your bruno/environments/local.bru:"
echo ""
echo "vars {"
echo "  baseUrl: http://localhost:8080"
echo "  testUserId: 60399ada-5092-4665-b002-e0fc0345cb1b"
echo "  jwtToken: $JWT"
echo "}"

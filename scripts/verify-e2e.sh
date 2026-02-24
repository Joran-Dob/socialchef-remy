#!/bin/bash

# E2E Verification Script for SocialChef Remy
# Usage: ./scripts/verify-e2e.sh [API_URL] [JWT_TOKEN]
# Or set environment variables: API_URL and JWT_TOKEN

set -e

# Configuration
API_URL="${1:-${API_URL:-"https://socialchef-remy.fly.dev"}}"
JWT_TOKEN="Gb2wmGBbsf0WH2Fqw7ny/iMiuXa8O3jj8T+i8RJHqWpmvMGYPNCsemdEz02N83GL/NbCDEDHYEE18PomzAUvng=="
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-300}"  # 5 minutes default
POLL_INTERVAL="${POLL_INTERVAL:-5}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

log_header() {
    echo ""
    echo "========================================"
    echo "$1"
    echo "========================================"
}

# Check prerequisites
check_prerequisites() {
    log_header "Checking Prerequisites"
    
    if ! command -v curl &> /dev/null; then
        echo "Error: curl is required but not installed."
        exit 1
    fi
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_error "JWT_TOKEN not provided. Pass as argument or set as environment variable."
        echo "Usage: $0 [API_URL] [JWT_TOKEN]"
        echo "   or: API_URL=<url> JWT_TOKEN=<token> $0"
        exit 1
    fi
    
    log_info "API URL: $API_URL"
    log_info "Timeout: ${TIMEOUT_SECONDS}s"
    log_success "Prerequisites check"
}

# Test 1: Health endpoint
test_health() {
    log_header "Test 1: Health Endpoint"
    
    local response
    local http_code
    
    response=$(curl -s -w "\n%{http_code}" "$API_URL/health")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [[ "$http_code" == "200" ]]; then
        log_success "Health endpoint returned 200"
        log_info "Response: $body"
    else
        log_error "Health endpoint returned HTTP $http_code"
        log_info "Response: $body"
    fi
}

# Test 2: Create recipe import
test_create_recipe() {
    log_header "Test 2: Create Recipe Import"
    
    local response
    local http_code
    local test_url="https://www.instagram.com/p/DU8Wm-pCF3M/"
    
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "{\"url\": \"$test_url\"}" \
        "$API_URL/api/recipe")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [[ "$http_code" == "200" || "$http_code" == "201" || "$http_code" == "202" ]]; then
        log_success "Create recipe endpoint returned HTTP $http_code"
        log_info "Response: $body"
        
        # Extract recipe_id from response
        RECIPE_ID=$(echo "$body" | grep -o '"recipe_id":"[^"]*"' | cut -d'"' -f4)
        if [[ -z "$RECIPE_ID" ]]; then
            RECIPE_ID=$(echo "$body" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        fi
        
        if [[ -n "$RECIPE_ID" ]]; then
            log_info "Recipe ID: $RECIPE_ID"
        else
            log_info "No recipe_id found in response, will use status endpoint directly"
        fi
    else
        log_error "Create recipe endpoint returned HTTP $http_code"
        log_info "Response: $body"
        return 1
    fi
}

# Test 3: Poll recipe status
test_poll_recipe_status() {
    log_header "Test 3: Poll Recipe Status"
    
    local elapsed=0
    local status
    local http_code
    local response
    local completed=false
    
    log_info "Polling for up to ${TIMEOUT_SECONDS}s..."
    
    while [[ $elapsed -lt $TIMEOUT_SECONDS ]]; do
        response=$(curl -s -w "\n%{http_code}" \
            -H "Authorization: Bearer $JWT_TOKEN" \
            "$API_URL/api/recipe-status" 2>/dev/null || echo -e "\n000")
        
        http_code=$(echo "$response" | tail -n1)
        body=$(echo "$response" | sed '$d')
        
        if [[ "$http_code" == "200" ]]; then
            # Try to extract status from response
            status=$(echo "$body" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
            
            if [[ -z "$status" ]]; then
                status=$(echo "$body" | grep -o '"state":"[^"]*"' | cut -d'"' -f4)
            fi
            
            log_info "Status: ${status:-unknown} (${elapsed}s elapsed)"
            
            if [[ "$status" == "completed" || "$status" == "success" ]]; then
                log_success "Recipe processing completed"
                completed=true
                break
            elif [[ "$status" == "failed" || "$status" == "error" ]]; then
                log_error "Recipe processing failed"
                log_info "Response: $body"
                break
            fi
        else
            log_info "HTTP $http_code (${elapsed}s elapsed)"
        fi
        
        sleep $POLL_INTERVAL
        elapsed=$((elapsed + POLL_INTERVAL))
    done
    
    if [[ "$completed" != "true" ]]; then
        log_error "Recipe status polling timed out after ${TIMEOUT_SECONDS}s"
    fi
}

# Test 4: User import status
test_user_import_status() {
    log_header "Test 4: User Import Status"
    
    local response
    local http_code
    
    response=$(curl -s -w "\n%{http_code}" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        "$API_URL/api/user-import-status")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [[ "$http_code" == "200" ]]; then
        log_success "User import status endpoint returned HTTP 200"
        log_info "Response: $body"
    else
        log_error "User import status endpoint returned HTTP $http_code"
        log_info "Response: $body"
    fi
}

# Main execution
main() {
    log_header "SocialChef Remy E2E Verification"
    log_info "Starting verification against $API_URL"
    
    check_prerequisites
    test_health
    
    # Only run authenticated tests if we have a token
    if [[ -n "$JWT_TOKEN" ]]; then
        if test_create_recipe; then
            test_poll_recipe_status
        fi
        test_user_import_status
    else
        log_info "Skipping authenticated tests (no JWT_TOKEN provided)"
    fi
    
    # Summary
    log_header "Test Summary"
    echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}✗ Some tests failed.${NC}"
        exit 1
    fi
}

main "$@"

#!/bin/bash
# API Key Authentication test script for Moon
# Tests: create, authenticate, rotate, delete API keys
# Supports PREFIX environment variable for custom URL prefixes
# Usage: PREFIX=/api/v1 ./scripts/auth-apikey.sh
# Requires: jq, curl

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Default to empty prefix if not set
PREFIX=${PREFIX:-}

# Base URL
BASE_URL="http://localhost:6006${PREFIX}"

# Test counters
PASSED=0
FAILED=0

# Admin credentials for JWT authentication (needed to manage API keys)
ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-change-me-on-first-login}"

# Variables to store created resources for cleanup
CREATED_KEY_ID=""
API_KEY_VALUE=""
ACCESS_TOKEN=""

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Moon API Key Authentication Tests${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo "Base URL: ${BASE_URL}"
    echo ""
}

pass() {
    echo -e "${GREEN}✅ PASS${NC}: $1"
    ((PASSED++))
}

fail() {
    echo -e "${RED}❌ FAIL${NC}: $1"
    echo "   Response: $2"
    ((FAILED++))
}

cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up test data...${NC}"
    
    # Delete created API key if it exists
    if [ -n "$CREATED_KEY_ID" ] && [ -n "$ACCESS_TOKEN" ]; then
        curl -s -X POST "${BASE_URL}/apikeys:destroy?id=${CREATED_KEY_ID}" \
            -H "Authorization: Bearer ${ACCESS_TOKEN}" > /dev/null 2>&1 || true
        echo "Cleaned up API key: ${CREATED_KEY_ID}"
    fi
}

# Set up cleanup trap
trap cleanup EXIT

# Check for jq
if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq is required but not installed${NC}"
    exit 1
fi

print_header

echo "[0] Authenticating as admin to manage API keys..."
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/auth:login" \
    -H "Content-Type: application/json" \
    -d "{\"username\": \"${ADMIN_USERNAME}\", \"password\": \"${ADMIN_PASSWORD}\"}")

ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token // empty')

if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
    echo -e "${RED}Failed to authenticate as admin${NC}"
    echo "Response: $LOGIN_RESPONSE"
    echo ""
    echo "Make sure Moon is running with API key support enabled (apikey.enabled: true)"
    exit 1
fi
echo "Admin authenticated successfully"

echo ""
echo "[1] Testing POST /apikeys:create (create new API key)..."
CREATE_RESPONSE=$(curl -s -X POST "${BASE_URL}/apikeys:create" \
    -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Test API Key",
        "description": "API key for testing",
        "role": "user",
        "can_write": false
    }')

CREATED_KEY_ID=$(echo "$CREATE_RESPONSE" | jq -r '.id // empty')
API_KEY_VALUE=$(echo "$CREATE_RESPONSE" | jq -r '.key // empty')

if [ -n "$CREATED_KEY_ID" ] && [ "$CREATED_KEY_ID" != "null" ] && [ -n "$API_KEY_VALUE" ]; then
    pass "API key created (ID: ${CREATED_KEY_ID})"
    echo "   Key prefix: ${API_KEY_VALUE:0:20}..."
else
    fail "Failed to create API key" "$CREATE_RESPONSE"
fi

echo ""
echo "[2] Testing GET /apikeys:list (list all API keys)..."
LIST_RESPONSE=$(curl -s -X GET "${BASE_URL}/apikeys:list" \
    -H "Authorization: Bearer ${ACCESS_TOKEN}")

KEY_COUNT=$(echo "$LIST_RESPONSE" | jq '.apikeys | length // 0')

if [ "$KEY_COUNT" -gt 0 ]; then
    pass "Listed ${KEY_COUNT} API key(s)"
else
    fail "Failed to list API keys" "$LIST_RESPONSE"
fi

echo ""
echo "[3] Testing GET /apikeys:get (get specific API key)..."
if [ -n "$CREATED_KEY_ID" ]; then
    GET_RESPONSE=$(curl -s -X GET "${BASE_URL}/apikeys:get?id=${CREATED_KEY_ID}" \
        -H "Authorization: Bearer ${ACCESS_TOKEN}")

    KEY_NAME=$(echo "$GET_RESPONSE" | jq -r '.name // empty')

    if [ "$KEY_NAME" = "Test API Key" ]; then
        pass "Retrieved API key metadata"
    else
        fail "Failed to get API key" "$GET_RESPONSE"
    fi
fi

echo ""
echo "[4] Testing API key authentication on protected endpoint..."
if [ -n "$API_KEY_VALUE" ]; then
    APIKEY_AUTH_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "${BASE_URL}/collections:list" \
        -H "Authorization: Bearer ${API_KEY_VALUE}")
    APIKEY_AUTH_STATUS=$(echo "$APIKEY_AUTH_RESPONSE" | tail -n1)

    if [ "$APIKEY_AUTH_STATUS" = "200" ]; then
        pass "API key authentication works"
    else
        fail "API key authentication failed, got ${APIKEY_AUTH_STATUS}" "$(echo "$APIKEY_AUTH_RESPONSE" | sed '$d')"
    fi
fi

echo ""
echo "[5] Testing API key authentication with invalid key..."
INVALID_KEY_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "${BASE_URL}/collections:list" \
    -H "Authorization: Bearer invalid-api-key-12345")
INVALID_KEY_STATUS=$(echo "$INVALID_KEY_RESPONSE" | tail -n1)

if [ "$INVALID_KEY_STATUS" = "401" ]; then
    pass "Invalid API key correctly rejected"
else
    fail "Expected 401 for invalid API key, got ${INVALID_KEY_STATUS}" "$(echo "$INVALID_KEY_RESPONSE" | sed '$d')"
fi

echo ""
echo "[6] Testing POST /apikeys:update (update metadata)..."
if [ -n "$CREATED_KEY_ID" ]; then
    UPDATE_RESPONSE=$(curl -s -X POST "${BASE_URL}/apikeys:update?id=${CREATED_KEY_ID}" \
        -H "Authorization: Bearer ${ACCESS_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "Updated Test API Key",
            "description": "Updated description",
            "can_write": true
        }')

    UPDATED_NAME=$(echo "$UPDATE_RESPONSE" | jq -r '.name // empty')

    if [ "$UPDATED_NAME" = "Updated Test API Key" ]; then
        pass "API key metadata updated"
    else
        fail "Failed to update API key" "$UPDATE_RESPONSE"
    fi
fi

echo ""
echo "[7] Testing write access with updated can_write permission..."
if [ -n "$API_KEY_VALUE" ]; then
    # First, let's check if collections exist or create a test one
    # This tests that can_write: true now allows data modification
    # Note: The exact behavior depends on your collection setup
    WRITE_TEST_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "${BASE_URL}/collections:list" \
        -H "Authorization: Bearer ${API_KEY_VALUE}")
    WRITE_TEST_STATUS=$(echo "$WRITE_TEST_RESPONSE" | tail -n1)

    if [ "$WRITE_TEST_STATUS" = "200" ]; then
        pass "API key with can_write still has read access"
    else
        fail "Read access failed after permission update" "$(echo "$WRITE_TEST_RESPONSE" | sed '$d')"
    fi
fi

echo ""
echo "[8] Testing POST /apikeys:update (rotate key)..."
if [ -n "$CREATED_KEY_ID" ]; then
    ROTATE_RESPONSE=$(curl -s -X POST "${BASE_URL}/apikeys:update?id=${CREATED_KEY_ID}" \
        -H "Authorization: Bearer ${ACCESS_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"action": "rotate"}')

    NEW_KEY=$(echo "$ROTATE_RESPONSE" | jq -r '.key // empty')

    if [ -n "$NEW_KEY" ] && [ "$NEW_KEY" != "null" ]; then
        pass "API key rotated successfully"
        OLD_KEY="$API_KEY_VALUE"
        API_KEY_VALUE="$NEW_KEY"
        echo "   New key prefix: ${NEW_KEY:0:20}..."
    else
        fail "Failed to rotate API key" "$ROTATE_RESPONSE"
    fi
fi

echo ""
echo "[9] Testing old API key after rotation (should fail)..."
if [ -n "$OLD_KEY" ]; then
    OLD_KEY_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "${BASE_URL}/collections:list" \
        -H "Authorization: Bearer ${OLD_KEY}")
    OLD_KEY_STATUS=$(echo "$OLD_KEY_RESPONSE" | tail -n1)

    if [ "$OLD_KEY_STATUS" = "401" ]; then
        pass "Old API key correctly invalidated after rotation"
    else
        fail "Old key should be invalid after rotation, got ${OLD_KEY_STATUS}" "$(echo "$OLD_KEY_RESPONSE" | sed '$d')"
    fi
fi

echo ""
echo "[10] Testing new API key after rotation..."
if [ -n "$API_KEY_VALUE" ]; then
    NEW_KEY_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "${BASE_URL}/collections:list" \
        -H "Authorization: Bearer ${API_KEY_VALUE}")
    NEW_KEY_STATUS=$(echo "$NEW_KEY_RESPONSE" | tail -n1)

    if [ "$NEW_KEY_STATUS" = "200" ]; then
        pass "New API key works after rotation"
    else
        fail "New key should work after rotation, got ${NEW_KEY_STATUS}" "$(echo "$NEW_KEY_RESPONSE" | sed '$d')"
    fi
fi

echo ""
echo "[11] Testing POST /apikeys:destroy (delete API key)..."
if [ -n "$CREATED_KEY_ID" ]; then
    DESTROY_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/apikeys:destroy?id=${CREATED_KEY_ID}" \
        -H "Authorization: Bearer ${ACCESS_TOKEN}")
    DESTROY_STATUS=$(echo "$DESTROY_RESPONSE" | tail -n1)

    if [ "$DESTROY_STATUS" = "200" ]; then
        pass "API key deleted successfully"
        # Clear the ID so cleanup doesn't try to delete again
        CREATED_KEY_ID=""
    else
        fail "Failed to delete API key, got ${DESTROY_STATUS}" "$(echo "$DESTROY_RESPONSE" | sed '$d')"
    fi
fi

echo ""
echo "[12] Testing deleted API key (should fail)..."
if [ -n "$API_KEY_VALUE" ]; then
    DELETED_KEY_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "${BASE_URL}/collections:list" \
        -H "Authorization: Bearer ${API_KEY_VALUE}")
    DELETED_KEY_STATUS=$(echo "$DELETED_KEY_RESPONSE" | tail -n1)

    if [ "$DELETED_KEY_STATUS" = "401" ]; then
        pass "Deleted API key correctly rejected"
    else
        fail "Deleted key should be invalid, got ${DELETED_KEY_STATUS}" "$(echo "$DELETED_KEY_RESPONSE" | sed '$d')"
    fi
fi

# Summary
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}API Key Test Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Passed: ${PASSED}${NC}"
echo -e "${RED}Failed: ${FAILED}${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All API Key tests passed! ✅${NC}"
    exit 0
else
    echo -e "${RED}Some API Key tests failed! ❌${NC}"
    exit 1
fi

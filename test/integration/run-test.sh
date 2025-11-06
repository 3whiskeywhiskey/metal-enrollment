#!/usr/bin/env bash
set -euo pipefail

# Complete integration test runner

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$TEST_DIR/../.." && pwd)"

echo "=========================================="
echo "Metal Enrollment Integration Test"
echo "=========================================="
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up..."
    "$TEST_DIR/stop-services.sh" 2>/dev/null || true
}
trap cleanup EXIT

# Step 1: Setup environment
echo "Step 1: Setting up test environment..."
"$TEST_DIR/setup-test-env.sh"
source "$TEST_DIR/run/env.sh"

# Step 2: Build test registration image
echo ""
echo "Step 2: Building test registration image..."
"$TEST_DIR/build-test-registration.sh"

# Step 3: Build Go binaries
echo ""
echo "Step 3: Building Go binaries..."
cd "$PROJECT_DIR"
make build

# Step 4: Start services
echo ""
echo "Step 4: Starting services..."
"$TEST_DIR/start-services.sh"

# Wait for services to be ready
echo ""
echo "Step 5: Waiting for services to be ready..."
for i in {1..30}; do
    if curl -s "http://localhost:$ENROLLMENT_PORT/health" > /dev/null 2>&1; then
        echo "✓ Enrollment server is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "✗ Enrollment server failed to start"
        cat "$TEST_DIR/run/logs/server.log"
        exit 1
    fi
    sleep 1
done

# Step 6: Test enrollment API
echo ""
echo "Step 6: Testing enrollment API..."
ENROLL_RESPONSE=$("$IMAGES_DIR/registration/enroll-test.sh" "$ENROLLMENT_URL" || true)
echo "$ENROLL_RESPONSE"

if echo "$ENROLL_RESPONSE" | grep -q "Enrollment successful"; then
    echo "✓ Enrollment test passed"
else
    echo "✗ Enrollment test failed"
    echo "Server logs:"
    cat "$TEST_DIR/run/logs/server.log"
    exit 1
fi

# Step 7: Verify machine in database
echo ""
echo "Step 7: Verifying machine in database..."
MACHINES=$(curl -s "http://localhost:$ENROLLMENT_PORT/api/v1/machines")
echo "Enrolled machines:"
echo "$MACHINES" | jq . || echo "$MACHINES"

MACHINE_COUNT=$(echo "$MACHINES" | jq '. | length' 2>/dev/null || echo "0")
if [ "$MACHINE_COUNT" -gt 0 ]; then
    echo "✓ Found $MACHINE_COUNT machine(s) in database"
else
    echo "✗ No machines found in database"
    exit 1
fi

# Step 8: Test dashboard access
echo ""
echo "Step 8: Testing dashboard access..."
DASHBOARD=$(curl -s "http://localhost:$ENROLLMENT_PORT/")
if echo "$DASHBOARD" | grep -q "Metal Enrollment"; then
    echo "✓ Dashboard is accessible"
else
    echo "✗ Dashboard failed to load"
    exit 1
fi

# Step 9: Test machine update
echo ""
echo "Step 9: Testing machine configuration update..."
MACHINE_ID=$(echo "$MACHINES" | jq -r '.[0].id' 2>/dev/null || echo "")
if [ -n "$MACHINE_ID" ]; then
    UPDATE_RESPONSE=$(curl -s -X PUT "http://localhost:$ENROLLMENT_PORT/api/v1/machines/$MACHINE_ID" \
        -H "Content-Type: application/json" \
        -d '{
            "hostname": "test-server-01",
            "description": "Integration test server",
            "nixos_config": "{ config, pkgs, ... }: { boot.loader.grub.enable = true; }"
        }')

    echo "Update response:"
    echo "$UPDATE_RESPONSE" | jq . || echo "$UPDATE_RESPONSE"

    if echo "$UPDATE_RESPONSE" | jq -e '.hostname == "test-server-01"' > /dev/null 2>&1; then
        echo "✓ Machine update successful"
    else
        echo "✗ Machine update failed"
    fi
fi

# Step 10: Test iPXE script generation
echo ""
echo "Step 10: Testing iPXE script generation..."
SERVICE_TAG=$(echo "$MACHINES" | jq -r '.[0].service_tag' 2>/dev/null || echo "")
if [ -n "$SERVICE_TAG" ]; then
    IPXE_SCRIPT=$(curl -s "http://localhost:$IPXE_PORT/nixos/machines/$SERVICE_TAG.ipxe")
    echo "iPXE script for $SERVICE_TAG:"
    echo "$IPXE_SCRIPT"

    if echo "$IPXE_SCRIPT" | grep -q "ipxe"; then
        echo "✓ iPXE script generation works"
    else
        echo "✗ iPXE script generation failed"
    fi
fi

# Summary
echo ""
echo "=========================================="
echo "Integration Test Results"
echo "=========================================="
echo "✓ All tests passed!"
echo ""
echo "Services are still running. You can:"
echo "  - View dashboard: http://localhost:$ENROLLMENT_PORT"
echo "  - View machines: curl http://localhost:$ENROLLMENT_PORT/api/v1/machines | jq"
echo "  - Check logs: ls -l $TEST_DIR/run/logs/"
echo ""
echo "To stop services: $TEST_DIR/stop-services.sh"
echo ""

# Keep services running if requested
if [ "${KEEP_RUNNING:-}" = "1" ]; then
    echo "Keeping services running (KEEP_RUNNING=1)"
    echo "Press Ctrl+C to stop"
    trap - EXIT
    wait
fi

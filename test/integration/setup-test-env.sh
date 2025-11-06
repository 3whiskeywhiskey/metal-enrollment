#!/usr/bin/env bash
set -euo pipefail

# Integration test environment setup
# This script sets up a complete local PXE environment for testing

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$TEST_DIR/../.." && pwd)"
RUN_DIR="$TEST_DIR/run"

echo "=== Metal Enrollment Integration Test Setup ==="
echo "Project dir: $PROJECT_DIR"
echo "Test dir: $TEST_DIR"
echo "Run dir: $RUN_DIR"

# Create test directories
mkdir -p "$RUN_DIR"/{db,images,logs,pxe/{tftp,http},vm}

# Configuration
export DB_FILE="$RUN_DIR/db/test.db"
export IMAGES_DIR="$RUN_DIR/images"
export TFTP_DIR="$RUN_DIR/pxe/tftp"
export HTTP_DIR="$RUN_DIR/pxe/http"
export VM_DIR="$RUN_DIR/vm"

# Network configuration
export TEST_NETWORK="127.0.0.0/8"
export TEST_GATEWAY="127.0.0.1"
export TEST_DHCP_RANGE="127.0.0.100,127.0.0.200"
export SERVER_IP="127.0.0.1"

# Service ports
export ENROLLMENT_PORT=8080
export BUILDER_PORT=8081
export IPXE_PORT=8082

# URLs
export BASE_URL="http://$SERVER_IP:$IPXE_PORT"
export ENROLLMENT_URL="http://$SERVER_IP:$ENROLLMENT_PORT/api/v1/enroll"
export API_URL="http://$SERVER_IP:$ENROLLMENT_PORT/api/v1"

echo ""
echo "Environment configured:"
echo "  Network: $TEST_NETWORK"
echo "  Server IP: $SERVER_IP"
echo "  Enrollment API: $ENROLLMENT_URL"
echo "  iPXE Server: $BASE_URL"
echo ""

# Export for other scripts
cat > "$RUN_DIR/env.sh" <<EOF
export DB_FILE="$DB_FILE"
export IMAGES_DIR="$IMAGES_DIR"
export TFTP_DIR="$TFTP_DIR"
export HTTP_DIR="$HTTP_DIR"
export VM_DIR="$VM_DIR"
export TEST_NETWORK="$TEST_NETWORK"
export TEST_GATEWAY="$TEST_GATEWAY"
export TEST_DHCP_RANGE="$TEST_DHCP_RANGE"
export SERVER_IP="$SERVER_IP"
export ENROLLMENT_PORT=$ENROLLMENT_PORT
export BUILDER_PORT=$BUILDER_PORT
export IPXE_PORT=$IPXE_PORT
export BASE_URL="$BASE_URL"
export ENROLLMENT_URL="$ENROLLMENT_URL"
export API_URL="$API_URL"
EOF

echo "Environment setup complete!"
echo "Source: source $RUN_DIR/env.sh"

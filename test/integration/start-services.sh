#!/usr/bin/env bash
set -euo pipefail

# Start all services for integration testing

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$TEST_DIR/../.." && pwd)"
source "$TEST_DIR/run/env.sh"

echo "=== Starting Metal Enrollment Services ==="

# Build binaries if not already built
if [ ! -f "$PROJECT_DIR/bin/server" ]; then
    echo "Building binaries..."
    cd "$PROJECT_DIR" && make build
fi

# Start enrollment server
echo "Starting enrollment server on port $ENROLLMENT_PORT..."
DB_DRIVER=sqlite3 \
DB_DSN="$DB_FILE" \
LISTEN_ADDR=":$ENROLLMENT_PORT" \
BUILDER_URL="http://localhost:$BUILDER_PORT" \
"$PROJECT_DIR/bin/server" > "$TEST_DIR/run/logs/server.log" 2>&1 &
echo $! > "$TEST_DIR/run/server.pid"

sleep 2

# Start iPXE server
echo "Starting iPXE server on port $IPXE_PORT..."
BASE_URL="$BASE_URL" \
ENROLLMENT_URL="$ENROLLMENT_URL" \
API_URL="$API_URL" \
IMAGES_DIR="$IMAGES_DIR" \
LISTEN_ADDR=":$IPXE_PORT" \
"$PROJECT_DIR/bin/ipxe-server" > "$TEST_DIR/run/logs/ipxe.log" 2>&1 &
echo $! > "$TEST_DIR/run/ipxe.pid"

sleep 2

# Verify services are running
echo ""
echo "Checking services..."

for port in $ENROLLMENT_PORT $IPXE_PORT; do
    if nc -z localhost $port 2>/dev/null; then
        echo "✓ Service on port $port is running"
    else
        echo "✗ Service on port $port failed to start"
        cat "$TEST_DIR/run/logs"/*.log
        exit 1
    fi
done

echo ""
echo "All services started successfully!"
echo ""
echo "Service URLs:"
echo "  Dashboard: http://localhost:$ENROLLMENT_PORT"
echo "  API: http://localhost:$ENROLLMENT_PORT/api/v1"
echo "  iPXE: http://localhost:$IPXE_PORT"
echo ""
echo "Logs:"
echo "  Server: tail -f $TEST_DIR/run/logs/server.log"
echo "  iPXE: tail -f $TEST_DIR/run/logs/ipxe.log"
echo ""
echo "PIDs saved in $TEST_DIR/run/*.pid"

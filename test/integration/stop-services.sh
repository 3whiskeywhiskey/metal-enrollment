#!/usr/bin/env bash
set -euo pipefail

# Stop all services

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== Stopping Metal Enrollment Services ==="

for pidfile in "$TEST_DIR/run"/*.pid; do
    if [ -f "$pidfile" ]; then
        pid=$(cat "$pidfile")
        name=$(basename "$pidfile" .pid)
        if ps -p "$pid" > /dev/null 2>&1; then
            echo "Stopping $name (PID: $pid)..."
            kill "$pid" 2>/dev/null || true
            sleep 1
            # Force kill if still running
            if ps -p "$pid" > /dev/null 2>&1; then
                kill -9 "$pid" 2>/dev/null || true
            fi
        fi
        rm "$pidfile"
    fi
done

# Stop dnsmasq if running
if [ -f "$TEST_DIR/run/dnsmasq.pid" ]; then
    pid=$(cat "$TEST_DIR/run/dnsmasq.pid")
    if ps -p "$pid" > /dev/null 2>&1; then
        echo "Stopping dnsmasq (PID: $pid)..."
        kill "$pid" 2>/dev/null || true
    fi
    rm "$TEST_DIR/run/dnsmasq.pid"
fi

echo "All services stopped"

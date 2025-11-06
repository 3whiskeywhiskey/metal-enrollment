#!/usr/bin/env bash
set -euo pipefail

# Build a simplified registration image for testing
# Since we don't have Nix in the container, we'll create a mock registration script

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$TEST_DIR/run/env.sh" 2>/dev/null || true

IMAGES_DIR="${IMAGES_DIR:-$TEST_DIR/run/images}"

echo "=== Building Test Registration Image ==="

# Create registration images directory
mkdir -p "$IMAGES_DIR/registration"

# Create a minimal enrollment script that will run in the test environment
cat > "$IMAGES_DIR/registration/enroll-test.sh" <<'ENROLLMENT_SCRIPT'
#!/bin/bash
# Test enrollment script - simulates hardware detection

ENROLLMENT_URL="${1:-http://10.88.0.1:8080/api/v1/enroll}"

echo "Starting test enrollment..."
echo "Enrollment URL: $ENROLLMENT_URL"

# Generate random service tag
SERVICE_TAG="TEST$(date +%s)"
MAC_ADDRESS="52:54:00:$(printf '%02x:%02x:%02x' $((RANDOM%256)) $((RANDOM%256)) $((RANDOM%256)))"

echo "Service Tag: $SERVICE_TAG"
echo "MAC Address: $MAC_ADDRESS"

# Build enrollment payload
PAYLOAD=$(cat <<EOF
{
  "service_tag": "$SERVICE_TAG",
  "mac_address": "$MAC_ADDRESS",
  "hardware": {
    "manufacturer": "QEMU",
    "model": "Standard PC (Q35 + ICH9, 2009)",
    "serial_number": "$SERVICE_TAG",
    "bios_version": "1.0.0",
    "cpu": {
      "model": "QEMU Virtual CPU version 2.5+",
      "cores": 2,
      "threads": 2,
      "sockets": 1,
      "max_freq_mhz": 2000,
      "architecture": "x86_64"
    },
    "memory": {
      "total_bytes": 2147483648,
      "total_gb": 2.0,
      "modules": [
        {
          "slot": "DIMM0",
          "size_bytes": 2147483648,
          "type": "RAM",
          "speed": 0
        }
      ]
    },
    "disks": [
      {
        "device": "/dev/vda",
        "model": "QEMU HARDDISK",
        "size_bytes": 10737418240,
        "size_gb": 10.0,
        "type": "VIRTIO",
        "serial": "QM00001",
        "rotational": false
      }
    ],
    "nics": [
      {
        "name": "eth0",
        "mac_address": "$MAC_ADDRESS",
        "driver": "virtio_net",
        "speed": "1000Mbps",
        "link_status": "up",
        "pci_address": "0000:00:03.0"
      }
    ],
    "gpus": []
  }
}
EOF
)

echo "Sending enrollment request..."
echo "$PAYLOAD" | jq . || echo "$PAYLOAD"

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD" \
    "$ENROLLMENT_URL" 2>&1 || echo -e "\n000")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$RESPONSE" | head -n-1)

echo "HTTP Status: $HTTP_CODE"
echo "Response: $RESPONSE_BODY"

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    echo "✓ Enrollment successful!"
    exit 0
else
    echo "✗ Enrollment failed"
    exit 1
fi
ENROLLMENT_SCRIPT

chmod +x "$IMAGES_DIR/registration/enroll-test.sh"

echo "✓ Test registration script created at:"
echo "  $IMAGES_DIR/registration/enroll-test.sh"
echo ""
echo "This script simulates a machine booting and enrolling."
echo "Run it with: $IMAGES_DIR/registration/enroll-test.sh <enrollment-url>"

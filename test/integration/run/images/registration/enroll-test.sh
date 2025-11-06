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

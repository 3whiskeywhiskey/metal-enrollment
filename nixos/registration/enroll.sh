#!/usr/bin/env bash
set -euo pipefail

# Metal Enrollment Registration Script
# This script runs on the registration image to catalog hardware and enroll the machine

ENROLLMENT_URL="${ENROLLMENT_URL:-http://enrollment.local:8080/api/v1/enroll}"
LOG_FILE="/var/log/metal-enrollment.log"

log() {
    echo "[$(date -Iseconds)] $*" | tee -a "$LOG_FILE"
}

error() {
    log "ERROR: $*"
    exit 1
}

log "Starting metal enrollment registration..."

# Wait for network to be ready
log "Waiting for network..."
for i in {1..30}; do
    if ping -c 1 -W 1 8.8.8.8 &> /dev/null; then
        log "Network is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        error "Network timeout"
    fi
    sleep 2
done

# Detect service tag
log "Detecting service tag..."
SERVICE_TAG=""

if command -v dmidecode &> /dev/null; then
    SERVICE_TAG=$(dmidecode -s system-serial-number 2>/dev/null | tr -d '[:space:]' || echo "")
fi

if [ -z "$SERVICE_TAG" ] || [ "$SERVICE_TAG" = "Not" ] || [ "$SERVICE_TAG" = "Default" ]; then
    # Fallback to MAC address if service tag not available
    SERVICE_TAG=$(ip link show | grep -A1 "state UP" | grep ether | awk '{print $2}' | head -n1 | tr -d ':' | tr '[:lower:]' '[:upper:]')
fi

if [ -z "$SERVICE_TAG" ]; then
    error "Could not determine service tag or MAC address"
fi

log "Service tag: $SERVICE_TAG"

# Detect MAC address
MAC_ADDRESS=$(ip link show | grep -A1 "state UP" | grep ether | awk '{print $2}' | head -n1)
if [ -z "$MAC_ADDRESS" ]; then
    error "Could not determine MAC address"
fi

log "MAC address: $MAC_ADDRESS"

# Gather hardware information

# System information
log "Gathering system information..."
MANUFACTURER=$(dmidecode -s system-manufacturer 2>/dev/null || echo "Unknown")
MODEL=$(dmidecode -s system-product-name 2>/dev/null || echo "Unknown")
SERIAL_NUMBER=$(dmidecode -s system-serial-number 2>/dev/null || echo "Unknown")
BIOS_VERSION=$(dmidecode -s bios-version 2>/dev/null || echo "Unknown")

# CPU information
log "Gathering CPU information..."
CPU_MODEL=$(lscpu | grep "Model name" | cut -d: -f2 | xargs)
CPU_SOCKETS=$(lscpu | grep "Socket(s)" | cut -d: -f2 | xargs)
CPU_CORES=$(lscpu | grep "Core(s) per socket" | cut -d: -f2 | xargs)
CPU_THREADS=$(lscpu | grep "Thread(s) per core" | cut -d: -f2 | xargs)
CPU_MAX_MHZ=$(lscpu | grep "CPU max MHz" | cut -d: -f2 | xargs | cut -d. -f1)
CPU_ARCH=$(uname -m)

# Memory information
log "Gathering memory information..."
TOTAL_MEMORY_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
TOTAL_MEMORY_GB=$(echo "scale=2; $TOTAL_MEMORY_KB / 1024 / 1024" | bc)

# Detect memory modules
MEMORY_MODULES="[]"
if command -v dmidecode &> /dev/null; then
    MEMORY_MODULES=$(dmidecode -t memory | awk '
        /Memory Device/ { in_device=1; slot=""; size=""; type=""; speed=""; next }
        in_device && /Locator:/ && !/Bank/ { slot=$2; gsub(/[^a-zA-Z0-9]/, "", slot) }
        in_device && /Size:/ {
            if ($2 != "No") {
                size_val=$2
                size_unit=$3
                if (size_unit == "GB") size=size_val * 1024 * 1024 * 1024
                else if (size_unit == "MB") size=size_val * 1024 * 1024
            }
        }
        in_device && /Type:/ && !/Error/ { type=$2 }
        in_device && /Speed:/ && !/Unknown/ && !/Configured/ { speed=$2 }
        in_device && /^$/ {
            if (slot != "" && size != "") {
                if (first) printf ","
                printf "{\"slot\":\"%s\",\"size_bytes\":%s,\"type\":\"%s\",\"speed\":%s}", slot, size, type, speed
                first=1
            }
            in_device=0
        }
        BEGIN { printf "["; first=0 }
        END { printf "]" }
    ')
fi

# Disk information
log "Gathering disk information..."
DISKS_JSON="["
first=true
for disk in /dev/sd? /dev/nvme?n?; do
    if [ -b "$disk" ]; then
        if [ "$first" = false ]; then
            DISKS_JSON+=","
        fi
        first=false

        DISK_NAME=$(basename "$disk")
        DISK_SIZE=$(blockdev --getsize64 "$disk" 2>/dev/null || echo "0")
        DISK_SIZE_GB=$(echo "scale=2; $DISK_SIZE / 1024 / 1024 / 1024" | bc)
        DISK_MODEL=$(smartctl -i "$disk" 2>/dev/null | grep "Device Model" | cut -d: -f2 | xargs || echo "Unknown")
        DISK_SERIAL=$(smartctl -i "$disk" 2>/dev/null | grep "Serial Number" | cut -d: -f2 | xargs || echo "Unknown")
        DISK_ROTATIONAL=$([ -f "/sys/block/$DISK_NAME/queue/rotational" ] && cat "/sys/block/$DISK_NAME/queue/rotational" || echo "0")

        if [[ "$disk" == /dev/nvme* ]]; then
            DISK_TYPE="NVMe"
        elif [ "$DISK_ROTATIONAL" = "0" ]; then
            DISK_TYPE="SSD"
        else
            DISK_TYPE="HDD"
        fi

        DISKS_JSON+="{\"device\":\"$disk\",\"model\":\"$DISK_MODEL\",\"size_bytes\":$DISK_SIZE,\"size_gb\":$DISK_SIZE_GB,\"type\":\"$DISK_TYPE\",\"serial\":\"$DISK_SERIAL\",\"rotational\":$([ "$DISK_ROTATIONAL" = "1" ] && echo "true" || echo "false")}"
    fi
done
DISKS_JSON+="]"

# Network interface information
log "Gathering network interface information..."
NICS_JSON="["
first=true
for nic in /sys/class/net/*; do
    if [ -d "$nic" ] && [ "$(basename "$nic")" != "lo" ]; then
        if [ "$first" = false ]; then
            NICS_JSON+=","
        fi
        first=false

        NIC_NAME=$(basename "$nic")
        NIC_MAC=$(cat "$nic/address" 2>/dev/null || echo "00:00:00:00:00:00")
        NIC_DRIVER=$(readlink "$nic/device/driver" 2>/dev/null | xargs basename || echo "Unknown")
        NIC_SPEED=$(ethtool "$NIC_NAME" 2>/dev/null | grep Speed | cut -d: -f2 | xargs || echo "Unknown")
        NIC_LINK=$(cat "$nic/operstate" 2>/dev/null || echo "unknown")
        NIC_PCI=$(readlink "$nic/device" 2>/dev/null | xargs basename || echo "")

        NICS_JSON+="{\"name\":\"$NIC_NAME\",\"mac_address\":\"$NIC_MAC\",\"driver\":\"$NIC_DRIVER\",\"speed\":\"$NIC_SPEED\",\"link_status\":\"$NIC_LINK\",\"pci_address\":\"$NIC_PCI\"}"
    fi
done
NICS_JSON+="]"

# GPU information (if any)
log "Gathering GPU information..."
GPUS_JSON="[]"
if command -v lspci &> /dev/null; then
    GPUS_JSON=$(lspci | grep -i vga | awk -F: '{
        pci=$1
        gsub(/^[ \t]+/, "", pci)
        desc=$3
        gsub(/^[ \t]+/, "", desc)
        vendor=""
        model=desc

        if (desc ~ /NVIDIA/) vendor="NVIDIA"
        else if (desc ~ /AMD|ATI/) vendor="AMD"
        else if (desc ~ /Intel/) vendor="Intel"
        else vendor="Unknown"

        if (first) printf ","
        printf "{\"model\":\"%s\",\"vendor\":\"%s\",\"pci_address\":\"%s\"}", model, vendor, pci
        first=1
    } BEGIN { printf "["; first=0 } END { printf "]" }')
fi

# Build JSON payload
log "Building enrollment payload..."
PAYLOAD=$(cat <<EOF
{
  "service_tag": "$SERVICE_TAG",
  "mac_address": "$MAC_ADDRESS",
  "hardware": {
    "manufacturer": "$MANUFACTURER",
    "model": "$MODEL",
    "serial_number": "$SERIAL_NUMBER",
    "bios_version": "$BIOS_VERSION",
    "cpu": {
      "model": "$CPU_MODEL",
      "cores": ${CPU_CORES:-1},
      "threads": ${CPU_THREADS:-1},
      "sockets": ${CPU_SOCKETS:-1},
      "max_freq_mhz": ${CPU_MAX_MHZ:-0},
      "architecture": "$CPU_ARCH"
    },
    "memory": {
      "total_bytes": $(($TOTAL_MEMORY_KB * 1024)),
      "total_gb": $TOTAL_MEMORY_GB,
      "modules": $MEMORY_MODULES
    },
    "disks": $DISKS_JSON,
    "nics": $NICS_JSON,
    "gpus": $GPUS_JSON
  }
}
EOF
)

# Send enrollment request
log "Sending enrollment request to $ENROLLMENT_URL..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD" \
    "$ENROLLMENT_URL" || echo -e "\n000")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    log "Enrollment successful!"
    log "Response: $RESPONSE_BODY"

    # Show success message on console
    echo ""
    echo "=========================================="
    echo "  ENROLLMENT SUCCESSFUL"
    echo "=========================================="
    echo "Service Tag: $SERVICE_TAG"
    echo "Status: Enrolled"
    echo ""
    echo "This machine is now registered with the"
    echo "Metal Enrollment system. An administrator"
    echo "can configure and deploy a custom NixOS"
    echo "image for this machine."
    echo ""
    echo "You can safely reboot this machine."
    echo "=========================================="
    echo ""

    # Wait a bit before exiting
    sleep 10
else
    error "Enrollment failed with HTTP code $HTTP_CODE: $RESPONSE_BODY"
fi

log "Enrollment process completed"

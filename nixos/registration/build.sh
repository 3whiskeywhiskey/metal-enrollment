#!/usr/bin/env bash
set -euo pipefail

# Build script for the registration image

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$SCRIPT_DIR/../../images/registration}"

echo "Building NixOS registration image..."
echo "Output directory: $OUTPUT_DIR"

# Build the netboot ramdisk
nix-build '<nixpkgs/nixos>' \
    -A config.system.build.netbootRamdisk \
    -I nixos-config="$SCRIPT_DIR/configuration.nix" \
    -o "$SCRIPT_DIR/result"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Copy kernel and initrd to output directory
echo "Copying artifacts..."
cp "$SCRIPT_DIR/result/bzImage" "$OUTPUT_DIR/bzImage"
cp "$SCRIPT_DIR/result/initrd" "$OUTPUT_DIR/initrd"

echo "Build complete!"
echo "Kernel: $OUTPUT_DIR/bzImage"
echo "Initrd: $OUTPUT_DIR/initrd"
echo ""
echo "To deploy, copy these files to your HTTP server:"
echo "  scp $OUTPUT_DIR/* user@server:/var/lib/metal-enrollment/images/registration/"

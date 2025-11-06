# Setup Guide

This guide walks you through setting up Metal Enrollment from scratch.

## Prerequisites

### Infrastructure Requirements

1. **Kubernetes Cluster**
   - At least one NixOS node for running the image builder
   - Storage classes available:
     - `local-path` or equivalent for database
     - `nfs` or `ReadWriteMany` storage for shared image storage
   - Ingress controller (nginx recommended)

2. **Network Infrastructure**
   - DHCP server configured for PXE boot
   - TFTP server with `snp.efi` configured
   - Network access from machines to enrollment services

3. **Development Tools** (for building)
   - Go 1.22 or later
   - Docker or podman
   - Nix package manager
   - kubectl

### MikroTik RouterOS Configuration

Your existing DHCP/TFTP setup should work. Ensure:

1. DHCP is configured to provide next-server and boot-filename
2. TFTP serves `snp.efi`
3. The HTTP container is replaced with the iPXE server

Example RouterOS DHCP network config:
```
/ip dhcp-server network
add address=192.168.1.0/24 gateway=192.168.1.1 \
    next-server=192.168.1.1 \
    boot-file-name=snp.efi
```

## Step-by-Step Setup

### 1. Clone and Build

```bash
git clone https://github.com/yourusername/metal-enrollment.git
cd metal-enrollment

# Install Go dependencies
make deps

# Build all binaries
make build
```

### 2. Build Registration Image

```bash
# Build the NixOS registration image
make build-registration

# This creates:
# - images/registration/bzImage
# - images/registration/initrd
```

### 3. Configure for Your Environment

Edit `deployments/kubernetes/configmap.yaml`:

```yaml
data:
  # Your iPXE server's public IP/URL
  BASE_URL: "http://192.168.1.100"

  # Enrollment API endpoint (accessible from booting machines)
  ENROLLMENT_URL: "http://192.168.1.100/api/v1/enroll"

  # Internal API URL (within cluster)
  API_URL: "http://enrollment-server.metal-enrollment.svc.cluster.local:8080/api/v1"
```

### 4. Label Your NixOS Node

The builder needs to run on a NixOS node:

```bash
kubectl label nodes <nixos-node-name> nixos=true
```

Update `deployments/kubernetes/deployment-builder.yaml` to use this label:

```yaml
nodeSelector:
  nixos: "true"
```

### 5. Build and Push Docker Images

```bash
# Build images
make docker-build

# Tag for your registry
export REGISTRY=your-registry.com
docker tag metal-enrollment/server:latest $REGISTRY/metal-enrollment-server:latest
docker tag metal-enrollment/builder:latest $REGISTRY/metal-enrollment-builder:latest
docker tag metal-enrollment/ipxe-server:latest $REGISTRY/metal-enrollment-ipxe-server:latest

# Push to registry
docker push $REGISTRY/metal-enrollment-server:latest
docker push $REGISTRY/metal-enrollment-builder:latest
docker push $REGISTRY/metal-enrollment-ipxe-server:latest
```

Update the image references in the deployment manifests to use your registry.

### 6. Deploy to Kubernetes

```bash
# Deploy all components
make deploy

# Check status
kubectl get pods -n metal-enrollment
kubectl get svc -n metal-enrollment
```

### 7. Deploy Registration Image

Copy the registration image to the iPXE server's image directory:

```bash
# Get the pod name
POD=$(kubectl get pods -n metal-enrollment -l app=enrollment-ipxe-server -o jsonpath='{.items[0].metadata.name}')

# Copy registration image files
kubectl cp images/registration/bzImage metal-enrollment/$POD:/images/registration/bzImage
kubectl cp images/registration/initrd metal-enrollment/$POD:/images/registration/initrd
```

Alternatively, if using a shared volume, copy directly to the storage:

```bash
# Mount the PVC and copy files
kubectl run -n metal-enrollment -i --tty --rm copy-images \
  --image=alpine --restart=Never \
  --overrides='{"spec":{"containers":[{"name":"copy-images","image":"alpine","command":["sleep","3600"],"volumeMounts":[{"name":"images","mountPath":"/images"}]}],"volumes":[{"name":"images","persistentVolumeClaim":{"claimName":"metal-enrollment-images"}}]}}'

# In another terminal
kubectl cp images/registration/bzImage metal-enrollment/copy-images:/images/registration/
kubectl cp images/registration/initrd metal-enrollment/copy-images:/images/registration/
```

### 8. Configure iPXE Server Access

The iPXE server needs to be accessible from your PXE booting machines. Options:

**Option A: LoadBalancer** (if supported)
```bash
kubectl get svc -n metal-enrollment enrollment-ipxe
# Note the external IP
```

**Option B: NodePort**
```yaml
# Edit deployment-ipxe-server.yaml
spec:
  type: NodePort
  ports:
  - port: 80
    nodePort: 30080  # Choose available port
```

**Option C: Run on RDS** (your current setup)

Deploy just the iPXE server as a container on your RDS:
```bash
# Build and export image
docker save metal-enrollment/ipxe-server:latest | gzip > ipxe-server.tar.gz

# Copy to RDS and load
# Configure to mount images volume and run
```

### 9. Update Your DHCP/TFTP Setup

Ensure `snp.efi` chains to your iPXE server:

```ipxe
#!ipxe
# In your snp.efi or chain-loaded script

# Get service tag (Dell SMBIOS)
iseq ${manufacturer} Dell && chain http://192.168.1.100/nixos/machines/${asset}.ipxe ||

# Fallback to MAC-based for non-Dell
set servicetag ${mac}
chain http://192.168.1.100/nixos/machines/${servicetag}.ipxe
```

### 10. Access the Dashboard

```bash
# Port-forward for testing
kubectl port-forward -n metal-enrollment svc/enrollment-server 8080:8080

# Access at http://localhost:8080
```

Or configure ingress:

```bash
# Get ingress IP
kubectl get ingress -n metal-enrollment

# Add to /etc/hosts
echo "<ingress-ip> enrollment.metal.local" >> /etc/hosts

# Access at http://enrollment.metal.local
```

## Testing

### Test Enrollment API

```bash
curl http://localhost:8080/api/v1/enroll -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "service_tag": "TEST001",
    "mac_address": "00:11:22:33:44:55",
    "hardware": {
      "manufacturer": "Test",
      "model": "TestServer",
      "serial_number": "TEST001",
      "bios_version": "1.0",
      "cpu": {
        "model": "Test CPU",
        "cores": 4,
        "threads": 8,
        "sockets": 1,
        "max_freq_mhz": 3000,
        "architecture": "x86_64"
      },
      "memory": {
        "total_bytes": 17179869184,
        "total_gb": 16,
        "modules": []
      },
      "disks": [],
      "nics": [],
      "gpus": []
    }
  }'
```

Check the dashboard - you should see the test machine.

### Test iPXE Server

```bash
# Forward iPXE server port
POD=$(kubectl get pods -n metal-enrollment -l app=enrollment-ipxe-server -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n metal-enrollment $POD 8081:8080

# Request iPXE script
curl http://localhost:8081/nixos/machines/TEST001.ipxe

# Should return registration iPXE script
```

### Test with Real Hardware

1. Boot a machine via PXE
2. Watch the console output
3. Check enrollment logs:
   ```bash
   kubectl logs -n metal-enrollment -l app=enrollment-server -f
   ```
4. Machine should appear in dashboard within 1-2 minutes

## Troubleshooting

### Registration Image Not Booting

- Check TFTP server logs
- Verify network connectivity
- Check DHCP provides correct next-server
- Test iPXE script URL manually

### Enrollment Fails

- Check network connectivity from booting machine
- Verify ENROLLMENT_URL is accessible
- Check enrollment server logs:
  ```bash
  kubectl logs -n metal-enrollment deployment/enrollment-server
  ```

### Builder Fails

- Ensure running on NixOS node
- Check /nix is mounted
- Verify privileged mode enabled
- Check builder logs:
  ```bash
  kubectl logs -n metal-enrollment deployment/enrollment-builder
  ```

### Images Not Serving

- Verify PVC is mounted
- Check file permissions
- Ensure registration image was copied
- Check iPXE server logs:
  ```bash
  kubectl logs -n metal-enrollment deployment/enrollment-ipxe-server
  ```

## Next Steps

1. Configure SSH keys in machine template
2. Set up proper ingress/DNS
3. Add authentication to API
4. Create machine configurations
5. Enroll your first machine!

See [USAGE.md](USAGE.md) for detailed usage instructions.

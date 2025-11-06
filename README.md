# Metal Enrollment

A comprehensive bare metal machine enrollment and provisioning system for automated infrastructure. Similar to Canonical's MaaS, this system automatically catalogs hardware and serves custom NixOS PXE boot images to enrolled machines.

## Features

- **Automated Hardware Discovery**: Registration image automatically detects and catalogs hardware details
- **Web Dashboard**: Manage enrolled machines, view hardware specs, and configure deployments
- **Custom NixOS Images**: Build and serve machine-specific NixOS configurations
- **PXE Boot Integration**: Seamless integration with existing DHCP/TFTP infrastructure
- **RESTful API**: Full API for programmatic access and automation
- **Kubernetes Native**: Designed to run in Kubernetes clusters

## Architecture

The system consists of four main components:

1. **Enrollment Server** - Central API and web dashboard for machine management
2. **Image Builder** - NixOS image builder service (runs on NixOS nodes)
3. **iPXE Server** - Serves iPXE scripts and boot images
4. **Registration Image** - Minimal NixOS image that boots on unknown machines and reports hardware

### Flow

```
┌─────────────────┐
│ Unknown Machine │ PXE Boot → TFTP serves snp.efi
└─────────────────┘              │
                                 ▼
                    Check service tag → /nixos/machines/<tag>.ipxe
                                 │
                      ┌──────────┴──────────┐
                      │                     │
                 Not Found              Found + Ready
                      │                     │
                      ▼                     ▼
            Registration Image      Custom NixOS Image
                      │
                      ▼
            Hardware Detection
                      │
                      ▼
           POST /api/v1/enroll
                      │
                      ▼
            ┌──────────────────────┐
            │  Enrollment Server   │
            │  - Database          │
            │  - Web Dashboard     │
            │  - API               │
            └──────────────────────┘
                      │
                      ▼
           Admin configures machine
                      │
                      ▼
           Trigger build → Image Builder
                                 │
                                 ▼
                      Build custom NixOS image
                                 │
                                 ▼
                      Deploy to iPXE server
                                 │
                                 ▼
                      Machine reboots → Custom image
```

## Quick Start

### Prerequisites

- Kubernetes cluster with at least one NixOS node (for image builder)
- Existing PXE infrastructure (DHCP + TFTP)
- Storage class for persistent volumes
- Go 1.22+ (for local development)
- Nix (for building registration image)

### Building

```bash
# Build all binaries
make build

# Build registration image
make build-registration

# Build Docker images
make docker-build
```

### Deploying to Kubernetes

1. **Update configuration**:
   Edit `deployments/kubernetes/configmap.yaml` with your environment settings:
   - `BASE_URL`: Public URL where iPXE server is accessible
   - `ENROLLMENT_URL`: URL for enrollment API
   - `API_URL`: Internal API URL

2. **Build and push Docker images**:
   ```bash
   make docker-build
   # Tag and push to your registry
   docker tag metal-enrollment/server:latest your-registry/metal-enrollment-server:latest
   docker tag metal-enrollment/builder:latest your-registry/metal-enrollment-builder:latest
   docker tag metal-enrollment/ipxe-server:latest your-registry/metal-enrollment-ipxe-server:latest
   # Push images...
   ```

3. **Deploy to Kubernetes**:
   ```bash
   make deploy
   ```

4. **Build and deploy registration image**:
   ```bash
   cd nixos/registration
   ./build.sh

   # Copy to iPXE server images directory
   kubectl cp bzImage metal-enrollment/enrollment-ipxe-server-xxx:/images/registration/
   kubectl cp initrd metal-enrollment/enrollment-ipxe-server-xxx:/images/registration/
   ```

### PXE Infrastructure Setup

Your existing MikroTik RDS setup should work with minimal changes:

1. **TFTP**: Continue serving `snp.efi` via TFTP
2. **HTTP Container**: Replace with the `ipxe-server` container
3. **Service Tag Detection**: Already in place in `snp.efi`

The iPXE script in `snp.efi` should chain to:
```
http://<ipxe-server>/nixos/machines/<servicetag>.ipxe
```

## Usage

### Enrolling a New Machine

1. **Boot machine via PXE**
   - Machine boots from network
   - Receives `snp.efi` from TFTP
   - Requests iPXE script based on service tag
   - Gets registration image (unknown machine)

2. **Automatic Enrollment**
   - Registration image boots
   - Hardware detection runs
   - Machine enrolls automatically
   - Appears in dashboard

3. **Configure Machine**
   - Access dashboard at `http://<enrollment-server>:8080`
   - Click on enrolled machine
   - Set hostname and description
   - Add NixOS configuration
   - Click "Save Configuration"

4. **Build Custom Image**
   - Click "Build" button
   - Image builder creates custom NixOS image
   - Image deployed to iPXE server
   - Status shows "Ready"

5. **Reboot Machine**
   - Machine reboots
   - Receives custom image
   - Boots into configured system

### API Usage

#### Enroll a Machine
```bash
curl -X POST http://localhost:8080/api/v1/enroll \
  -H "Content-Type: application/json" \
  -d '{
    "service_tag": "ABC123",
    "mac_address": "00:11:22:33:44:55",
    "hardware": { ... }
  }'
```

#### List Machines
```bash
curl http://localhost:8080/api/v1/machines
```

#### Get Machine Details
```bash
curl http://localhost:8080/api/v1/machines/<machine-id>
```

#### Update Machine
```bash
curl -X PUT http://localhost:8080/api/v1/machines/<machine-id> \
  -H "Content-Type: application/json" \
  -d '{
    "hostname": "server01",
    "nixos_config": "{ ... }"
  }'
```

#### Trigger Build
```bash
curl -X POST http://localhost:8080/api/v1/machines/<machine-id>/build
```

## Configuration

### Environment Variables

#### Enrollment Server
- `DB_DRIVER`: Database driver (`sqlite3` or `postgres`)
- `DB_DSN`: Database connection string
- `LISTEN_ADDR`: HTTP listen address (default: `:8080`)
- `BUILDER_URL`: URL of builder service

#### Image Builder
- `DB_DRIVER`: Database driver
- `DB_DSN`: Database connection string
- `LISTEN_ADDR`: HTTP listen address (default: `:8081`)
- `BUILD_DIR`: Temporary build directory
- `OUTPUT_DIR`: Output directory for built images
- `NIXOS_DIR`: NixOS configurations directory

#### iPXE Server
- `BASE_URL`: Base URL for iPXE scripts
- `ENROLLMENT_URL`: Enrollment API URL
- `API_URL`: API base URL
- `IMAGES_DIR`: Directory for serving images
- `LISTEN_ADDR`: HTTP listen address (default: `:8080`)

## Development

### Running Locally

```bash
# Start enrollment server
go run cmd/server/main.go

# Start builder (on NixOS machine)
go run cmd/builder/main.go

# Start iPXE server
go run cmd/ipxe-server/main.go
```

### Project Structure

```
metal-enrollment/
├── cmd/                      # Command-line applications
│   ├── server/              # Main enrollment server
│   ├── builder/             # Image builder service
│   └── ipxe-server/         # iPXE/image serving
├── pkg/                      # Shared packages
│   ├── api/                 # API server implementation
│   ├── database/            # Database layer
│   ├── models/              # Data models
│   └── web/                 # Web dashboard
├── nixos/                    # NixOS configurations
│   ├── registration/        # Registration image config
│   └── machine-template/    # Template for custom images
├── deployments/              # Deployment configurations
│   ├── kubernetes/          # Kubernetes manifests
│   └── docker/              # Dockerfiles
└── docs/                     # Documentation
```

## Documentation

- [Setup Guide](docs/SETUP.md) - Detailed setup instructions
- [Architecture](docs/ARCHITECTURE.md) - System architecture and design

## Security Considerations

- Registration image runs as root (needed for hardware detection)
- Default root password set in registration image - change in production
- No authentication on API by default - add auth layer for production
- Builder service requires privileged container for Nix builds
- SSH keys should be added to machine configurations

## Roadmap

- [ ] Add authentication and authorization
- [ ] Support for PostgreSQL
- [ ] Machine grouping and bulk operations
- [ ] Integration with configuration management (Terraform, Ansible)
- [ ] IPMI/BMC integration for remote power control
- [ ] Machine metrics and monitoring
- [ ] Automated testing of boot images
- [ ] Support for non-Dell hardware (generic service tag detection)

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

MIT License - see LICENSE file for details

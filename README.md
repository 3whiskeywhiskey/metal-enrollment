# Metal Enrollment

A comprehensive bare metal machine enrollment and provisioning system for automated infrastructure. Similar to Canonical's MaaS, this system automatically catalogs hardware and serves custom NixOS PXE boot images to enrolled machines.

## Features

- **Automated Hardware Discovery**: Registration image automatically detects and catalogs hardware details
- **Web Dashboard**: Manage enrolled machines, view hardware specs, and configure deployments
- **Custom NixOS Images**: Build and serve machine-specific NixOS configurations
- **PXE Boot Integration**: Seamless integration with existing DHCP/TFTP infrastructure
- **RESTful API**: Full API for programmatic access and automation
- **Kubernetes Native**: Designed to run in Kubernetes clusters
- **Authentication & Authorization**: JWT-based authentication with role-based access control (Admin, Operator, Viewer)
- **PostgreSQL Support**: Production-ready PostgreSQL database support alongside SQLite
- **Machine Grouping**: Organize machines into logical groups for easier management
- **Bulk Operations**: Perform operations on multiple machines simultaneously
- **IPMI/BMC Integration**: Remote power control and sensor monitoring via IPMI
- **Machine Metrics**: Collect and monitor CPU, memory, disk, and network metrics
- **Prometheus Export**: Export metrics in Prometheus format for monitoring
- **Terraform Provider**: Manage machines using Terraform infrastructure as code
- **Ansible Integration**: Dynamic inventory for Ansible automation
- **Image Testing**: Automated testing framework for boot images

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

#### Authentication

##### Login
```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "your-password"
  }'
```

Response includes a JWT token:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-01-02T15:04:05Z",
  "user": { ... }
}
```

##### Using Authentication
Add the token to subsequent requests:
```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/machines
```

#### Machine Management

##### Enroll a Machine (no auth required)
```bash
curl -X POST http://localhost:8080/api/v1/enroll \
  -H "Content-Type: application/json" \
  -d '{
    "service_tag": "ABC123",
    "mac_address": "00:11:22:33:44:55",
    "hardware": { ... }
  }'
```

##### List Machines
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/machines
```

##### Get Machine Details
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/machines/<machine-id>
```

##### Update Machine (requires Operator or Admin role)
```bash
curl -X PUT http://localhost:8080/api/v1/machines/<machine-id> \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "hostname": "server01",
    "nixos_config": "{ ... }"
  }'
```

##### Trigger Build (requires Operator or Admin role)
```bash
curl -X POST http://localhost:8080/api/v1/machines/<machine-id>/build \
  -H "Authorization: Bearer <token>"
```

#### Group Management

##### Create Group (requires Operator or Admin role)
```bash
curl -X POST http://localhost:8080/api/v1/groups \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-servers",
    "description": "Production web servers",
    "tags": ["production", "web"]
  }'
```

##### List Groups
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/groups
```

##### Add Machine to Group (requires Operator or Admin role)
```bash
curl -X PUT http://localhost:8080/api/v1/groups/<group-id>/machines/<machine-id> \
  -H "Authorization: Bearer <token>"
```

##### Get Group Machines
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/groups/<group-id>/machines
```

#### Bulk Operations (requires Operator or Admin role)

##### Bulk Update Machines
```bash
curl -X POST http://localhost:8080/api/v1/bulk \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "group_id": "group-id",
    "operation": "update",
    "data": {
      "description": "Updated via bulk operation"
    }
  }'
```

##### Bulk Build Machines
```bash
curl -X POST http://localhost:8080/api/v1/bulk \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "machine_ids": ["id1", "id2", "id3"],
    "operation": "build"
  }'
```

#### Power Control (IPMI/BMC)

##### Configure BMC
```bash
curl -X PUT http://localhost:8080/api/v1/machines/<machine-id> \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "bmc_info": {
      "ip_address": "10.0.0.100",
      "username": "admin",
      "password": "password",
      "type": "IPMI",
      "port": 623,
      "enabled": true
    }
  }'
```

##### Power Control Operations
```bash
# Power on
curl -X POST http://localhost:8080/api/v1/machines/<machine-id>/power \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"operation": "on"}'

# Power off
curl -X POST http://localhost:8080/api/v1/machines/<machine-id>/power \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"operation": "off"}'

# Reset
curl -X POST http://localhost:8080/api/v1/machines/<machine-id>/power \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"operation": "reset"}'

# Get power status
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/machines/<machine-id>/power/status
```

##### Get BMC Sensor Readings
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/machines/<machine-id>/bmc/sensors
```

#### Machine Metrics

##### Submit Metrics (from machine)
```bash
curl -X POST http://localhost:8080/api/v1/machines/<machine-id>/metrics \
  -H "Content-Type: application/json" \
  -d '{
    "cpu_usage_percent": 45.2,
    "memory_used_bytes": 8589934592,
    "memory_total_bytes": 17179869184,
    "disk_used_bytes": 107374182400,
    "disk_total_bytes": 536870912000,
    "network_rx_bytes": 1073741824,
    "network_tx_bytes": 536870912,
    "load_average_1": 2.5,
    "load_average_5": 2.1,
    "load_average_15": 1.8,
    "temperature": 45.0,
    "power_state": "on",
    "uptime": 86400
  }'
```

##### Get Latest Metrics
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/machines/<machine-id>/metrics/latest
```

##### Get Metrics History
```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/machines/<machine-id>/metrics/history?since=24h&limit=1000"
```

##### Prometheus Metrics Export
```bash
# Public endpoint - no authentication required
curl http://localhost:8080/api/v1/metrics
```

#### Image Testing

##### Create Image Test
```bash
curl -X POST http://localhost:8080/api/v1/image-tests \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "image_path": "/images/custom-image.img",
    "image_type": "custom",
    "test_type": "boot",
    "machine_id": "<machine-id>"
  }'
```

##### List Image Tests
```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/image-tests?image_type=custom&limit=50"
```

#### User Management (Admin only)

##### Create User
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "operator1",
    "email": "operator@example.com",
    "password": "secure-password",
    "role": "operator"
  }'
```

##### List Users
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/users
```

## Configuration Management Integrations

### Terraform Provider

The Metal Enrollment Terraform provider allows you to manage machines using Infrastructure as Code.

See [integrations/terraform/README.md](integrations/terraform/README.md) for full documentation.

Example usage:
```hcl
provider "metal-enrollment" {
  api_url = "http://localhost:8080"
  token   = var.metal_enrollment_token
}

resource "metal-enrollment_machine" "web_server" {
  service_tag  = "ABC123"
  hostname     = "web-server-01"
  description  = "Production web server"
  nixos_config = file("${path.module}/nixos-config.nix")

  bmc {
    ip_address = "10.0.0.100"
    username   = "admin"
    password   = var.bmc_password
    enabled    = true
  }
}
```

### Ansible Dynamic Inventory

The Ansible dynamic inventory script automatically discovers machines and groups them for automation.

Setup:
```bash
# Make the script executable
chmod +x integrations/ansible/inventory.py

# Configure environment
export METAL_ENROLLMENT_URL="http://localhost:8080"
export METAL_ENROLLMENT_TOKEN="your-jwt-token"  # Optional

# Test the inventory
./integrations/ansible/inventory.py --list

# Use with ansible
ansible -i integrations/ansible/inventory.py all -m ping

# Use with ansible-playbook
ansible-playbook -i integrations/ansible/inventory.py site.yml
```

Machines are automatically grouped by:
- **Status**: `status_enrolled`, `status_ready`, `status_provisioned`, etc.
- **Custom groups**: Any groups created via the API

## Configuration

### Environment Variables

#### Enrollment Server
- `DB_DRIVER`: Database driver (`sqlite3` or `postgres`)
- `DB_DSN`: Database connection string
- `LISTEN_ADDR`: HTTP listen address (default: `:8080`)
- `BUILDER_URL`: URL of builder service
- `ENABLE_AUTH`: Enable authentication (default: `true`)
- `JWT_SECRET`: Secret key for JWT token signing (change in production!)
- `JWT_EXPIRY`: JWT token expiration duration (default: `24h`)

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

- **Authentication**: JWT-based authentication is enabled by default
  - Change the default `JWT_SECRET` in production
  - Default admin credentials (admin/admin) should be changed immediately
- **Authorization**: Role-based access control with three levels:
  - **Admin**: Full system access (user management, all operations)
  - **Operator**: Machine and group management (cannot manage users)
  - **Viewer**: Read-only access to machines and groups
- **Database**:
  - Use PostgreSQL in production with proper credentials
  - SQLite is suitable for development/testing only
- **Registration**: Machine enrollment endpoint is public (by design)
- **Builder Service**: Requires privileged container for Nix builds
- **SSH Keys**: Should be added to machine configurations for secure access

### Getting Started with Authentication

1. **Start the server with admin creation**:
   ```bash
   ./server --create-admin
   ```

2. **Login to get a token**:
   ```bash
   TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login \
     -H "Content-Type: application/json" \
     -d '{"username":"admin","password":"admin"}' | jq -r '.token')
   ```

3. **Change admin password**:
   ```bash
   curl -X PUT http://localhost:8080/api/v1/users/<admin-id> \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"password":"new-secure-password"}'
   ```

4. **Create additional users**:
   ```bash
   curl -X POST http://localhost:8080/api/v1/users \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "username":"operator1",
       "email":"operator@example.com",
       "password":"secure-password",
       "role":"operator"
     }'
   ```

### Disabling Authentication (Not Recommended)

For development or testing, you can disable authentication:
```bash
./server --enable-auth=false
# or
export ENABLE_AUTH=false
./server
```

## Roadmap

- [x] Add authentication and authorization
- [x] Support for PostgreSQL
- [x] Machine grouping and bulk operations
- [x] Integration with configuration management (Terraform, Ansible)
- [x] IPMI/BMC integration for remote power control
- [x] Machine metrics and monitoring
- [x] Automated testing of boot images
- [ ] Support for non-Dell hardware (generic service tag detection)
- [ ] Webhook notifications for machine events
- [ ] Advanced filtering and search for machines
- [ ] Machine templates for common configurations

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

MIT License - see LICENSE file for details

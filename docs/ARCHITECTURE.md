# Architecture

This document describes the architecture and design decisions of the Metal Enrollment system.

## Overview

Metal Enrollment is a bare metal provisioning system that automates hardware discovery and deployment of custom NixOS images via PXE boot. It's designed to integrate with existing PXE infrastructure while providing a modern, cloud-native management interface.

## Components

### 1. Enrollment Server

**Purpose**: Central API and web dashboard for machine management.

**Technology**: Go, SQLite/PostgreSQL, HTML templates

**Responsibilities**:
- Accept hardware enrollment requests
- Store machine metadata and configurations
- Provide web dashboard for machine management
- Coordinate with builder service
- Expose RESTful API

**APIs**:
- `POST /api/v1/enroll` - Enroll new machine
- `GET /api/v1/machines` - List machines
- `GET /api/v1/machines/{id}` - Get machine details
- `PUT /api/v1/machines/{id}` - Update machine
- `POST /api/v1/machines/{id}/build` - Trigger build
- `GET /api/v1/builds/{id}` - Get build status

**Database Schema**:
```sql
machines:
  - id (primary key)
  - service_tag (unique)
  - mac_address
  - status (enrolled, configured, building, ready, failed)
  - hostname
  - description
  - hardware (JSON/JSONB)
  - nixos_config
  - last_build_id
  - enrolled_at, updated_at, last_seen_at

builds:
  - id (primary key)
  - machine_id (foreign key)
  - status (pending, building, success, failed)
  - config
  - log_output
  - error
  - artifact_url
  - created_at, completed_at
```

### 2. Image Builder

**Purpose**: Build custom NixOS netboot images.

**Technology**: Go, Nix

**Responsibilities**:
- Poll for pending build jobs
- Execute NixOS builds
- Extract kernel and initrd
- Deploy artifacts to shared storage
- Update build status

**Build Process**:
1. Receive build request (pull from database)
2. Create temporary build directory
3. Write machine configuration to `configuration.nix`
4. Execute `nix-build` to create netboot ramdisk
5. Extract kernel (`bzImage`) and initrd
6. Copy to shared storage at `/images/machines/{service_tag}/`
7. Update build status and machine status

**Requirements**:
- Must run on NixOS host
- Needs access to Nix store (`/nix`)
- Requires privileged container for Nix builds
- Shared storage with iPXE server

### 3. iPXE Server

**Purpose**: Serve iPXE scripts and boot images.

**Technology**: Go, HTTP file server

**Responsibilities**:
- Generate iPXE scripts based on machine state
- Serve kernel and initrd files
- Provide registration image for unknown machines
- Check machine enrollment status

**Request Flow**:
```
PXE Boot → snp.efi → GET /nixos/machines/{servicetag}.ipxe
                              ↓
                    Check machine in database
                              ↓
                   ┌──────────┴──────────┐
                   │                     │
              Not found              Found + Ready
                   │                     │
                   ▼                     ▼
       registration.ipxe        machine-specific.ipxe
       (kernel + initrd          (custom kernel + initrd
        from /images/             from /images/machines/
         registration/)            {servicetag}/)
```

**iPXE Script Templates**:

Registration:
```ipxe
#!ipxe
kernel {base}/images/registration/bzImage init=... enrollment_url=...
initrd {base}/images/registration/initrd
boot
```

Custom Machine:
```ipxe
#!ipxe
kernel {base}/images/machines/{tag}/bzImage init=...
initrd {base}/images/machines/{tag}/initrd
boot
```

### 4. Registration Image

**Purpose**: Minimal NixOS image for hardware detection and enrollment.

**Technology**: NixOS

**Boot Process**:
1. Boot via iPXE
2. Initialize network
3. Run hardware detection script
4. POST to enrollment API
5. Display success/failure message

**Hardware Detection**:
- Service tag (DMI)
- MAC addresses
- CPU info (lscpu)
- Memory (dmidecode, /proc/meminfo)
- Disks (blockdev, smartctl)
- NICs (sysfs, ethtool)
- GPUs (lspci)

**Enrollment Payload**:
```json
{
  "service_tag": "ABC123",
  "mac_address": "00:11:22:33:44:55",
  "hardware": {
    "manufacturer": "Dell Inc.",
    "model": "PowerEdge R740",
    "cpu": { ... },
    "memory": { ... },
    "disks": [ ... ],
    "nics": [ ... ]
  }
}
```

## Data Flow

### Machine Enrollment Flow

```
┌──────────┐
│ Unknown  │
│ Machine  │
└────┬─────┘
     │ 1. PXE Boot
     ▼
┌──────────┐
│   TFTP   │ 2. Serve snp.efi
└────┬─────┘
     │
     ▼
┌──────────┐
│ snp.efi  │ 3. Chain to iPXE server
└────┬─────┘      GET /nixos/machines/{tag}.ipxe
     │
     ▼
┌──────────────┐
│ iPXE Server  │ 4. Machine not found
└────┬─────────┘    Return registration.ipxe
     │
     ▼
┌─────────────────┐
│ Registration    │ 5. Boot & detect hardware
│ Image           │
└────┬────────────┘
     │ 6. POST /api/v1/enroll
     ▼
┌──────────────────┐
│ Enrollment       │ 7. Create machine record
│ Server           │    Status: enrolled
└──────────────────┘
```

### Machine Configuration Flow

```
┌──────────┐
│  Admin   │
└────┬─────┘
     │ 1. Access dashboard
     ▼
┌──────────────────┐
│ Web Dashboard    │ 2. View enrolled machines
└────┬─────────────┘
     │ 3. Select machine
     │    Set hostname, config
     ▼
┌──────────────────┐
│ Enrollment       │ 4. Update machine
│ Server           │    Status: configured
└──────────────────┘
```

### Image Build Flow

```
┌──────────┐
│  Admin   │
└────┬─────┘
     │ 1. Click "Build"
     ▼
┌──────────────────┐
│ Enrollment       │ 2. Create build record
│ Server           │    Status: pending
└────┬─────────────┘    Update machine: building
     │
     │ 3. Poll for pending builds
     ▼
┌──────────────────┐
│ Image Builder    │ 4. Execute nix-build
└────┬─────────────┘
     │ 5. Copy artifacts
     ▼
┌──────────────────┐
│ Shared Storage   │ 6. /images/machines/{tag}/
│ (PVC)            │    - bzImage
└──────────────────┘    - initrd
     │
     │ 7. Update build: success
     │    Update machine: ready
     ▼
┌──────────────────┐
│ Enrollment       │
│ Server           │
└──────────────────┘
```

### Machine Boot with Custom Image

```
┌──────────┐
│ Machine  │
└────┬─────┘
     │ 1. PXE Boot (reboot)
     ▼
┌──────────┐
│   TFTP   │ 2. Serve snp.efi
└────┬─────┘
     │
     ▼
┌──────────┐
│ snp.efi  │ 3. Chain to iPXE server
└────┬─────┘      GET /nixos/machines/{tag}.ipxe
     │
     ▼
┌──────────────┐
│ iPXE Server  │ 4. Machine found, status: ready
└────┬─────────┘    Check /images/machines/{tag}/bzImage exists
     │              Return custom.ipxe
     ▼
┌─────────────────┐
│ Custom NixOS    │ 5. Boot custom image
│ Image           │
└─────────────────┘
```

## Design Decisions

### Why Go?

- Fast compilation and deployment
- Excellent HTTP server capabilities
- Good Kubernetes client libraries
- Single binary deployment
- Strong concurrency support

### Why SQLite/PostgreSQL?

- SQLite: Simple deployment, no external dependencies
- PostgreSQL: Production-grade, better concurrency
- Both supported via database interface

### Why NixOS for Images?

- Declarative configuration
- Reproducible builds
- Netboot support built-in
- Excellent hardware support
- Same ecosystem for registration and custom images

### Why Kubernetes?

- Standardized deployment
- Easy scaling
- Built-in service discovery
- Persistent volume management
- Integration with existing infrastructure

### Why Shared Storage?

Builder and iPXE server need access to built images. Options considered:

1. **Shared PVC** (chosen)
   - Simple, standard Kubernetes pattern
   - Works with NFS or similar
   - No custom sync needed

2. Object storage (S3)
   - More complex
   - Additional dependency
   - Better for multi-cluster

3. Container registry
   - Images not containers
   - Unnecessary overhead

### Image Builder Polling vs Push

The builder polls for pending builds rather than receiving push notifications:

**Pros**:
- Simpler architecture
- No webhook infrastructure needed
- Resilient to builder restarts
- Natural rate limiting

**Cons**:
- Slight delay (10 seconds)
- Database polling overhead

For low-volume use case, polling is simpler and adequate.

## Security Considerations

### Current Implementation

- No authentication on API (development)
- Root access in registration image (necessary)
- Privileged builder container (needed for Nix)
- Default root password in registration image

### Production Recommendations

1. **Add API Authentication**
   - JWT tokens
   - API keys
   - OAuth2/OIDC

2. **Secure Registration**
   - Remove default password
   - Use SSH keys only
   - Network isolation

3. **Builder Security**
   - Separate builder node
   - Network policies
   - Resource limits

4. **Storage Security**
   - Encrypt PVCs
   - Access controls
   - Image signing

## Scalability

### Current Limits

- Single builder instance
- SQLite limitations
- Shared storage I/O

### Scaling Strategies

1. **Multiple Builders**
   - Add database locking
   - Use PostgreSQL
   - Job queue system

2. **Distributed Storage**
   - Object storage
   - CDN for images
   - Regional deployments

3. **Caching**
   - Cache built images
   - Cache hardware data
   - CDN for static assets

## Future Enhancements

1. **Authentication & Authorization**
   - User management
   - Role-based access
   - Audit logging

2. **Advanced Features**
   - Machine grouping
   - Bulk operations
   - Configuration templates
   - IPMI integration

3. **Monitoring & Observability**
   - Prometheus metrics
   - Grafana dashboards
   - Distributed tracing
   - Log aggregation

4. **Integration**
   - Terraform provider
   - Ansible module
   - GitOps workflow
   - Webhook notifications

## References

- [iPXE Documentation](https://ipxe.org/docs)
- [NixOS Manual](https://nixos.org/manual/nixos/stable/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Canonical MaaS](https://maas.io/) (inspiration)

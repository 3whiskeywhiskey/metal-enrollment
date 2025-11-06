# Metal Enrollment Terraform Provider

This directory contains a Terraform provider for managing Metal Enrollment machines.

## Building the Provider

```bash
cd integrations/terraform
go build -o terraform-provider-metal-enrollment
```

## Installation

1. Build the provider
2. Copy the binary to your Terraform plugins directory:

```bash
# Linux/macOS
mkdir -p ~/.terraform.d/plugins/local/metal-enrollment/metal-enrollment/1.0.0/$(go env GOOS)_$(go env GOARCH)/
cp terraform-provider-metal-enrollment ~/.terraform.d/plugins/local/metal-enrollment/metal-enrollment/1.0.0/$(go env GOOS)_$(go env GOARCH)/
```

## Usage

```hcl
terraform {
  required_providers {
    metal-enrollment = {
      source  = "local/metal-enrollment/metal-enrollment"
      version = "1.0.0"
    }
  }
}

provider "metal-enrollment" {
  api_url = "http://localhost:8080"
  token   = "your-jwt-token"  # Optional
}

# Data source to read a machine
data "metal-enrollment_machine" "server01" {
  service_tag = "ABC123"
}

# Resource to manage a machine
resource "metal-enrollment_machine" "web_server" {
  service_tag  = data.metal-enrollment_machine.server01.service_tag
  hostname     = "web-server-01"
  description  = "Production web server"
  nixos_config = file("${path.module}/nixos-config.nix")

  bmc {
    ip_address = "10.0.0.100"
    username   = "admin"
    password   = var.bmc_password
    type       = "IPMI"
    port       = 623
    enabled    = true
  }
}

# Resource to manage a group
resource "metal-enrollment_group" "web_servers" {
  name        = "web-servers"
  description = "Production web servers"
  tags        = ["production", "web"]
}

# Add machine to group
resource "metal-enrollment_group_membership" "web_server_membership" {
  group_id   = metal-enrollment_group.web_servers.id
  machine_id = metal-enrollment_machine.web_server.id
}

# Data source to list all machines
data "metal-enrollment_machines" "all" {}

# Output all machine hostnames
output "all_hostnames" {
  value = [for m in data.metal-enrollment_machines.all.machines : m.hostname]
}

# Power control
resource "metal-enrollment_power_operation" "reboot_web" {
  machine_id = metal-enrollment_machine.web_server.id
  operation  = "reset"
}
```

## Resources

- `metal-enrollment_machine` - Manage machine configuration
- `metal-enrollment_group` - Manage machine groups
- `metal-enrollment_group_membership` - Manage group memberships
- `metal-enrollment_power_operation` - Execute power operations

## Data Sources

- `metal-enrollment_machine` - Read machine information
- `metal-enrollment_machines` - List all machines
- `metal-enrollment_group` - Read group information
- `metal-enrollment_groups` - List all groups

## Configuration

The provider supports the following configuration options:

- `api_url` - (Required) URL of the Metal Enrollment API
- `token` - (Optional) JWT authentication token
- `insecure` - (Optional) Skip TLS verification (default: false)

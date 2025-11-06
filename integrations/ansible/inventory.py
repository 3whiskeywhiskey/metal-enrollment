#!/usr/bin/env python3
"""
Ansible dynamic inventory script for Metal Enrollment system.

This script queries the Metal Enrollment API and generates an Ansible inventory
in JSON format, grouping machines by their configured groups and status.

Usage:
    ./inventory.py --list
    ./inventory.py --host <hostname>

Configuration:
    Set environment variables:
    - METAL_ENROLLMENT_URL: URL to the Metal Enrollment API (default: http://localhost:8080)
    - METAL_ENROLLMENT_TOKEN: JWT token for authentication (optional)
"""

import argparse
import json
import os
import sys
from urllib.request import Request, urlopen
from urllib.error import URLError, HTTPError


class MetalInventory:
    def __init__(self):
        self.api_url = os.environ.get('METAL_ENROLLMENT_URL', 'http://localhost:8080')
        self.token = os.environ.get('METAL_ENROLLMENT_TOKEN', '')
        self.inventory = {
            '_meta': {
                'hostvars': {}
            }
        }

    def _make_request(self, endpoint):
        """Make HTTP request to the API"""
        url = f"{self.api_url}/api/v1/{endpoint}"
        headers = {}

        if self.token:
            headers['Authorization'] = f'Bearer {self.token}'

        req = Request(url, headers=headers)

        try:
            response = urlopen(req)
            return json.loads(response.read().decode('utf-8'))
        except HTTPError as e:
            print(f"HTTP Error: {e.code} - {e.reason}", file=sys.stderr)
            sys.exit(1)
        except URLError as e:
            print(f"URL Error: {e.reason}", file=sys.stderr)
            sys.exit(1)
        except Exception as e:
            print(f"Error: {str(e)}", file=sys.stderr)
            sys.exit(1)

    def get_machines(self):
        """Get all machines from the API"""
        return self._make_request('machines')

    def get_groups(self):
        """Get all groups from the API"""
        return self._make_request('groups')

    def get_group_machines(self, group_id):
        """Get machines in a specific group"""
        return self._make_request(f'groups/{group_id}/machines')

    def build_inventory(self):
        """Build the complete inventory"""
        # Get all machines
        machines = self.get_machines()

        # Initialize status-based groups
        status_groups = {}

        # Process each machine
        for machine in machines:
            hostname = machine.get('hostname', machine['service_tag'])

            # Build hostvars
            hostvars = {
                'ansible_host': machine.get('hostname', machine['service_tag']),
                'machine_id': machine['id'],
                'service_tag': machine['service_tag'],
                'mac_address': machine['mac_address'],
                'status': machine['status'],
                'description': machine.get('description', ''),
                'hardware': machine.get('hardware', {}),
            }

            # Add BMC info if available
            if machine.get('bmc_info'):
                bmc = machine['bmc_info']
                if bmc.get('enabled'):
                    hostvars['bmc_address'] = bmc.get('ip_address', '')
                    hostvars['bmc_username'] = bmc.get('username', '')
                    # Note: password is not exposed for security

            self.inventory['_meta']['hostvars'][hostname] = hostvars

            # Add to status-based group
            status = machine['status']
            if status not in status_groups:
                status_groups[status] = []
            status_groups[status].append(hostname)

        # Add status groups to inventory
        for status, hosts in status_groups.items():
            group_name = f"status_{status}"
            self.inventory[group_name] = {
                'hosts': hosts
            }

        # Get custom groups
        try:
            groups = self.get_groups()
            for group in groups:
                group_machines = self.get_group_machines(group['id'])
                group_name = group['name'].replace(' ', '_').lower()

                hosts = []
                for machine in group_machines:
                    hostname = machine.get('hostname', machine['service_tag'])
                    hosts.append(hostname)

                self.inventory[group_name] = {
                    'hosts': hosts,
                    'vars': {
                        'group_description': group.get('description', ''),
                        'group_tags': group.get('tags', [])
                    }
                }
        except Exception as e:
            print(f"Warning: Failed to get custom groups: {e}", file=sys.stderr)

        # Add meta groups
        self.inventory['all'] = {
            'children': list(status_groups.keys())
        }

        return self.inventory

    def get_host(self, hostname):
        """Get hostvars for a specific host"""
        inventory = self.build_inventory()
        return inventory['_meta']['hostvars'].get(hostname, {})


def main():
    parser = argparse.ArgumentParser(
        description='Ansible dynamic inventory for Metal Enrollment'
    )
    parser.add_argument('--list', action='store_true',
                        help='List all hosts')
    parser.add_argument('--host', help='Get variables for a specific host')

    args = parser.parse_args()

    inventory = MetalInventory()

    if args.list:
        result = inventory.build_inventory()
        print(json.dumps(result, indent=2))
    elif args.host:
        result = inventory.get_host(args.host)
        print(json.dumps(result, indent=2))
    else:
        parser.print_help()
        sys.exit(1)


if __name__ == '__main__':
    main()

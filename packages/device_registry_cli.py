#!/usr/bin/env python3
"""Command-line interface for Device Registry System."""

import argparse
import json
import sys
from pathlib import Path

# Add source paths  
sys.path.insert(0, str(Path(__file__).parent / "bifrost-core" / "src"))

from bifrost_core.device_registry import DeviceRegistry


def main():
    """Main CLI function."""
    parser = argparse.ArgumentParser(
        description="Device Registry CLI",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Show registry statistics
  python device_registry_cli.py --stats
  
  # List all devices
  python device_registry_cli.py --list
  
  # Find devices by protocol
  python device_registry_cli.py --protocol modbus_tcp
  
  # Import registry from file
  python device_registry_cli.py --import sample_registry.yaml
  
  # Export registry to file
  python device_registry_cli.py --export output.json --format json
  
  # Generate compatibility report
  python device_registry_cli.py --compatibility modbus_tcp
        """
    )
    
    parser.add_argument(
        "--stats", action="store_true",
        help="Show registry statistics"
    )
    
    parser.add_argument(
        "--list", choices=["all", "virtual", "real"], default=None,
        help="List devices (all, virtual, or real)"
    )
    
    parser.add_argument(
        "--protocol", type=str,
        help="Find devices supporting specific protocol"
    )
    
    parser.add_argument(
        "--compatibility", type=str,
        help="Generate compatibility report for protocol"
    )
    
    parser.add_argument(
        "--import", dest="import_file", type=str,
        help="Import registry from file"
    )
    
    parser.add_argument(
        "--export", type=str,
        help="Export registry to file"
    )
    
    parser.add_argument(
        "--format", choices=["json", "yaml"], default="json",
        help="Format for import/export (default: json)"
    )
    
    parser.add_argument(
        "--registry", type=str, default=None,
        help="Load initial registry from file"
    )
    
    parser.add_argument(
        "--verbose", "-v", action="store_true",
        help="Verbose output"
    )
    
    args = parser.parse_args()
    
    # Create registry
    registry = DeviceRegistry()
    
    # Load initial registry if specified
    if args.registry:
        try:
            if args.registry.endswith('.yaml') or args.registry.endswith('.yml'):
                registry.import_from_yaml(args.registry)
            else:
                registry.import_from_json(args.registry)
            if args.verbose:
                print(f"Loaded registry from {args.registry}")
        except Exception as e:
            print(f"Error loading registry: {e}", file=sys.stderr)
            return 1
    
    # Handle import
    if args.import_file:
        try:
            if args.import_file.endswith('.yaml') or args.import_file.endswith('.yml'):
                registry.import_from_yaml(args.import_file)
            else:
                registry.import_from_json(args.import_file)
            print(f"Imported registry from {args.import_file}")
        except Exception as e:
            print(f"Error importing registry: {e}", file=sys.stderr)
            return 1
    
    # Handle stats
    if args.stats:
        virtual_devices = registry.list_virtual_devices()
        real_devices = registry.list_real_devices()
        
        # Count protocols
        protocols = set()
        for device in virtual_devices:
            protocols.add(device.protocol)
        for device in real_devices:
            protocols.update(device.protocols.keys())
        
        # Count by protocol
        protocol_counts = {}
        for device in virtual_devices:
            protocol_counts[device.protocol] = protocol_counts.get(device.protocol, 0) + 1
        for device in real_devices:
            for protocol in device.protocols.keys():
                protocol_counts[protocol] = protocol_counts.get(protocol, 0) + 1
        
        print("Device Registry Statistics")
        print("=" * 30)
        print(f"Virtual devices: {len(virtual_devices)}")
        print(f"Real devices: {len(real_devices)}")
        print(f"Total devices: {len(virtual_devices) + len(real_devices)}")
        print(f"Supported protocols: {len(protocols)}")
        
        if protocol_counts:
            print("\nProtocol Distribution:")
            for protocol, count in sorted(protocol_counts.items()):
                print(f"  {protocol}: {count} devices")
    
    # Handle list
    if args.list:
        if args.list in ["all", "virtual"]:
            virtual_devices = registry.list_virtual_devices()
            if virtual_devices:
                print("Virtual Devices:")
                print("-" * 50)
                for device in virtual_devices:
                    print(f"  ID: {device.id}")
                    print(f"  Type: {device.type}")
                    print(f"  Protocol: {device.protocol}")
                    if device.configuration and hasattr(device.configuration, 'performance'):
                        perf = device.configuration.performance
                        if perf and perf.max_throughput:
                            print(f"  Max Throughput: {perf.max_throughput}")
                    if device.test_scenarios:
                        print(f"  Test Scenarios: {', '.join(device.test_scenarios)}")
                    print()
            else:
                print("No virtual devices found.")
        
        if args.list in ["all", "real"]:
            real_devices = registry.list_real_devices()
            if real_devices:
                print("Real Devices:")
                print("-" * 50)
                for device in real_devices:
                    print(f"  ID: {device.id}")
                    print(f"  Manufacturer: {device.manufacturer}")
                    print(f"  Model: {device.model}")
                    if device.firmware:
                        print(f"  Firmware: {device.firmware}")
                    print(f"  Protocols: {', '.join(device.protocols.keys())}")
                    if device.test_notes:
                        print(f"  Notes: {device.test_notes}")
                    print()
            else:
                print("No real devices found.")
    
    # Handle protocol search
    if args.protocol:
        devices = registry.find_devices_by_protocol(args.protocol)
        virtual_count = len(devices['virtual'])
        real_count = len(devices['real'])
        
        print(f"Devices supporting {args.protocol}:")
        print("=" * 40)
        print(f"Virtual devices: {virtual_count}")
        print(f"Real devices: {real_count}")
        print(f"Total: {virtual_count + real_count}")
        
        if args.verbose:
            if devices['virtual']:
                print("\nVirtual Devices:")
                for device in devices['virtual']:
                    print(f"  - {device.id} ({device.type})")
            
            if devices['real']:
                print("\nReal Devices:")
                for device in devices['real']:
                    print(f"  - {device.id} ({device.manufacturer} {device.model})")
    
    # Handle compatibility report
    if args.compatibility:
        report = registry.get_compatibility_report(args.compatibility)
        
        print(f"Compatibility Report for {args.compatibility}")
        print("=" * 50)
        print(f"Total tested devices: {report['total_tested']}")
        print(f"Virtual devices: {report['virtual_devices_count']}")
        print(f"Real devices: {report['real_devices_count']}")
        
        if args.verbose and report['real_devices']:
            print("\nReal Device Details:")
            for device in report['real_devices']:
                print(f"  - {device['id']} ({device['manufacturer']})")
                print(f"    Status: {device['status']}")
                if device['limitations']:
                    print(f"    Limitations: {', '.join(device['limitations'])}")
    
    # Handle export
    if args.export:
        try:
            if args.format == "yaml":
                data = registry.export_to_yaml(args.export)
            else:
                data = registry.export_to_json(args.export)
            print(f"Exported registry to {args.export} ({args.format} format)")
        except Exception as e:
            print(f"Error exporting registry: {e}", file=sys.stderr)
            return 1
    
    # If no specific action, show help
    if not any([args.stats, args.list, args.protocol, args.compatibility, 
                args.import_file, args.export]):
        parser.print_help()
    
    return 0


if __name__ == "__main__":
    sys.exit(main())
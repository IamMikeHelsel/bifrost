#!/usr/bin/env python3
"""Demo script for Device Registry API."""

import sys
from pathlib import Path

# Add the source paths
sys.path.insert(0, str(Path(__file__).parent / "bifrost-core" / "src"))
sys.path.insert(0, str(Path(__file__).parent / "bifrost" / "src"))

from bifrost_core.device_registry import (
    DeviceRegistry, VirtualDevice, RealDevice, 
    VirtualDeviceConfiguration, PerformanceMetrics, ProtocolSupport
)

# Import the API class directly from the file
import importlib.util
api_spec = importlib.util.spec_from_file_location(
    "device_registry_api", 
    str(Path(__file__).parent / "bifrost" / "src" / "bifrost" / "device_registry_api.py")
)
api_module = importlib.util.module_from_spec(api_spec)
api_spec.loader.exec_module(api_module)
DeviceRegistryAPI = api_module.DeviceRegistryAPI

def main():
    """Demonstrate Device Registry API functionality."""
    print("=== Device Registry API Demo ===\n")
    
    # Create API instance
    api = DeviceRegistryAPI()
    
    # Show initial stats
    print("1. Initial Registry Stats:")
    stats = api.get_registry_stats()
    print(f"   Total devices: {stats['total_devices']}")
    print(f"   Supported protocols: {stats['supported_protocols']}")
    print()
    
    # Create virtual device
    print("2. Creating Virtual Device...")
    virtual_device_data = {
        "id": "modbus_tcp_sim_v1.0",
        "type": "simulator",
        "protocol": "modbus_tcp",
        "configuration": {
            "registers": 1000,
            "functions": [1, 2, 3, 4, 5, 6, 15, 16],
            "performance": {
                "max_throughput": "1500 regs/sec",
                "latency": "0.5ms"
            }
        },
        "test_scenarios": ["factory_floor_modbus", "performance_benchmark"]
    }
    
    virtual_device = api.create_virtual_device(virtual_device_data)
    print(f"   Created virtual device: {virtual_device['id']}")
    print(f"   Protocol: {virtual_device['protocol']}")
    print()
    
    # Create real device
    print("3. Creating Real Device...")
    real_device_data = {
        "id": "schneider_m221",
        "manufacturer": "Schneider Electric",
        "model": "Modicon M221",
        "firmware": "1.7.2.0",
        "protocols": {
            "modbus_tcp": {
                "status": "validated",
                "performance": {
                    "throughput": "800 regs/sec",
                    "latency": "2ms"
                },
                "limitations": ["No holding register write"]
            }
        },
        "test_notes": "Requires specific timeout settings"
    }
    
    real_device = api.create_real_device(real_device_data)
    print(f"   Created real device: {real_device['id']}")
    print(f"   Manufacturer: {real_device['manufacturer']}")
    print()
    
    # Show updated stats
    print("4. Updated Registry Stats:")
    stats = api.get_registry_stats()
    print(f"   Total devices: {stats['total_devices']}")
    print(f"   Virtual devices: {stats['total_virtual_devices']}")
    print(f"   Real devices: {stats['total_real_devices']}")
    print(f"   Supported protocols: {stats['supported_protocols']}")
    print(f"   Protocol distribution: {stats['protocol_distribution']}")
    print()
    
    # Find devices by protocol
    print("5. Finding Devices by Protocol (modbus_tcp):")
    devices = api.find_devices_by_protocol("modbus_tcp")
    print(f"   Virtual devices found: {len(devices['virtual'])}")
    print(f"   Real devices found: {len(devices['real'])}")
    print()
    
    # Generate compatibility report
    print("6. Compatibility Report for modbus_tcp:")
    report = api.get_compatibility_report("modbus_tcp")
    print(f"   Protocol: {report['protocol']}")
    print(f"   Total tested devices: {report['total_tested']}")
    print(f"   Virtual devices tested: {report['virtual_devices_count']}")
    print(f"   Real devices tested: {report['real_devices_count']}")
    print()
    
    # Export data
    print("7. Exporting Registry Data:")
    json_data = api.export_registry("json")
    print(f"   JSON export size: {len(json_data)} characters")
    
    yaml_data = api.export_registry("yaml")
    print(f"   YAML export size: {len(yaml_data)} characters")
    print()
    
    # Test import (create new API and import data)
    print("8. Testing Import Functionality:")
    new_api = DeviceRegistryAPI()
    new_api.import_registry(json_data, "json")
    
    new_stats = new_api.get_registry_stats()
    print(f"   Imported devices: {new_stats['total_devices']}")
    print(f"   Data integrity check: {'PASS' if new_stats['total_devices'] == stats['total_devices'] else 'FAIL'}")
    print()
    
    # Show supported protocols
    print("9. Supported Protocols:")
    protocols = api.get_supported_protocols()
    for protocol in protocols:
        print(f"   - {protocol}")
    print()
    
    print("=== Demo Complete ===")

if __name__ == "__main__":
    main()
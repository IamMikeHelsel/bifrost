#!/usr/bin/env python3
"""Demo script for Device Registry Test Integration."""

import sys
from datetime import datetime, timedelta
from pathlib import Path

# Add the source paths
sys.path.insert(0, str(Path(__file__).parent / "bifrost-core" / "src"))

from bifrost_core.device_registry import (
    DeviceRegistry, RealDevice, VirtualDevice,
    ProtocolSupport, PerformanceMetrics, VirtualDeviceConfiguration
)
from bifrost_core.test_integration import (
    DeviceTestTracker, TestStatus
)

def main():
    """Demonstrate Device Registry Test Integration."""
    print("=== Device Registry Test Integration Demo ===\n")
    
    # Create test tracker (includes device registry)
    tracker = DeviceTestTracker()
    
    # Register some devices
    print("1. Registering Test Devices...")
    
    # Virtual device
    virtual_device = VirtualDevice(
        id="modbus_sim_1",
        type="simulator",
        protocol="modbus_tcp",
        configuration=VirtualDeviceConfiguration(
            registers=1000,
            performance=PerformanceMetrics()
        )
    )
    tracker.device_registry.register_virtual_device(virtual_device)
    
    # Real device
    real_device = RealDevice(
        id="test_plc_1",
        manufacturer="Test Vendor",
        model="Test PLC Model",
        firmware="1.0.0",
        protocols={
            "modbus_tcp": ProtocolSupport(
                status="testing",
                performance=PerformanceMetrics(throughput="500 regs/sec")
            ),
            "ethernet_ip": ProtocolSupport(
                status="testing"
            )
        }
    )
    tracker.device_registry.register_real_device(real_device)
    
    print(f"   Registered virtual device: {virtual_device.id}")
    print(f"   Registered real device: {real_device.id}")
    print()
    
    # Start test session
    print("2. Starting Test Session...")
    session = tracker.start_test_session("integration_test_session", {
        "test_suite": "device_compatibility",
        "environment": "test_lab"
    })
    print(f"   Started session: {session.session_id}")
    print()
    
    # Simulate test execution
    print("3. Executing Tests...")
    base_time = datetime.now()
    
    test_scenarios = [
        # Virtual device tests (should update performance metrics)
        ("modbus_sim_1", "modbus_tcp", "read_coils_test", TestStatus.PASSED, {"throughput": 1200, "latency": 0.8}),
        ("modbus_sim_1", "modbus_tcp", "read_registers_test", TestStatus.PASSED, {"throughput": 1100, "latency": 0.9}),
        ("modbus_sim_1", "modbus_tcp", "write_registers_test", TestStatus.PASSED, {"throughput": 1000, "latency": 1.0}),
        
        # Real device tests (should update protocol status)
        ("test_plc_1", "modbus_tcp", "basic_connectivity_test", TestStatus.PASSED, None),
        ("test_plc_1", "modbus_tcp", "bulk_read_test", TestStatus.PASSED, None),
        ("test_plc_1", "modbus_tcp", "write_operations_test", TestStatus.PASSED, None),
        ("test_plc_1", "modbus_tcp", "stress_test", TestStatus.FAILED, None, "Connection timeout after 30 seconds"),
        ("test_plc_1", "ethernet_ip", "device_discovery_test", TestStatus.PASSED, None),
        ("test_plc_1", "ethernet_ip", "tag_read_test", TestStatus.FAILED, None, "Unsupported data type"),
    ]
    
    for i, scenario in enumerate(test_scenarios):
        device_id, protocol, test_name, status = scenario[:4]
        performance_metrics = scenario[4] if len(scenario) > 4 else None
        error_message = scenario[5] if len(scenario) > 5 else None
        
        start_time = base_time + timedelta(minutes=i)
        end_time = start_time + timedelta(seconds=30)
        
        result = tracker.record_test_result(
            session_id=session.session_id,
            test_name=test_name,
            device_id=device_id,
            protocol=protocol,
            status=status,
            start_time=start_time,
            end_time=end_time,
            performance_metrics=performance_metrics,
            error_message=error_message
        )
        
        status_icon = "✓" if status == TestStatus.PASSED else "✗"
        print(f"   {status_icon} {test_name} on {device_id} ({protocol}) - {status.value}")
    
    print()
    
    # End test session
    print("4. Ending Test Session...")
    tracker.end_test_session(session.session_id)
    print(f"   Session {session.session_id} completed")
    print()
    
    # Update devices from test results
    print("5. Updating Device Registry from Test Results...")
    
    # Update virtual device (should get performance metrics)
    virtual_updated = tracker.update_device_from_test_results("modbus_sim_1")
    print(f"   Virtual device updated: {virtual_updated}")
    
    # Update real device (should get status updates)
    real_updated = tracker.update_device_from_test_results("test_plc_1")
    print(f"   Real device updated: {real_updated}")
    print()
    
    # Show updated device information
    print("6. Updated Device Information...")
    
    updated_virtual = tracker.device_registry.get_virtual_device("modbus_sim_1")
    if updated_virtual and updated_virtual.configuration and updated_virtual.configuration.performance:
        perf = updated_virtual.configuration.performance
        print(f"   Virtual Device Performance:")
        print(f"     Throughput: {perf.throughput}")
        print(f"     Latency: {perf.latency}")
    
    updated_real = tracker.device_registry.get_real_device("test_plc_1")
    if updated_real:
        print(f"   Real Device Status:")
        for protocol, support in updated_real.protocols.items():
            print(f"     {protocol}: {support.status}")
        if updated_real.test_notes:
            print(f"     Notes: {updated_real.test_notes}")
    print()
    
    # Generate test report
    print("7. Generating Test Reports...")
    
    overall_report = tracker.generate_test_report()
    print(f"   Overall Test Results:")
    print(f"     Total tests: {overall_report['total_tests']}")
    print(f"     Passed: {overall_report['passed_tests']}")
    print(f"     Failed: {overall_report['failed_tests']}")
    print(f"     Success rate: {overall_report['success_rate']:.1f}%")
    print(f"     Devices tested: {overall_report['tested_devices']}")
    print(f"     Protocols tested: {overall_report['tested_protocols']}")
    
    # Protocol-specific reports
    modbus_report = tracker.generate_test_report("modbus_tcp")
    ethernet_ip_report = tracker.generate_test_report("ethernet_ip")
    
    print(f"   Modbus TCP: {modbus_report['passed_tests']}/{modbus_report['total_tests']} tests passed ({modbus_report['success_rate']:.1f}%)")
    print(f"   EtherNet/IP: {ethernet_ip_report['passed_tests']}/{ethernet_ip_report['total_tests']} tests passed ({ethernet_ip_report['success_rate']:.1f}%)")
    print()
    
    # Device test status
    print("8. Device Test Status...")
    
    virtual_status = tracker.get_device_test_status("modbus_sim_1")
    real_status = tracker.get_device_test_status("test_plc_1")
    
    print(f"   Virtual Device (modbus_sim_1):")
    print(f"     Status: {virtual_status['status']}")
    print(f"     Success rate: {virtual_status['success_rate']:.1f}%")
    
    print(f"   Real Device (test_plc_1):")
    print(f"     Status: {real_status['status']}")
    print(f"     Success rate: {real_status['success_rate']:.1f}%")
    print(f"     Protocol status:")
    for protocol, pstatus in real_status['protocol_status'].items():
        print(f"       {protocol}: {pstatus['latest_result']} ({pstatus['success_rate']:.1f}%)")
    print()
    
    # Export test results
    print("9. Exporting Test Results...")
    
    json_export = tracker.export_test_results("json")
    print(f"   JSON export size: {len(json_export)} characters")
    
    # Show registry compatibility report
    print("10. Final Device Registry State...")
    
    registry_stats = {
        'virtual_devices': len(tracker.device_registry.list_virtual_devices()),
        'real_devices': len(tracker.device_registry.list_real_devices())
    }
    
    print(f"    Devices in registry: {registry_stats['virtual_devices']} virtual, {registry_stats['real_devices']} real")
    
    modbus_compat = tracker.device_registry.get_compatibility_report("modbus_tcp")
    print(f"    Modbus TCP compatibility: {modbus_compat['total_tested']} devices tested")
    
    ethernet_ip_compat = tracker.device_registry.get_compatibility_report("ethernet_ip")
    print(f"    EtherNet/IP compatibility: {ethernet_ip_compat['total_tested']} devices tested")
    
    print("\n=== Demo Complete ===")
    print("\nThe Device Registry Test Integration provides:")
    print("- Automated test result tracking")
    print("- Device status updates based on test outcomes")
    print("- Performance metric collection for virtual devices")
    print("- Protocol compatibility validation")
    print("- Comprehensive test reporting")
    print("- Integration with existing device registry")

if __name__ == "__main__":
    main()
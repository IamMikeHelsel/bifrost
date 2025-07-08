"""Tests for the Device Registry System."""

import json
import tempfile
from datetime import datetime
from pathlib import Path

import pytest

try:
    from bifrost_core.device_registry import (
        DeviceRegistry,
        PerformanceMetrics,
        ProtocolSupport,
        RealDevice,
        VirtualDevice,
        VirtualDeviceConfiguration,
    )
    from pydantic import ValidationError
except ImportError:
    # Handle case where pydantic is not available
    pytest.skip("Pydantic not available", allow_module_level=True)


class TestPerformanceMetrics:
    """Tests for PerformanceMetrics model."""

    def test_create_basic_metrics(self):
        """Test creating basic performance metrics."""
        metrics = PerformanceMetrics(
            throughput="1500 regs/sec",
            latency="0.5ms"
        )
        assert metrics.throughput == "1500 regs/sec"
        assert metrics.latency == "0.5ms"
        assert metrics.max_throughput is None

    def test_create_empty_metrics(self):
        """Test creating empty performance metrics."""
        metrics = PerformanceMetrics()
        assert metrics.throughput is None
        assert metrics.latency is None
        assert metrics.max_throughput is None

    def test_extra_fields_allowed(self):
        """Test that extra fields are allowed in metrics."""
        metrics = PerformanceMetrics(
            throughput="1000 reqs/sec",
            custom_metric="custom_value"
        )
        assert metrics.throughput == "1000 reqs/sec"
        # Extra fields should be accessible via __dict__
        assert hasattr(metrics, 'custom_metric')


class TestVirtualDevice:
    """Tests for VirtualDevice model."""

    def test_create_basic_virtual_device(self):
        """Test creating a basic virtual device."""
        device = VirtualDevice(
            id="modbus_tcp_sim_v1.0",
            type="simulator",
            protocol="modbus_tcp"
        )
        assert device.id == "modbus_tcp_sim_v1.0"
        assert device.type == "simulator"
        assert device.protocol == "modbus_tcp"
        assert device.configuration is None
        assert device.test_scenarios is None

    def test_create_virtual_device_with_configuration(self):
        """Test creating virtual device with configuration."""
        config = VirtualDeviceConfiguration(
            registers=1000,
            functions=[1, 2, 3, 4, 5, 6, 15, 16],
            performance=PerformanceMetrics(
                max_throughput="1500 regs/sec",
                latency="0.5ms"
            )
        )
        
        device = VirtualDevice(
            id="modbus_tcp_sim_v1.0",
            type="simulator", 
            protocol="modbus_tcp",
            configuration=config,
            test_scenarios=["factory_floor_modbus", "performance_benchmark"]
        )
        
        assert device.configuration is not None
        assert device.configuration.registers == 1000
        assert device.configuration.functions == [1, 2, 3, 4, 5, 6, 15, 16]
        assert device.test_scenarios == ["factory_floor_modbus", "performance_benchmark"]

    def test_protocol_validation(self):
        """Test protocol validation."""
        # Valid protocol should work
        device = VirtualDevice(
            id="test_device",
            type="simulator",
            protocol="MODBUS_TCP"  # Should be converted to lowercase
        )
        assert device.protocol == "modbus_tcp"

    def test_required_fields(self):
        """Test that required fields are enforced."""
        with pytest.raises(ValidationError):
            VirtualDevice()  # Missing required fields


class TestRealDevice:
    """Tests for RealDevice model."""

    def test_create_basic_real_device(self):
        """Test creating a basic real device."""
        device = RealDevice(
            id="schneider_m221",
            manufacturer="Schneider Electric",
            model="Modicon M221"
        )
        assert device.id == "schneider_m221"
        assert device.manufacturer == "Schneider Electric"
        assert device.model == "Modicon M221"
        assert device.firmware is None
        assert device.protocols == {}

    def test_create_real_device_with_protocols(self):
        """Test creating real device with protocol support."""
        protocol_support = ProtocolSupport(
            status="validated",
            performance=PerformanceMetrics(
                throughput="800 regs/sec",
                latency="2ms"
            ),
            limitations=["No holding register write"]
        )
        
        device = RealDevice(
            id="schneider_m221",
            manufacturer="Schneider Electric", 
            model="Modicon M221",
            firmware="1.7.2.0",
            protocols={"modbus_tcp": protocol_support},
            test_notes="Requires specific timeout settings"
        )
        
        assert device.firmware == "1.7.2.0"
        assert "modbus_tcp" in device.protocols
        assert device.protocols["modbus_tcp"].status == "validated"
        assert device.test_notes == "Requires specific timeout settings"

    def test_manufacturer_validation(self):
        """Test manufacturer validation."""
        # Empty manufacturer should fail
        with pytest.raises(ValidationError):
            RealDevice(
                id="test",
                manufacturer="",
                model="Test Model"
            )


class TestProtocolSupport:
    """Tests for ProtocolSupport model."""

    def test_create_protocol_support(self):
        """Test creating protocol support information."""
        support = ProtocolSupport(
            status="validated",
            performance=PerformanceMetrics(throughput="1000 reqs/sec"),
            limitations=["Limited to 100 connections"]
        )
        assert support.status == "validated"
        assert support.performance.throughput == "1000 reqs/sec"
        assert support.limitations == ["Limited to 100 connections"]

    def test_status_validation(self):
        """Test status validation."""
        # Valid status should work
        support = ProtocolSupport(status="VALIDATED")  # Should be converted to lowercase
        assert support.status == "validated"
        
        # Invalid status should fail
        with pytest.raises(ValidationError):
            ProtocolSupport(status="invalid_status")


class TestDeviceRegistry:
    """Tests for DeviceRegistry class."""

    @pytest.fixture
    def registry(self) -> DeviceRegistry:
        """Create a fresh device registry for testing."""
        return DeviceRegistry()

    @pytest.fixture
    def sample_virtual_device(self) -> VirtualDevice:
        """Create a sample virtual device."""
        return VirtualDevice(
            id="modbus_sim_1",
            type="simulator",
            protocol="modbus_tcp",
            configuration=VirtualDeviceConfiguration(
                registers=1000,
                functions=[1, 2, 3, 4, 5, 6]
            )
        )

    @pytest.fixture  
    def sample_real_device(self) -> RealDevice:
        """Create a sample real device."""
        return RealDevice(
            id="schneider_m221",
            manufacturer="Schneider Electric",
            model="Modicon M221",
            firmware="1.7.2.0",
            protocols={
                "modbus_tcp": ProtocolSupport(
                    status="validated",
                    performance=PerformanceMetrics(throughput="800 regs/sec")
                )
            }
        )

    def test_register_and_get_virtual_device(self, registry: DeviceRegistry, sample_virtual_device: VirtualDevice):
        """Test registering and retrieving virtual device."""
        registry.register_virtual_device(sample_virtual_device)
        
        retrieved = registry.get_virtual_device("modbus_sim_1")
        assert retrieved is not None
        assert retrieved.id == "modbus_sim_1"
        assert retrieved.type == "simulator"

    def test_register_and_get_real_device(self, registry: DeviceRegistry, sample_real_device: RealDevice):
        """Test registering and retrieving real device."""
        registry.register_real_device(sample_real_device)
        
        retrieved = registry.get_real_device("schneider_m221")
        assert retrieved is not None
        assert retrieved.id == "schneider_m221"
        assert retrieved.manufacturer == "Schneider Electric"

    def test_get_nonexistent_device(self, registry: DeviceRegistry):
        """Test getting non-existent device returns None."""
        assert registry.get_virtual_device("nonexistent") is None
        assert registry.get_real_device("nonexistent") is None

    def test_list_devices(self, registry: DeviceRegistry, sample_virtual_device: VirtualDevice, sample_real_device: RealDevice):
        """Test listing all devices."""
        registry.register_virtual_device(sample_virtual_device)
        registry.register_real_device(sample_real_device)
        
        virtual_devices = registry.list_virtual_devices()
        real_devices = registry.list_real_devices()
        
        assert len(virtual_devices) == 1
        assert len(real_devices) == 1
        assert virtual_devices[0].id == "modbus_sim_1"
        assert real_devices[0].id == "schneider_m221"

    def test_find_devices_by_protocol(self, registry: DeviceRegistry, sample_virtual_device: VirtualDevice, sample_real_device: RealDevice):
        """Test finding devices by protocol."""
        registry.register_virtual_device(sample_virtual_device)
        registry.register_real_device(sample_real_device)
        
        devices = registry.find_devices_by_protocol("modbus_tcp")
        
        assert len(devices['virtual']) == 1
        assert len(devices['real']) == 1
        assert devices['virtual'][0].id == "modbus_sim_1"
        assert devices['real'][0].id == "schneider_m221"

    def test_find_devices_by_nonexistent_protocol(self, registry: DeviceRegistry):
        """Test finding devices by non-existent protocol."""
        devices = registry.find_devices_by_protocol("nonexistent_protocol")
        
        assert len(devices['virtual']) == 0
        assert len(devices['real']) == 0

    def test_compatibility_report(self, registry: DeviceRegistry, sample_virtual_device: VirtualDevice, sample_real_device: RealDevice):
        """Test generating compatibility report."""
        registry.register_virtual_device(sample_virtual_device)
        registry.register_real_device(sample_real_device)
        
        report = registry.get_compatibility_report("modbus_tcp")
        
        assert report['protocol'] == "modbus_tcp"
        assert report['virtual_devices_count'] == 1
        assert report['real_devices_count'] == 1
        assert report['total_tested'] == 2
        assert len(report['virtual_devices']) == 1
        assert len(report['real_devices']) == 1

    def test_export_import_json(self, registry: DeviceRegistry, sample_virtual_device: VirtualDevice, sample_real_device: RealDevice):
        """Test JSON export and import functionality."""
        registry.register_virtual_device(sample_virtual_device)
        registry.register_real_device(sample_real_device)
        
        # Export to JSON string
        json_data = registry.export_to_json()
        assert json_data is not None
        assert "device_registry" in json_data
        
        # Test JSON is valid
        parsed = json.loads(json_data)
        assert "device_registry" in parsed
        assert "virtual_devices" in parsed["device_registry"]
        assert "real_devices" in parsed["device_registry"]
        
        # Import to new registry
        new_registry = DeviceRegistry()
        new_registry.import_from_dict(parsed)
        
        assert len(new_registry.list_virtual_devices()) == 1
        assert len(new_registry.list_real_devices()) == 1

    def test_export_import_json_file(self, registry: DeviceRegistry, sample_virtual_device: VirtualDevice):
        """Test JSON export and import with file."""
        registry.register_virtual_device(sample_virtual_device)
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
            temp_path = f.name
            
        try:
            # Export to file
            registry.export_to_json(temp_path)
            
            # Import from file
            new_registry = DeviceRegistry()
            new_registry.import_from_json(temp_path)
            
            assert len(new_registry.list_virtual_devices()) == 1
            device = new_registry.get_virtual_device("modbus_sim_1")
            assert device is not None
            assert device.type == "simulator"
            
        finally:
            Path(temp_path).unlink()

    def test_clear_registry(self, registry: DeviceRegistry, sample_virtual_device: VirtualDevice, sample_real_device: RealDevice):
        """Test clearing the registry."""
        registry.register_virtual_device(sample_virtual_device)
        registry.register_real_device(sample_real_device)
        
        assert len(registry.list_virtual_devices()) == 1
        assert len(registry.list_real_devices()) == 1
        
        registry.clear()
        
        assert len(registry.list_virtual_devices()) == 0
        assert len(registry.list_real_devices()) == 0

    def test_remove_devices(self, registry: DeviceRegistry, sample_virtual_device: VirtualDevice, sample_real_device: RealDevice):
        """Test removing individual devices."""
        registry.register_virtual_device(sample_virtual_device)
        registry.register_real_device(sample_real_device)
        
        # Remove virtual device
        result = registry.remove_virtual_device("modbus_sim_1")
        assert result is True
        assert len(registry.list_virtual_devices()) == 0
        
        # Try to remove non-existent device
        result = registry.remove_virtual_device("nonexistent")
        assert result is False
        
        # Remove real device  
        result = registry.remove_real_device("schneider_m221")
        assert result is True
        assert len(registry.list_real_devices()) == 0

    def test_export_import_yaml(self, registry: DeviceRegistry, sample_virtual_device: VirtualDevice):
        """Test YAML export and import functionality."""
        registry.register_virtual_device(sample_virtual_device)
        
        # Export to YAML string
        yaml_data = registry.export_to_yaml()
        assert yaml_data is not None
        assert "device_registry" in yaml_data
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            temp_path = f.name
            
        try:
            # Export to file
            registry.export_to_yaml(temp_path)
            
            # Import from file
            new_registry = DeviceRegistry()
            new_registry.import_from_yaml(temp_path)
            
            assert len(new_registry.list_virtual_devices()) == 1
            device = new_registry.get_virtual_device("modbus_sim_1")
            assert device is not None
            assert device.type == "simulator"
            
        finally:
            Path(temp_path).unlink()

    def test_duplicate_device_registration(self, registry: DeviceRegistry, sample_virtual_device: VirtualDevice):
        """Test registering devices with duplicate IDs."""
        registry.register_virtual_device(sample_virtual_device)
        
        # Register another device with same ID - should overwrite
        new_device = VirtualDevice(
            id="modbus_sim_1",
            type="different_simulator",
            protocol="opcua"
        )
        registry.register_virtual_device(new_device)
        
        devices = registry.list_virtual_devices()
        assert len(devices) == 1
        assert devices[0].type == "different_simulator"
        assert devices[0].protocol == "opcua"


class TestIntegrationWithExampleData:
    """Integration tests using the example data from the issue."""
    
    def test_example_yaml_schema_compatibility(self):
        """Test that our models can handle the example data from the issue."""
        # Create virtual device from example
        virtual_device = VirtualDevice(
            id="modbus_tcp_sim_v1.0",
            type="simulator",
            protocol="modbus_tcp",
            configuration=VirtualDeviceConfiguration(
                registers=1000,
                functions=[1, 2, 3, 4, 5, 6, 15, 16],
                performance=PerformanceMetrics(
                    max_throughput="1500 regs/sec",
                    latency="0.5ms"
                )
            ),
            test_scenarios=["factory_floor_modbus", "performance_benchmark"]
        )
        
        # Create real device from example
        real_device = RealDevice(
            id="schneider_m221",
            manufacturer="Schneider Electric",
            model="Modicon M221", 
            firmware="1.7.2.0",
            protocols={
                "modbus_tcp": ProtocolSupport(
                    status="validated",
                    performance=PerformanceMetrics(
                        throughput="800 regs/sec",
                        latency="2ms"
                    ),
                    limitations=["No holding register write"]
                )
            },
            test_date=datetime(2024, 11, 15),
            test_notes="Requires specific timeout settings"
        )
        
        # Test registry operations
        registry = DeviceRegistry()
        registry.register_virtual_device(virtual_device)
        registry.register_real_device(real_device)
        
        # Verify data integrity
        assert len(registry.list_virtual_devices()) == 1
        assert len(registry.list_real_devices()) == 1
        
        # Test compatibility report
        report = registry.get_compatibility_report("modbus_tcp")
        assert report['total_tested'] == 2
        assert report['virtual_devices_count'] == 1
        assert report['real_devices_count'] == 1
        
        # Test export functionality
        exported_data = registry.export_to_dict()
        assert "device_registry" in exported_data
        assert len(exported_data["device_registry"]["virtual_devices"]) == 1
        assert len(exported_data["device_registry"]["real_devices"]) == 1
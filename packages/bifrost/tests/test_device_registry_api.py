"""Tests for the Device Registry API."""

import json
from datetime import datetime

import pytest

try:
    from bifrost.device_registry_api import DeviceRegistryAPI
    from bifrost_core.device_registry import (
        DeviceRegistry,
        VirtualDevice,
        RealDevice,
        ProtocolSupport,
        PerformanceMetrics,
        VirtualDeviceConfiguration
    )
except ImportError as e:
    pytest.skip(f"Required modules not available: {e}", allow_module_level=True)


class TestDeviceRegistryAPI:
    """Tests for DeviceRegistryAPI class."""
    
    @pytest.fixture
    def api(self) -> DeviceRegistryAPI:
        """Create a fresh API instance for testing."""
        return DeviceRegistryAPI()
    
    @pytest.fixture
    def sample_virtual_device_data(self) -> dict:
        """Sample virtual device data."""
        return {
            "id": "modbus_sim_test",
            "type": "simulator",
            "protocol": "modbus_tcp",
            "configuration": {
                "registers": 1000,
                "functions": [1, 2, 3, 4, 5, 6]
            },
            "test_scenarios": ["basic_test", "performance_test"]
        }
    
    @pytest.fixture
    def sample_real_device_data(self) -> dict:
        """Sample real device data."""
        return {
            "id": "test_plc",
            "manufacturer": "Test Manufacturer",
            "model": "Test Model",
            "firmware": "1.0.0",
            "protocols": {
                "modbus_tcp": {
                    "status": "validated",
                    "performance": {
                        "throughput": "500 regs/sec",
                        "latency": "5ms"
                    }
                }
            },
            "test_notes": "Test device for API validation"
        }
    
    def test_create_api_instance(self):
        """Test creating API instance."""
        api = DeviceRegistryAPI()
        assert api is not None
        assert api.registry is not None
    
    def test_create_api_with_registry(self):
        """Test creating API with existing registry."""
        registry = DeviceRegistry()
        api = DeviceRegistryAPI(registry)
        assert api.registry is registry
    
    def test_get_empty_devices(self, api: DeviceRegistryAPI):
        """Test getting devices from empty registry."""
        virtual_devices = api.get_all_virtual_devices()
        real_devices = api.get_all_real_devices()
        
        assert virtual_devices == []
        assert real_devices == []
    
    def test_create_virtual_device(self, api: DeviceRegistryAPI, sample_virtual_device_data: dict):
        """Test creating virtual device via API."""
        result = api.create_virtual_device(sample_virtual_device_data)
        
        assert result['id'] == 'modbus_sim_test'
        assert result['type'] == 'simulator'
        assert result['protocol'] == 'modbus_tcp'
        
        # Verify it's in the registry
        devices = api.get_all_virtual_devices()
        assert len(devices) == 1
        assert devices[0]['id'] == 'modbus_sim_test'
    
    def test_create_real_device(self, api: DeviceRegistryAPI, sample_real_device_data: dict):
        """Test creating real device via API."""
        result = api.create_real_device(sample_real_device_data)
        
        assert result['id'] == 'test_plc'
        assert result['manufacturer'] == 'Test Manufacturer'
        assert result['model'] == 'Test Model'
        
        # Verify it's in the registry
        devices = api.get_all_real_devices()
        assert len(devices) == 1
        assert devices[0]['id'] == 'test_plc'
    
    def test_get_device_by_id(self, api: DeviceRegistryAPI, sample_virtual_device_data: dict):
        """Test getting device by ID."""
        api.create_virtual_device(sample_virtual_device_data)
        
        device = api.get_virtual_device('modbus_sim_test')
        assert device is not None
        assert device['id'] == 'modbus_sim_test'
        
        # Test non-existent device
        device = api.get_virtual_device('nonexistent')
        assert device is None
    
    def test_update_virtual_device(self, api: DeviceRegistryAPI, sample_virtual_device_data: dict):
        """Test updating virtual device."""
        api.create_virtual_device(sample_virtual_device_data)
        
        # Update the device
        updated_data = sample_virtual_device_data.copy()
        updated_data['type'] = 'updated_simulator'
        
        result = api.update_virtual_device('modbus_sim_test', updated_data)
        
        assert result is not None
        assert result['type'] == 'updated_simulator'
        
        # Verify in registry
        device = api.get_virtual_device('modbus_sim_test')
        assert device['type'] == 'updated_simulator'
    
    def test_update_nonexistent_device(self, api: DeviceRegistryAPI):
        """Test updating non-existent device."""
        result = api.update_virtual_device('nonexistent', {'id': 'nonexistent', 'type': 'test', 'protocol': 'test'})
        assert result is None
    
    def test_delete_virtual_device(self, api: DeviceRegistryAPI, sample_virtual_device_data: dict):
        """Test deleting virtual device."""
        api.create_virtual_device(sample_virtual_device_data)
        
        # Verify device exists
        assert len(api.get_all_virtual_devices()) == 1
        
        # Delete device
        result = api.delete_virtual_device('modbus_sim_test')
        assert result is True
        
        # Verify device is gone
        assert len(api.get_all_virtual_devices()) == 0
        
        # Try to delete non-existent device
        result = api.delete_virtual_device('nonexistent')
        assert result is False
    
    def test_find_devices_by_protocol(self, api: DeviceRegistryAPI, sample_virtual_device_data: dict, sample_real_device_data: dict):
        """Test finding devices by protocol."""
        api.create_virtual_device(sample_virtual_device_data)
        api.create_real_device(sample_real_device_data)
        
        devices = api.find_devices_by_protocol('modbus_tcp')
        
        assert len(devices['virtual']) == 1
        assert len(devices['real']) == 1
        assert devices['virtual'][0]['id'] == 'modbus_sim_test'
        assert devices['real'][0]['id'] == 'test_plc'
    
    def test_get_compatibility_report(self, api: DeviceRegistryAPI, sample_virtual_device_data: dict, sample_real_device_data: dict):
        """Test getting compatibility report."""
        api.create_virtual_device(sample_virtual_device_data)
        api.create_real_device(sample_real_device_data)
        
        report = api.get_compatibility_report('modbus_tcp')
        
        assert report['protocol'] == 'modbus_tcp'
        assert report['virtual_devices_count'] == 1
        assert report['real_devices_count'] == 1
        assert report['total_tested'] == 2
    
    def test_get_supported_protocols(self, api: DeviceRegistryAPI, sample_virtual_device_data: dict, sample_real_device_data: dict):
        """Test getting supported protocols."""
        # Empty registry
        protocols = api.get_supported_protocols()
        assert protocols == []
        
        # Add devices
        api.create_virtual_device(sample_virtual_device_data)
        api.create_real_device(sample_real_device_data)
        
        protocols = api.get_supported_protocols()
        assert 'modbus_tcp' in protocols
    
    def test_export_import_json(self, api: DeviceRegistryAPI, sample_virtual_device_data: dict):
        """Test export/import functionality."""
        api.create_virtual_device(sample_virtual_device_data)
        
        # Export
        exported_data = api.export_registry('json')
        assert exported_data is not None
        
        # Clear registry and import
        api.registry.clear()
        assert len(api.get_all_virtual_devices()) == 0
        
        api.import_registry(exported_data, 'json')
        assert len(api.get_all_virtual_devices()) == 1
    
    def test_export_import_yaml(self, api: DeviceRegistryAPI, sample_virtual_device_data: dict):
        """Test YAML export/import functionality."""
        api.create_virtual_device(sample_virtual_device_data)
        
        # Export
        exported_data = api.export_registry('yaml')
        assert exported_data is not None
        
        # Clear registry and import
        api.registry.clear()
        assert len(api.get_all_virtual_devices()) == 0
        
        api.import_registry(exported_data, 'yaml')
        assert len(api.get_all_virtual_devices()) == 1
    
    def test_export_invalid_format(self, api: DeviceRegistryAPI):
        """Test export with invalid format."""
        with pytest.raises(ValueError):
            api.export_registry('invalid_format')
    
    def test_import_invalid_format(self, api: DeviceRegistryAPI):
        """Test import with invalid format."""
        with pytest.raises(ValueError):
            api.import_registry('{}', 'invalid_format')
    
    def test_get_registry_stats(self, api: DeviceRegistryAPI, sample_virtual_device_data: dict, sample_real_device_data: dict):
        """Test getting registry statistics."""
        # Empty registry
        stats = api.get_registry_stats()
        assert stats['total_virtual_devices'] == 0
        assert stats['total_real_devices'] == 0
        assert stats['total_devices'] == 0
        assert stats['supported_protocols'] == 0
        
        # Add devices
        api.create_virtual_device(sample_virtual_device_data)
        api.create_real_device(sample_real_device_data)
        
        stats = api.get_registry_stats()
        assert stats['total_virtual_devices'] == 1
        assert stats['total_real_devices'] == 1
        assert stats['total_devices'] == 2
        assert stats['supported_protocols'] == 1
        assert 'modbus_tcp' in stats['protocol_distribution']
        assert stats['protocol_distribution']['modbus_tcp'] == 2
    
    def test_invalid_device_data(self, api: DeviceRegistryAPI):
        """Test creating device with invalid data."""
        with pytest.raises(Exception):  # Should raise ValidationError
            api.create_virtual_device({})  # Missing required fields
        
        with pytest.raises(Exception):  # Should raise ValidationError
            api.create_real_device({'id': 'test'})  # Missing required fields


class TestDeviceRegistryAPIIntegration:
    """Integration tests for the API with example data."""
    
    def test_example_data_workflow(self):
        """Test complete workflow with example data from issue."""
        api = DeviceRegistryAPI()
        
        # Create virtual device from example
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
        
        # Create real device from example
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
        
        # Create devices
        virtual_result = api.create_virtual_device(virtual_device_data)
        real_result = api.create_real_device(real_device_data)
        
        assert virtual_result['id'] == "modbus_tcp_sim_v1.0"
        assert real_result['id'] == "schneider_m221"
        
        # Test search and compatibility
        devices = api.find_devices_by_protocol("modbus_tcp")
        assert len(devices['virtual']) == 1
        assert len(devices['real']) == 1
        
        report = api.get_compatibility_report("modbus_tcp")
        assert report['total_tested'] == 2
        
        # Test export/import
        exported = api.export_registry('json')
        
        # Create new API and import
        new_api = DeviceRegistryAPI()
        new_api.import_registry(exported, 'json')
        
        # Verify data integrity
        assert len(new_api.get_all_virtual_devices()) == 1
        assert len(new_api.get_all_real_devices()) == 1
        
        new_virtual = new_api.get_virtual_device("modbus_tcp_sim_v1.0")
        new_real = new_api.get_real_device("schneider_m221")
        
        assert new_virtual is not None
        assert new_real is not None
        assert new_virtual['configuration']['registers'] == 1000
        assert new_real['manufacturer'] == "Schneider Electric"
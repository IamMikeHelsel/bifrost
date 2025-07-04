"""Tests for bifrost-core typing utilities."""

import pytest
from datetime import datetime
from bifrost_core import (
    Tag, DeviceInfo, ReadRequest, WriteRequest, PollingConfig,
    DataType, ProtocolType, parse_address, validate_data_type_conversion,
    get_default_value
)


class TestTag:
    """Test Tag class."""
    
    def test_tag_creation(self):
        tag = Tag(
            name="temperature",
            address="40001",
            data_type=DataType.FLOAT32,
            description="Temperature sensor",
            units="°C"
        )
        
        assert tag.name == "temperature"
        assert tag.address == "40001"
        assert tag.data_type == DataType.FLOAT32
        assert tag.description == "Temperature sensor"
        assert tag.units == "°C"
        assert not tag.read_only
    
    def test_tag_scaling(self):
        tag = Tag(
            name="pressure",
            address="40002",
            data_type=DataType.FLOAT32,
            scaling_factor=0.1,
            offset=10.0
        )
        
        # Test scaling: (raw_value * scaling_factor) + offset
        scaled = tag.apply_scaling(100)  # (100 * 0.1) + 10.0 = 20.0
        assert scaled == 20.0
    
    def test_tag_scaling_integer_result(self):
        tag = Tag(
            name="count",
            address="40003",
            data_type=DataType.INT32,
            scaling_factor=2,
            offset=5
        )
        
        # Should return integer for integer data types
        scaled = tag.apply_scaling(10)  # (10 * 2) + 5 = 25
        assert scaled == 25
        assert isinstance(scaled, int)
    
    def test_tag_no_scaling(self):
        tag = Tag(
            name="raw_value",
            address="40004", 
            data_type=DataType.INT32
        )
        
        # No scaling applied
        result = tag.apply_scaling(42)
        assert result == 42
    
    def test_tag_string_representation(self):
        tag = Tag("temp", "40001", DataType.FLOAT32)
        assert "Tag(temp, 40001, float32)" in str(tag)


class TestDeviceInfo:
    """Test DeviceInfo class."""
    
    def test_device_info_creation(self):
        device = DeviceInfo(
            device_id="PLC001",
            protocol=ProtocolType.MODBUS_TCP,
            host="192.168.1.100",
            port=502,
            name="Main PLC",
            manufacturer="Schneider Electric",
            model="M221"
        )
        
        assert device.device_id == "PLC001"
        assert device.protocol == ProtocolType.MODBUS_TCP
        assert device.host == "192.168.1.100"
        assert device.port == 502
        assert device.name == "Main PLC"
        assert device.manufacturer == "Schneider Electric"
        assert device.model == "M221"
    
    def test_device_connection_string(self):
        device = DeviceInfo(
            device_id="PLC001",
            protocol=ProtocolType.MODBUS_TCP,
            host="192.168.1.100",
            port=502
        )
        
        assert device.connection_string == "modbus_tcp://192.168.1.100:502"
    
    def test_device_connection_string_no_port(self):
        device = DeviceInfo(
            device_id="PLC001",
            protocol=ProtocolType.OPCUA,
            host="192.168.1.100"
        )
        
        assert device.connection_string == "opcua://192.168.1.100"
    
    def test_device_default_name(self):
        device = DeviceInfo(
            device_id="PLC001",
            protocol=ProtocolType.MODBUS_TCP,
            host="192.168.1.100"
        )
        
        # Name should default to device_id
        assert device.name == "PLC001"


class TestRequests:
    """Test request classes."""
    
    def test_read_request(self):
        tags = [
            Tag("temp", "40001", DataType.FLOAT32),
            Tag("pressure", "40002", DataType.FLOAT32)
        ]
        
        request = ReadRequest(tags, device_id="PLC001", timeout=5.0)
        
        assert request.tag_count == 2
        assert request.device_id == "PLC001"
        assert request.timeout == 5.0
        assert request.priority == 0
    
    def test_write_request(self):
        tag = Tag("setpoint", "40003", DataType.FLOAT32)
        request = WriteRequest(tag, 25.5, device_id="PLC001")
        
        assert request.tag == tag
        assert request.value == 25.5
        assert request.device_id == "PLC001"
    
    def test_write_request_read_only_tag(self):
        tag = Tag("readonly", "40004", DataType.FLOAT32, read_only=True)
        
        with pytest.raises(ValueError, match="read-only"):
            WriteRequest(tag, 100)


class TestPollingConfig:
    """Test PollingConfig class."""
    
    def test_polling_config_defaults(self):
        config = PollingConfig()
        
        assert config.interval_ms == 1000
        assert config.interval_seconds == 1.0
        assert config.max_batch_size == 100
        assert config.enabled is True
        assert config.max_consecutive_errors == 5
    
    def test_polling_config_custom(self):
        config = PollingConfig(
            interval_ms=500,
            max_batch_size=50,
            enabled=False,
            on_error_interval_ms=2000
        )
        
        assert config.interval_ms == 500
        assert config.interval_seconds == 0.5
        assert config.max_batch_size == 50
        assert config.enabled is False
        assert config.on_error_interval_ms == 2000
        assert config.on_error_interval_seconds == 2.0
    
    def test_polling_config_error_interval_default(self):
        config = PollingConfig(interval_ms=1000)
        
        # Should default to 2x the normal interval
        assert config.on_error_interval_ms == 2000


class TestUtilityFunctions:
    """Test utility functions."""
    
    def test_parse_address_modbus(self):
        # Test simple address
        result = parse_address("40001", ProtocolType.MODBUS_TCP)
        assert result["register_type"] == "holding"
        assert result["address"] == 40001
        
        # Test prefixed address
        result = parse_address("coil:1", ProtocolType.MODBUS_TCP)
        assert result["register_type"] == "coil"
        assert result["address"] == 1
    
    def test_parse_address_opcua(self):
        result = parse_address("ns=2;i=1", ProtocolType.OPCUA)
        assert result["node_id"] == "ns=2;i=1"
    
    def test_parse_address_generic(self):
        result = parse_address("some_address", ProtocolType.S7)
        assert result["address"] == "some_address"
    
    def test_validate_data_type_conversion(self):
        # Valid conversions
        assert validate_data_type_conversion("123", DataType.INT32) is True
        assert validate_data_type_conversion(123.5, DataType.FLOAT32) is True
        assert validate_data_type_conversion(1, DataType.BOOL) is True
        assert validate_data_type_conversion(123, DataType.STRING) is True
        
        # Invalid conversions
        assert validate_data_type_conversion("not_a_number", DataType.INT32) is False
        assert validate_data_type_conversion(None, DataType.FLOAT32) is False
    
    def test_get_default_value(self):
        assert get_default_value(DataType.BOOL) is False
        assert get_default_value(DataType.INT32) == 0
        assert get_default_value(DataType.FLOAT32) == 0.0
        assert get_default_value(DataType.STRING) == ""
        assert get_default_value(DataType.BYTES) == b""
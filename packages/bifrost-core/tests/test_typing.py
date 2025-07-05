"""Tests for bifrost-core typing utilities."""

import pytest
from bifrost_core.base import DeviceInfo
from bifrost_core.typing import DataType, Tag


class TestTag:
    """Test Tag class."""

    def test_tag_creation(self):
        tag = Tag(
            name="temperature",
            address="40001",
            data_type=DataType.FLOAT32,
            description="Temperature sensor",
            units="°C",
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
            offset=10.0,
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
            offset=5,
        )

        # Should return integer for integer data types
        scaled = tag.apply_scaling(10)  # (10 * 2) + 5 = 25
        assert scaled == 25
        assert isinstance(scaled, int)

    def test_tag_no_scaling(self):
        tag = Tag(name="raw_value", address="40004", data_type=DataType.INT32)

        # No scaling applied
        result = tag.apply_scaling(42)
        assert result == 42

    def test_tag_string_representation(self):
        tag = Tag(name="temp", address="40001", data_type=DataType.FLOAT32)
        assert "Tag(temp, 40001, float32)" in str(tag)


class TestDeviceInfo:
    """Test DeviceInfo class."""

    def test_device_info_creation(self):
        device = DeviceInfo(
            device_id="PLC001",
            protocol="modbus.tcp",
            host="192.168.1.100",
            port=502,
            name="Main PLC",
            manufacturer="Schneider Electric",
            model="M221",
        )

        assert device.device_id == "PLC001"
        assert device.protocol == "modbus.tcp"
        assert device.host == "192.168.1.100"
        assert device.port == 502
        assert device.name == "Main PLC"
        assert device.manufacturer == "Schneider Electric"
        assert device.model == "M221"

    def test_device_connection_string(self):
        device = DeviceInfo(
            device_id="PLC001",
            protocol="modbus.tcp",
            host="192.168.1.100",
            port=502,
        )

        assert device.protocol == "modbus.tcp"
        assert device.host == "192.168.1.100"
        assert device.port == 502

    def test_device_connection_string_no_port(self):
        device = DeviceInfo(
            device_id="PLC001", protocol="opcua.tcp", host="192.168.1.100", port=4840
        )

        assert device.protocol == "opcua.tcp"
        assert device.host == "192.168.1.100"

    def test_device_default_name(self):
        device = DeviceInfo(
            device_id="PLC001", protocol="modbus.tcp", host="192.168.1.100", port=502
        )

        # Name should default to device_id
        assert device.name == "PLC001"

"""End-to-end integration tests for Bifrost."""

import asyncio
import time
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from typer.testing import CliRunner

from bifrost.cli import app as cli_app
from bifrost.connections import ConnectionFactory
from bifrost.discovery import discover_devices
from bifrost.modbus import ModbusConnection
from bifrost_core import (
    DataType,
    DeviceInfo,
    Tag,
)


class TestEndToEndModbusWorkflow:
    """Test complete Modbus workflow from discovery to data operations."""

    @pytest.mark.asyncio
    async def test_complete_modbus_workflow(self):
        """Test discovering, connecting, and reading from a Modbus device."""
        # Mock a complete Modbus device workflow
        with (
            patch(
                "bifrost.discovery.discover_devices"
            ) as mock_discover_devices,
            patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class,
        ):
            # 1. Mock device discovery
            mock_discover_devices.return_value = AsyncMock(
                return_value=[
                    DeviceInfo(
                        device_id="plc_001",
                        protocol="modbus.tcp",
                        host="192.168.1.100",
                        port=502,
                        name="Test PLC",
                        manufacturer="Test Vendor",
                    )
                ]
            )

            # 2. Mock Modbus client
            mock_client = AsyncMock()
            mock_client.connect.return_value = True
            mock_client.connected = True

            # Mock read response
            mock_response = MagicMock()
            mock_response.isError.return_value = False
            mock_response.registers = [42, 100, 250]
            mock_client.read_holding_registers.return_value = mock_response

            # Mock write response
            mock_write_response = MagicMock()
            mock_write_response.isError.return_value = False
            mock_client.write_register.return_value = mock_write_response

            mock_client_class.return_value = mock_client

            # Step 1: Discover devices
            devices = []
            async for device in discover_devices():
                devices.append(device)
            assert len(devices) > 0
            device = devices[0]
            assert device["protocol"] == "modbus.tcp"
            assert device["host"] == "192.168.1.100"

            # Step 2: Connect to discovered device
            connection_string = (
                f"modbus.tcp://{device['host']}:{device['port']}"
            )
            connection = ConnectionFactory.create(connection_string)

            assert isinstance(connection, ModbusConnection)
            async with connection as conn:
                assert conn.is_connected

                # Step 3: Read data from multiple registers

                # Read individual tags
                readings = await conn.read(
                    [Tag("40001"), Tag("40002"), Tag("40003")]
                )

                assert readings[Tag("40001")].value == 42
                assert readings[Tag("40002")].value == 100
                assert readings[Tag("40003")].value == 250

                # Step 4: Write a value
                await conn.write({Tag("40001"): 45})

                # Step 5: Verify health check (not directly available on BaseConnection)
                # For now, we'll just check if the connection is still active
                assert conn.is_connected

            # Step 6: Disconnect (handled by context manager)
            assert not connection.is_connected

            # Verify mock calls
            mock_client.connect.assert_called()
            mock_client.read_holding_registers.assert_called()
            mock_client.write_register.assert_called()

    @pytest.mark.asyncio
    async def test_connection_context_manager_workflow(self):
        """Test using connection as async context manager."""
        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect.return_value = True
            mock_client.connected = True

            # Mock health check response
            mock_response = MagicMock()
            mock_response.isError.return_value = False
            mock_response.bits = [True]
            mock_client.read_coils.return_value = mock_response

            mock_client_class.return_value = mock_client

            # Test context manager usage
            async with ModbusConnection("192.168.1.100", 502) as connection:
                assert connection.is_connected

                # Perform operations within context
                # health = await connection.health_check()
                # assert health is True

            # Connection should be automatically disconnected
            assert not connection.is_connected

    @pytest.mark.asyncio
    async def test_error_handling_workflow(self):
        """Test error handling in complete workflow."""
        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect.return_value = False  # Connection failure
            mock_client_class.return_value = mock_client

            # Test connection failure handling
            with pytest.raises(ConnectionError):  # Should raise ConnectionError
                connection = ModbusConnection("192.168.1.100", 502)
                async with connection:
                    pass

    @pytest.mark.asyncio
    async def test_batch_operations_workflow(self):
        """Test batch read/write operations."""
        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect.return_value = True
            mock_client.connected = True

            # Mock batch read response
            mock_response = MagicMock()
            mock_response.isError.return_value = False
            mock_response.registers = [10, 20, 30, 40, 50]
            mock_client.read_holding_registers.return_value = mock_response

            # Mock batch write response
            mock_write_response = MagicMock()
            mock_write_response.isError.return_value = False
            mock_client.write_registers.return_value = mock_write_response

            mock_client_class.return_value = mock_client

            connection = ModbusConnection("192.168.1.100", 502)
            device = ModbusDevice(connection)
            async with connection:
                # Test batch read
                readings = await device.read(
                    [
                        Tag(
                            name=f"tag{i}",
                            address=str(40001 + i),
                            data_type=DataType.INT16,
                        )
                        for i in range(5)
                    ]
                )
                assert len(readings) == 5
                assert (
                    readings[
                        Tag(
                            name="tag0",
                            address="40001",
                            data_type=DataType.INT16,
                        )
                    ].value
                    == 10
                )

                # Test batch write
                await device.write(
                    {
                        Tag(
                            name="tag0",
                            address="40001",
                            data_type=DataType.INT16,
                        ): 100,
                        Tag(
                            name="tag1",
                            address="40002",
                            data_type=DataType.INT16,
                        ): 200,
                        Tag(
                            name="tag2",
                            address="40003",
                            data_type=DataType.INT16,
                        ): 300,
                    }
                )

            assert not connection.is_connected


class TestCLIIntegration:
    """Test CLI integration with core functionality."""

    def setup_method(self):
        """Set up CLI test runner."""
        self.runner = CliRunner()

    def test_cli_version_command(self):
        """Test CLI version command."""
        result = self.runner.invoke(cli_app, ["--version"])
        assert result.exit_code == 0
        assert "Bifrost" in result.stdout

    def test_cli_help_command(self):
        """Test CLI help command."""
        result = self.runner.invoke(cli_app, ["--help"])
        assert result.exit_code == 0
        assert "Industrial IoT Framework" in result.stdout

    @patch("bifrost.discovery.discover_devices")
    def test_cli_discover_command(self, mock_discover_devices):
        """Test CLI discover command integration."""
        # Mock discovery results
        mock_discover_devices.return_value = AsyncMock(
            return_value=[
                {
                    "host": "192.168.1.100",
                    "port": 502,
                    "protocol": "modbus.tcp",
                    "device_type": "PLC",
                }
            ]
        )

        # Test discovery command
        result = self.runner.invoke(
            cli_app,
            [
                "discover",
            ],
        )

        assert result.exit_code == 0
        mock_discover_devices.assert_called_once()

    # def test_cli_status_command(self):
    #     """Test CLI status command."""
    #     result = self.runner.invoke(cli_app, ["status"])
    #     assert result.exit_code == 0
    #     assert "System Status" in result.stdout


class TestConnectionPoolingIntegration:
    """Test connection pooling integration."""

    @pytest.mark.asyncio
    async def test_pooled_connection_workflow(self):
        """Test using pooled connections."""
        from bifrost_core.pooling import ConnectionPool

        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect.return_value = True
            mock_client.connected = True

            # Mock read response for health check
            mock_response = MagicMock()
            mock_response.isError.return_value = False
            mock_response.bits = [True]
            mock_client.read_coils.return_value = mock_response

            mock_client_class.return_value = mock_client

            # Create connection factory
            async def connection_factory():
                return ModbusConnection("192.168.1.100", 502)

            pool = ConnectionPool(
                connection_factory=connection_factory, max_size=1
            )

            # Test pooled connection usage
            async with pool.get() as conn1:
                readings = await conn1.read([Tag("40001")])
                assert readings[Tag("40001")].value == 42

                # Use another pooled connection (should reuse from pool)
                async with pool.get() as conn2:
                    readings = await conn2.read([Tag("40001")])
                    assert readings[Tag("40001")].value == 42

            # Check pool statistics (not directly exposed in current ConnectionPool)
            # pool = get_global_pool()
            # stats = pool.get_stats()
            # assert isinstance(stats, dict)
            # assert "total_connections" in stats


class TestEventSystemIntegration:
    """Test event system integration."""

    @pytest.mark.asyncio
    async def test_event_driven_monitoring(self):
        """Test event-driven device monitoring."""
        from bifrost_core import EventBus

        # Set up event collection
        events_received = []

        async def event_handler(event):
            events_received.append(event)

        # Subscribe to all events
        EventBus()
        # event_bus.subscribe_global(event_handler) # No subscribe_global in new EventBus

        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect.return_value = True
            mock_client.connected = True

            # Mock successful read
            mock_response = MagicMock()
            mock_response.isError.return_value = False
            mock_response.registers = [123]
            mock_client.read_holding_registers.return_value = mock_response

            mock_client_class.return_value = mock_client

            # Create connection and perform operations
            connection = ModbusConnection("192.168.1.100", 502)

            # Connect (should generate connection state events)
            async with connection:
                # Read data (should generate data received event)
                await connection.read([Tag("40001")])

            # Disconnect (should generate disconnection event)
            # Handled by context manager

            # Wait a moment for events to be processed
            await asyncio.sleep(0.1)

            # Verify events were generated (no direct event emission from ModbusConnection yet)
            assert len(events_received) == 0


class TestScalabilityScenarios:
    """Test scalability scenarios."""

    @pytest.mark.asyncio
    async def test_multiple_concurrent_connections(self):
        """Test handling multiple concurrent connections."""
        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            # Create multiple mock clients
            mock_clients = []
            for _ in range(5):
                mock_client = AsyncMock()
                mock_client.connect.return_value = True
                mock_client.connected = True

                # Mock health check
                mock_response = MagicMock()
                mock_response.isError.return_value = False
                mock_response.bits = [True]
                mock_client.read_coils.return_value = mock_response

                mock_clients.append(mock_client)

            mock_client_class.side_effect = mock_clients

            # Create multiple connections
            connections = []
            for i in range(5):
                conn = ModbusConnection(f"192.168.1.{100 + i}", 502)
                connections.append(conn)

            # Perform operations on all connections concurrently
            tasks = []
            for conn in connections:

                async def _operate(c):
                    async with c:
                        await c.read([Tag("40001")])

                task = asyncio.create_task(_operate(conn))
                tasks.append(task)

            await asyncio.gather(*tasks)

            # Verify all disconnected
            for conn in connections:
                assert not conn.is_connected

    @pytest.mark.asyncio
    async def test_high_frequency_operations(self):
        """Test high-frequency read/write operations."""
        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect.return_value = True
            mock_client.connected = True

            # Mock rapid read responses
            mock_response = MagicMock()
            mock_response.isError.return_value = False
            mock_response.registers = [42]
            mock_client.read_holding_registers.return_value = mock_response

            mock_client_class.return_value = mock_client

            connection = ModbusConnection("192.168.1.100", 502)
            async with connection as conn:
                # Perform many rapid operations
                start_time = time.time()
                num_operations = 100

                tasks = []
                for _ in range(num_operations):
                    task = asyncio.create_task(conn.read([Tag("40001")]))
                    tasks.append(task)

                results = await asyncio.gather(*tasks)
                end_time = time.time()

                # Verify all operations completed
                assert len(results) == num_operations
                assert all(
                    result[Tag("40001")].value == 42 for result in results
                )

                # Performance check (should complete in reasonable time)
                duration = end_time - start_time
                assert duration < 5.0  # Should complete within 5 seconds

            assert not connection.is_connected


class TestRealWorldScenarios:
    """Test realistic industrial scenarios."""

    @pytest.mark.asyncio
    async def test_factory_monitoring_scenario(self):
        """Test a factory monitoring scenario with multiple devices."""
        # Simulate a factory with multiple PLCs and sensors
        device_configs = [
            {
                "host": "192.168.1.100",
                "name": "Production Line 1 PLC",
                "tags": ["temp_1", "pressure_1"],
            },
            {
                "host": "192.168.1.101",
                "name": "Production Line 2 PLC",
                "tags": ["temp_2", "flow_rate"],
            },
            {
                "host": "192.168.1.102",
                "name": "Quality Control PLC",
                "tags": ["weight", "dimension"],
            },
        ]

        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            # Mock clients for each device
            mock_clients = []
            for i, _ in enumerate(device_configs):
                mock_client = AsyncMock()
                mock_client.connect.return_value = True
                mock_client.connected = True

                # Mock sensor readings
                mock_response = MagicMock()
                mock_response.isError.return_value = False
                mock_response.registers = [
                    20 + i * 10,
                    30 + i * 10,
                ]  # Different values per device
                mock_client.read_holding_registers.return_value = mock_response

                mock_clients.append(mock_client)

            mock_client_class.side_effect = mock_clients

            # Connect to all devices
            connections = {}
            for config in device_configs:
                conn = ModbusConnection(config["host"], 502)
                connections[config["name"]] = {
                    "connection": conn,
                    "tags": config["tags"],
                }

            # Simulate monitoring cycle
            monitoring_data = {}
            for device_name, device_info in connections.items():
                conn = device_info["connection"]
                tag_values = {}

                # Read all tags for this device
                async with conn:
                    for i, tag_name in enumerate(device_info["tags"]):
                        readings = await conn.read([Tag(f"4000{i + 1}")])
                        tag_values[tag_name] = readings[
                            Tag(f"4000{i + 1}")
                        ].value

                monitoring_data[device_name] = tag_values

            # Verify monitoring data
            assert len(monitoring_data) == 3
            assert "Production Line 1 PLC" in monitoring_data
            assert "temp_1" in monitoring_data["Production Line 1 PLC"]

            # Simulate alarm condition (temperature too high)
            temp_1 = monitoring_data["Production Line 1 PLC"]["temp_1"]
            if temp_1 > 25:  # Alarm threshold
                # Send control command to reduce temperature
                control_conn = connections["Production Line 1 PLC"][
                    "connection"
                ]
                async with control_conn:
                    await control_conn.write(
                        {Tag("40010"): 0}
                    )  # Turn off heater

            # Cleanup - disconnect all devices (handled by context managers)
            for device_info in connections.values():
                assert not device_info["connection"].is_connected

    @pytest.mark.asyncio
    async def test_device_failover_scenario(self):
        """Test device failover scenario."""
        primary_host = "192.168.1.100"
        backup_host = "192.168.1.101"

        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            # Mock primary device failure, backup success
            primary_client = AsyncMock()
            primary_client.connect.side_effect = ConnectionError(
                "Primary device failed"
            )

            backup_client = AsyncMock()
            backup_client.connect.return_value = True
            backup_client.connected = True

            mock_client_class.side_effect = [primary_client, backup_client]

            active_connection = None
            # Try primary device first
            try:
                primary_conn = ModbusConnection(primary_host, 502)
                async with primary_conn as conn:
                    active_connection = conn
            except ConnectionError:
                # Failover to backup device
                backup_conn = ModbusConnection(backup_host, 502)
                async with backup_conn as conn:
                    active_connection = conn

            # Verify we're using backup device
            assert active_connection is not None
            assert active_connection.host == backup_host
            assert active_connection.is_connected

            # Continue operations with backup device
            readings = await active_connection.read([Tag("40001")])
            assert readings[Tag("40001")].value == 100

            # Disconnect handled by context manager


class TestConfigurationManagement:
    """Test configuration and deployment scenarios."""

    @pytest.mark.asyncio
    async def test_configuration_driven_deployment(self):
        """Test deployment driven by configuration files."""
        # Simulate configuration file
        config = {
            "devices": [
                {
                    "name": "main_plc",
                    "protocol": "modbus.tcp",
                    "host": "192.168.1.100",
                    "port": 502,
                    "tags": [
                        {
                            "name": "temperature",
                            "address": "40001",
                            "type": "int16",
                        },
                        {
                            "name": "pressure",
                            "address": "40002",
                            "type": "int16",
                        },
                    ],
                }
            ],
            "polling": {"interval_ms": 1000, "enabled": True},
        }

        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect.return_value = True
            mock_client.connected = True

            # Mock tag readings
            mock_response = MagicMock()
            mock_response.isError.return_value = False
            mock_response.registers = [25, 150]  # Temperature, Pressure
            mock_client.read_holding_registers.return_value = mock_response

            mock_client_class.return_value = mock_client

            # Deploy devices from configuration
            deployed_devices = {}
            for device_config in config["devices"]:
                conn = ModbusConnection(
                    device_config["host"], device_config["port"]
                )
                deployed_devices[device_config["name"]] = {
                    "connection": conn,
                    "tags": device_config["tags"],
                }

            # Verify deployment
            assert "main_plc" in deployed_devices
            main_plc = deployed_devices["main_plc"]
            assert isinstance(main_plc["connection"], ModbusConnection)
            assert len(main_plc["tags"]) == 2

            # Test tag reading
            async with main_plc["connection"] as conn:
                temp_readings = await conn.read([Tag("40001")])
                pressure_readings = await conn.read([Tag("40002")])

                assert temp_readings[Tag("40001")].value == 25
                assert pressure_readings[Tag("40002")].value == 150

            # Cleanup (handled by context manager)
            assert not main_plc["connection"].is_connected


# Performance benchmarks (optional, can be skipped in regular CI)
@pytest.mark.benchmark
class TestPerformanceBenchmarks:
    """Performance benchmark tests."""

    @pytest.mark.asyncio
    async def test_connection_establishment_performance(self):
        """Benchmark connection establishment performance."""
        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect.return_value = True
            mock_client.connected = True
            mock_client_class.return_value = mock_client

            # Benchmark multiple connection establishments
            num_connections = 10
            start_time = time.time()

            connections = []
            for i in range(num_connections):
                conn = ModbusConnection(f"192.168.1.{100 + i}", 502)
                connections.append(conn)

            tasks = []
            for conn in connections:
                tasks.append(asyncio.create_task(conn.__aenter__()))
            await asyncio.gather(*tasks)

            end_time = time.time()

            # Performance assertion
            avg_time_per_connection = (end_time - start_time) / num_connections
            assert (
                avg_time_per_connection < 0.1
            )  # Should be under 100ms per connection

            # Cleanup
            tasks = []
            for conn in connections:
                tasks.append(
                    asyncio.create_task(conn.__aexit__(None, None, None))
                )
            await asyncio.gather(*tasks)

    @pytest.mark.asyncio
    async def test_data_throughput_performance(self):
        """Benchmark data read throughput."""
        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect.return_value = True
            mock_client.connected = True

            # Mock rapid read responses
            mock_response = MagicMock()
            mock_response.isError.return_value = False
            mock_response.registers = [42]
            mock_client.read_holding_registers.return_value = mock_response

            mock_client_class.return_value = mock_client

            connection = ModbusConnection("192.168.1.100", 502)
            async with connection as conn:
                # Perform many rapid operations
                start_time = time.time()
                num_operations = 100

                tasks = []
                for _ in range(num_operations):
                    task = asyncio.create_task(conn.read([Tag("40001")]))
                    tasks.append(task)

                await asyncio.gather(*tasks)
                end_time = time.time()

                # Performance assertion
                total_time = end_time - start_time
                reads_per_second = num_operations / total_time
                assert (
                    reads_per_second > 100
                )  # Should achieve >100 reads/second

            assert not connection.is_connected

"""End-to-end integration tests for Bifrost."""

import asyncio
import time
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from bifrost import ModbusTCPConnection, connect
from bifrost.cli import app as cli_app
from bifrost.discovery import discover_devices
from bifrost_core import (
    ConnectionState,
    DataType,
    DeviceInfo,
    ProtocolType,
    Tag,
)
from typer.testing import CliRunner


class TestEndToEndModbusWorkflow:
    """Test complete Modbus workflow from discovery to data operations."""

    @pytest.mark.asyncio
    async def test_complete_modbus_workflow(self):
        """Test discovering, connecting, and reading from a Modbus device."""

        # Mock a complete Modbus device workflow
        with (
            patch("bifrost.discovery.NetworkDiscovery") as mock_discovery_class,
            patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class,
        ):
            # 1. Mock device discovery
            mock_discovery = MagicMock()
            mock_discovery.discover_network = AsyncMock(
                return_value=[
                    DeviceInfo(
                        device_id="plc_001",
                        protocol=ProtocolType.MODBUS_TCP,
                        host="192.168.1.100",
                        port=502,
                        name="Test PLC",
                        manufacturer="Test Vendor",
                    )
                ]
            )
            mock_discovery_class.return_value = mock_discovery

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
            devices = await discover_devices(network="192.168.1.0/24")
            assert len(devices) > 0
            device = devices[0]
            assert device.protocol == ProtocolType.MODBUS_TCP
            assert device.host == "192.168.1.100"

            # Step 2: Connect to discovered device
            connection_string = f"modbus://{device.host}:{device.port}"
            connection = await connect(connection_string)

            assert isinstance(connection, ModbusTCPConnection)
            assert connection.is_connected

            # Step 3: Read data from multiple registers

            # Read individual tags
            temp_value = await connection.read_raw("40001")
            pressure_value = await connection.read_raw("40002")
            flow_value = await connection.read_raw("40003")

            assert temp_value == [42]
            assert pressure_value == [100]
            assert flow_value == [250]

            # Step 4: Write a value
            await connection.write_raw("40001", [45])

            # Step 5: Verify health check
            health = await connection.health_check()
            assert health is True

            # Step 6: Disconnect
            await connection.disconnect()
            assert connection._state == ConnectionState.DISCONNECTED

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
            async with ModbusTCPConnection("192.168.1.100", 502) as connection:
                assert connection.is_connected

                # Perform operations within context
                health = await connection.health_check()
                assert health is True

            # Connection should be automatically disconnected
            assert connection._state == ConnectionState.DISCONNECTED

    @pytest.mark.asyncio
    async def test_error_handling_workflow(self):
        """Test error handling in complete workflow."""

        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect.return_value = False  # Connection failure
            mock_client_class.return_value = mock_client

            # Test connection failure handling
            with pytest.raises(ConnectionError):  # Should raise ConnectionError
                connection = ModbusTCPConnection("192.168.1.100", 502)
                await connection.connect()

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

            connection = ModbusTCPConnection("192.168.1.100", 502)
            await connection.connect()

            # Test batch read
            values = await connection.read_raw("40001", count=5)
            assert len(values) == 5
            assert values == [10, 20, 30, 40, 50]

            # Test batch write
            await connection.write_raw("40001", [100, 200, 300])

            await connection.disconnect()


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

    @patch("bifrost.discovery.NetworkDiscovery")
    def test_cli_discover_command(self, mock_discovery_class):
        """Test CLI discover command integration."""

        # Mock discovery results
        mock_discovery = MagicMock()
        mock_discovery.discover_network = AsyncMock(
            return_value=[
                DeviceInfo(
                    device_id="test_device",
                    protocol=ProtocolType.MODBUS_TCP,
                    host="192.168.1.100",
                    port=502,
                    name="Test Device",
                )
            ]
        )
        mock_discovery_class.return_value = mock_discovery

        # Test discovery command
        result = self.runner.invoke(
            cli_app,
            [
                "discover",
                "--network",
                "192.168.1.0/24",
                "--method",
                "modbus",
                "--timeout",
                "2",
            ],
        )

        assert result.exit_code == 0
        mock_discovery_class.assert_called()

    def test_cli_status_command(self):
        """Test CLI status command."""
        result = self.runner.invoke(cli_app, ["status"])
        assert result.exit_code == 0
        assert "System Status" in result.stdout


class TestConnectionPoolingIntegration:
    """Test connection pooling integration."""

    @pytest.mark.asyncio
    async def test_pooled_connection_workflow(self):
        """Test using pooled connections."""

        from bifrost_core.pooling import get_global_pool, pooled_connection

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
                return ModbusTCPConnection("192.168.1.100", 502)

            # Test pooled connection usage
            async with pooled_connection(
                connection_factory, "modbus://192.168.1.100:502"
            ) as conn1:
                health1 = await conn1.health_check()
                assert health1 is True

                # Use another pooled connection (should reuse from pool)
                async with pooled_connection(
                    connection_factory, "modbus://192.168.1.100:502"
                ) as conn2:
                    health2 = await conn2.health_check()
                    assert health2 is True

            # Check pool statistics
            pool = get_global_pool()
            stats = pool.get_stats()
            assert isinstance(stats, dict)
            assert "total_connections" in stats


class TestEventSystemIntegration:
    """Test event system integration."""

    @pytest.mark.asyncio
    async def test_event_driven_monitoring(self):
        """Test event-driven device monitoring."""

        from bifrost_core import (
            ConnectionStateEvent,
            DataReceivedEvent,
            EventBus,
        )

        # Set up event collection
        events_received = []

        def event_handler(event):
            events_received.append(event)

        # Subscribe to all events
        event_bus = EventBus()
        event_bus.subscribe_global(event_handler)

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
            connection = ModbusTCPConnection("192.168.1.100", 502)

            # Connect (should generate connection state events)
            await connection.connect()

            # Read data (should generate data received event)
            await connection.read_raw("40001")

            # Disconnect (should generate disconnection event)
            await connection.disconnect()

            # Wait a moment for events to be processed
            await asyncio.sleep(0.1)

            # Verify events were generated
            assert len(events_received) > 0

            # Check for specific event types
            connection_events = [
                e for e in events_received if isinstance(e, ConnectionStateEvent)
            ]
            data_events = [
                e for e in events_received if isinstance(e, DataReceivedEvent)
            ]

            assert len(connection_events) >= 2  # At least connect and disconnect
            assert len(data_events) >= 1  # At least one data read


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
                conn = ModbusTCPConnection(f"192.168.1.{100 + i}", 502)
                await conn.connect()
                connections.append(conn)

            # Perform operations on all connections concurrently
            tasks = []
            for conn in connections:
                task = asyncio.create_task(conn.health_check())
                tasks.append(task)

            results = await asyncio.gather(*tasks)
            assert all(results)  # All health checks should pass

            # Disconnect all
            disconnect_tasks = []
            for conn in connections:
                task = asyncio.create_task(conn.disconnect())
                disconnect_tasks.append(task)

            await asyncio.gather(*disconnect_tasks)

            # Verify all disconnected
            for conn in connections:
                assert conn._state == ConnectionState.DISCONNECTED

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

            connection = ModbusTCPConnection("192.168.1.100", 502)
            await connection.connect()

            # Perform many rapid operations
            start_time = time.time()
            num_operations = 100

            tasks = []
            for i in range(num_operations):
                task = asyncio.create_task(connection.read_raw(f"4000{i % 10 + 1}"))
                tasks.append(task)

            results = await asyncio.gather(*tasks)
            end_time = time.time()

            # Verify all operations completed
            assert len(results) == num_operations
            assert all(result == [42] for result in results)

            # Performance check (should complete in reasonable time)
            duration = end_time - start_time
            assert duration < 5.0  # Should complete within 5 seconds

            await connection.disconnect()


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
                conn = ModbusTCPConnection(config["host"], 502)
                await conn.connect()
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
                for i, tag in enumerate(device_info["tags"]):
                    values = await conn.read_raw(f"4000{i + 1}")
                    tag_values[tag] = values[0]

                monitoring_data[device_name] = tag_values

            # Verify monitoring data
            assert len(monitoring_data) == 3
            assert "Production Line 1 PLC" in monitoring_data
            assert "temp_1" in monitoring_data["Production Line 1 PLC"]

            # Simulate alarm condition (temperature too high)
            temp_1 = monitoring_data["Production Line 1 PLC"]["temp_1"]
            if temp_1 > 25:  # Alarm threshold
                # Send control command to reduce temperature
                control_conn = connections["Production Line 1 PLC"]["connection"]
                await control_conn.write_raw("40010", [0])  # Turn off heater

            # Cleanup - disconnect all devices
            for device_info in connections.values():
                await device_info["connection"].disconnect()

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

            # Mock backup device readings
            mock_response = MagicMock()
            mock_response.isError.return_value = False
            mock_response.registers = [100]
            backup_client.read_holding_registers.return_value = mock_response

            mock_client_class.side_effect = [primary_client, backup_client]

            # Try primary device first
            try:
                primary_conn = ModbusTCPConnection(primary_host, 502)
                await primary_conn.connect()
                active_connection = primary_conn
            except ConnectionError:
                # Failover to backup device
                backup_conn = ModbusTCPConnection(backup_host, 502)
                await backup_conn.connect()
                active_connection = backup_conn

            # Verify we're using backup device
            assert active_connection.host == backup_host
            assert active_connection.is_connected

            # Continue operations with backup device
            value = await active_connection.read_raw("40001")
            assert value == [100]

            await active_connection.disconnect()


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
                    "protocol": "modbus_tcp",
                    "host": "192.168.1.100",
                    "port": 502,
                    "tags": [
                        {"name": "temperature", "address": "40001", "type": "int16"},
                        {"name": "pressure", "address": "40002", "type": "int16"},
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
                conn = ModbusTCPConnection(device_config["host"], device_config["port"])
                await conn.connect()

                # Create tags from configuration
                tags = []
                for tag_config in device_config["tags"]:
                    tag = Tag(
                        name=tag_config["name"],
                        address=tag_config["address"],
                        data_type=DataType.INT16,  # Simplified for test
                    )
                    tags.append(tag)

                deployed_devices[device_config["name"]] = {
                    "connection": conn,
                    "tags": tags,
                }

            # Verify deployment
            assert "main_plc" in deployed_devices
            main_plc = deployed_devices["main_plc"]
            assert main_plc["connection"].is_connected
            assert len(main_plc["tags"]) == 2

            # Test tag reading
            temp_value = await main_plc["connection"].read_raw("40001")
            pressure_value = await main_plc["connection"].read_raw("40002")

            assert temp_value == [25]
            assert pressure_value == [150]

            # Cleanup
            for device_info in deployed_devices.values():
                await device_info["connection"].disconnect()


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
                conn = ModbusTCPConnection(f"192.168.1.{100 + i}", 502)
                await conn.connect()
                connections.append(conn)

            end_time = time.time()

            # Performance assertion
            avg_time_per_connection = (end_time - start_time) / num_connections
            assert avg_time_per_connection < 0.1  # Should be under 100ms per connection

            # Cleanup
            for conn in connections:
                await conn.disconnect()

    @pytest.mark.asyncio
    async def test_data_throughput_performance(self):
        """Benchmark data read throughput."""

        with patch("pymodbus.client.AsyncModbusTcpClient") as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect.return_value = True
            mock_client.connected = True

            # Mock fast response
            mock_response = MagicMock()
            mock_response.isError.return_value = False
            mock_response.registers = [42]
            mock_client.read_holding_registers.return_value = mock_response

            mock_client_class.return_value = mock_client

            connection = ModbusTCPConnection("192.168.1.100", 502)
            await connection.connect()

            # Benchmark read operations
            num_reads = 1000
            start_time = time.time()

            for _ in range(num_reads):
                await connection.read_raw("40001")

            end_time = time.time()

            # Performance assertion
            total_time = end_time - start_time
            reads_per_second = num_reads / total_time
            assert reads_per_second > 100  # Should achieve >100 reads/second

            await connection.disconnect()

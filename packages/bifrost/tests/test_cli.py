"""Tests for CLI commands."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from bifrost.cli import app
from bifrost_core import DataPoint
from typer.testing import CliRunner


class TestCLIBasics:
    """Test basic CLI functionality."""

    @pytest.fixture
    def runner(self):
        """Create CLI test runner."""
        return CliRunner()

    def test_help_command(self, runner):
        """Test help command."""
        result = runner.invoke(app, ["--help"])
        assert result.exit_code == 0
        assert "Bifrost - Industrial Edge Computing Framework" in result.stdout
        assert "discover" in result.stdout
        assert "connect" in result.stdout
        assert "scan" in result.stdout
        assert "status" in result.stdout


class TestDiscoverCommand:
    """Test discover command."""

    @pytest.fixture
    def runner(self):
        """Create CLI test runner."""
        return CliRunner()

    @pytest.fixture
    def mock_discovery(self):
        """Mock discovery module."""
        with patch("bifrost.discovery") as mock:
            # Mock discover_devices function
            async def mock_discover():
                return [
                    {
                        "ip": "192.168.1.100",
                        "mac": "00:11:22:33:44:55",
                        "vendor": "Siemens",
                        "device_type": "PLC",
                        "open_ports": [502],
                    },
                    {
                        "ip": "192.168.1.101",
                        "mac": "00:11:22:33:44:66",
                        "vendor": "Schneider",
                        "device_type": "HMI",
                        "open_ports": [502, 44818],
                    },
                ]

            mock.discover_devices = AsyncMock(side_effect=mock_discover)
            yield mock

    def test_discover_default(self, runner, mock_discovery):
        """Test discover command with default options."""
        result = runner.invoke(app, ["discover"])
        assert result.exit_code == 0
        assert "192.168.1.100" in result.stdout
        assert "Siemens" in result.stdout
        assert "PLC" in result.stdout

    def test_discover_json_format(self, runner, mock_discovery):
        """Test discover command with JSON output."""
        result = runner.invoke(app, ["discover", "--format", "json"])
        assert result.exit_code == 0
        assert '"ip": "192.168.1.100"' in result.stdout
        assert '"device_type": "PLC"' in result.stdout

    def test_discover_with_network(self, runner, mock_discovery):
        """Test discover command with specific network."""
        result = runner.invoke(app, ["discover", "--network", "10.0.0.0/24"])
        assert result.exit_code == 0
        mock_discovery.discover_devices.assert_called_once()
        args, kwargs = mock_discovery.discover_devices.call_args
        assert kwargs["network"] == "10.0.0.0/24"


class TestConnectCommand:
    """Test connect command."""

    @pytest.fixture
    def runner(self):
        """Create CLI test runner."""
        return CliRunner()

    @pytest.fixture
    def mock_connection(self):
        """Mock connection factory."""
        with patch("bifrost.modbus.ModbusTCPConnection") as mock:
            conn = MagicMock()
            conn.connect = AsyncMock()
            conn.disconnect = AsyncMock()
            conn.read = AsyncMock(return_value=DataPoint("40001", 1234))
            conn.write = AsyncMock()
            conn.is_connected = True
            conn.host = "192.168.1.100"
            conn.port = 502

            async def create_conn(*args, **kwargs):
                return conn

            mock.side_effect = create_conn
            yield mock, conn

    def test_connect_modbus_read(self, runner, mock_connection):
        """Test connect command with Modbus read."""
        mock_factory, mock_conn = mock_connection

        # Simulate non-interactive mode with read command
        result = runner.invoke(
            app, ["connect", "modbus://192.168.1.100:502", "--read", "40001"]
        )

        assert result.exit_code == 0
        assert "1234" in result.stdout
        mock_conn.read.assert_called_once_with("40001")

    def test_connect_modbus_write(self, runner, mock_connection):
        """Test connect command with Modbus write."""
        mock_factory, mock_conn = mock_connection

        result = runner.invoke(
            app, ["connect", "modbus://192.168.1.100:502", "--write", "40001=5678"]
        )

        assert result.exit_code == 0
        mock_conn.write.assert_called_once_with("40001", 5678)

    def test_connect_invalid_url(self, runner):
        """Test connect command with invalid URL."""
        result = runner.invoke(app, ["connect", "invalid-url"])
        assert result.exit_code != 0
        assert "Error" in result.stdout or "Invalid" in result.stdout


class TestScanCommand:
    """Test scan command."""

    @pytest.fixture
    def runner(self):
        """Create CLI test runner."""
        return CliRunner()

    @pytest.fixture
    def mock_scanner(self):
        """Mock port scanner."""
        with patch("bifrost.discovery.scan_ports") as mock:

            async def mock_scan(host, ports):
                return {502: "open", 44818: "open", 80: "closed"}

            mock.side_effect = mock_scan
            yield mock

    def test_scan_host(self, runner, mock_scanner):
        """Test scan command for a host."""
        result = runner.invoke(app, ["scan", "192.168.1.100"])
        assert result.exit_code == 0
        assert "502" in result.stdout
        assert "open" in result.stdout

    def test_scan_with_ports(self, runner, mock_scanner):
        """Test scan command with specific ports."""
        result = runner.invoke(app, ["scan", "192.168.1.100", "--ports", "80,443,502"])
        assert result.exit_code == 0
        mock_scanner.assert_called_once()
        args, kwargs = mock_scanner.call_args
        assert args[0] == "192.168.1.100"
        assert 80 in args[1]
        assert 443 in args[1]
        assert 502 in args[1]


class TestStatusCommand:
    """Test status command."""

    @pytest.fixture
    def runner(self):
        """Create CLI test runner."""
        return CliRunner()

    @pytest.fixture
    def mock_pool(self):
        """Mock connection pool."""
        with patch("bifrost.cli.pool") as mock:
            pool = MagicMock()
            pool.get_stats.return_value = {
                "total_connections": 5,
                "active_connections": 2,
                "idle_connections": 3,
                "connections": [
                    {
                        "id": "conn1",
                        "protocol": "modbus",
                        "host": "192.168.1.100",
                        "state": "connected",
                        "uptime": 3600,
                    },
                    {
                        "id": "conn2",
                        "protocol": "opcua",
                        "host": "192.168.1.101",
                        "state": "disconnected",
                        "uptime": 0,
                    },
                ],
            }
            mock.return_value = pool
            yield pool

    def test_status_command(self, runner, mock_pool):
        """Test status command."""
        result = runner.invoke(app, ["status"])
        assert result.exit_code == 0
        assert "Active Connections: 2" in result.stdout
        assert "192.168.1.100" in result.stdout
        assert "modbus" in result.stdout

    def test_status_json_format(self, runner, mock_pool):
        """Test status command with JSON output."""
        result = runner.invoke(app, ["status", "--format", "json"])
        assert result.exit_code == 0
        assert '"total_connections": 5' in result.stdout
        assert '"protocol": "modbus"' in result.stdout


class TestAssignIPCommand:
    """Test assign-ip command."""

    @pytest.fixture
    def runner(self):
        """Create CLI test runner."""
        return CliRunner()

    @pytest.fixture
    def mock_bootp(self):
        """Mock BootP server."""
        with patch("bifrost.cli.BootPServer") as mock:
            server = MagicMock()
            server.start = AsyncMock()
            server.stop = AsyncMock()
            server.assign_ip = AsyncMock(return_value=True)
            mock.return_value = server
            yield server

    def test_assign_ip_command(self, runner, mock_bootp):
        """Test assign-ip command."""
        result = runner.invoke(
            app,
            [
                "assign-ip",
                "--mac",
                "00:11:22:33:44:55",
                "--ip",
                "192.168.1.200",
                "--timeout",
                "5",
            ],
        )
        assert result.exit_code == 0
        mock_bootp.assign_ip.assert_called_once()
        args, kwargs = mock_bootp.assign_ip.call_args
        assert args[0] == "00:11:22:33:44:55"
        assert args[1] == "192.168.1.200"


class TestCLIErrorHandling:
    """Test CLI error handling."""

    @pytest.fixture
    def runner(self):
        """Create CLI test runner."""
        return CliRunner()

    def test_keyboard_interrupt_handling(self, runner):
        """Test handling of keyboard interrupt."""
        with patch("bifrost.cli.discovery.discover_devices") as mock:
            mock.side_effect = KeyboardInterrupt()
            result = runner.invoke(app, ["discover"])
            # Should handle gracefully
            assert "Cancelled" in result.stdout or result.exit_code == 0

    def test_connection_error_handling(self, runner):
        """Test handling of connection errors."""
        with patch("bifrost.modbus.ModbusTCPConnection") as mock:

            async def raise_error(*args, **kwargs):
                raise ConnectionError("Connection failed")

            mock.side_effect = raise_error
            result = runner.invoke(app, ["connect", "modbus://192.168.1.100:502"])
            assert result.exit_code != 0
            assert "Connection failed" in result.stdout

"""Tests for CLI commands."""

from unittest.mock import AsyncMock, patch

import pytest
from typer.testing import CliRunner

from bifrost.cli import app


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
        assert "Bifrost - Industrial IoT Framework" in result.stdout
        assert "discover" in result.stdout


class TestDiscoverCommand:
    """Test discover command."""

    @pytest.fixture
    def runner(self):
        """Create CLI test runner."""
        return CliRunner()

    @pytest.fixture
    def mock_discover_devices(self):
        """Mock discover_devices function."""
        with patch("bifrost.cli.discover_devices") as mock:
            mock.return_value = AsyncMock(
                return_value=[
                    {
                        "host": "192.168.1.100",
                        "port": 502,
                        "protocol": "modbus.tcp",
                        "device_type": "PLC",
                    }
                ]
            )
            yield mock

    def test_discover_default(self, runner, mock_discover_devices):
        """Test discover command with default options."""
        result = runner.invoke(app, ["discover"])
        assert result.exit_code == 0
        assert "192.168.1.100" in result.stdout
        assert "modbus.tcp" in result.stdout
        assert "PLC" in result.stdout


class TestCLIErrorHandling:
    """Test CLI error handling."""

    @pytest.fixture
    def runner(self):
        """Create CLI test runner."""
        return CliRunner()

    def test_keyboard_interrupt_handling(self, runner):
        """Test handling of keyboard interrupt."""
        with patch("bifrost.cli.discover_devices") as mock:
            mock.side_effect = KeyboardInterrupt()
            result = runner.invoke(app, ["discover"])
            # Should handle gracefully
            assert "Cancelled" in result.stdout or result.exit_code == 0

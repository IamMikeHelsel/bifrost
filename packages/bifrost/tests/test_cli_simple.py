"""Simple tests for CLI commands that verify basic functionality."""

import pytest
from bifrost.cli import app
from typer.testing import CliRunner


class TestCLIBasics:
    """Test basic CLI functionality."""

    @pytest.fixture
    def runner(self):
        """Create CLI test runner."""
        return CliRunner()

    def test_app_exists(self):
        """Test that the CLI app exists."""
        assert app is not None
        assert hasattr(app, "registered_commands")

    def test_help_command(self, runner):
        """Test help command."""
        result = runner.invoke(app, ["--help"])
        assert result.exit_code == 0
        assert "bifrost" in result.stdout.lower()

    def test_discover_command_exists(self, runner):
        """Test that discover command exists."""
        result = runner.invoke(app, ["discover", "--help"])
        assert result.exit_code == 0
        assert "discover" in result.stdout.lower()

    def test_connect_command_exists(self, runner):
        """Test that connect command exists."""
        result = runner.invoke(app, ["connect", "--help"])
        assert result.exit_code == 0
        assert "connect" in result.stdout.lower()

    def test_scan_command_exists(self, runner):
        """Test that scan command exists."""
        result = runner.invoke(app, ["scan", "--help"])
        assert result.exit_code == 0
        assert "scan" in result.stdout.lower()

    def test_status_command_exists(self, runner):
        """Test that status command exists."""
        result = runner.invoke(app, ["status", "--help"])
        assert result.exit_code == 0
        assert "status" in result.stdout.lower()

    def test_assign_ip_command_exists(self, runner):
        """Test that assign-ip command exists."""
        result = runner.invoke(app, ["assign-ip", "--help"])
        assert result.exit_code == 0
        assert "assign" in result.stdout.lower()


class TestCLIValidation:
    """Test CLI input validation."""

    @pytest.fixture
    def runner(self):
        """Create CLI test runner."""
        return CliRunner()

    def test_connect_requires_host(self, runner):
        """Test that connect command requires a host."""
        result = runner.invoke(app, ["connect"])
        # Should either require host argument or show help
        assert result.exit_code != 0 or "help" in result.stdout.lower()

    def test_assign_ip_requires_params(self, runner):
        """Test that assign-ip requires parameters."""
        result = runner.invoke(app, ["assign-ip"])
        # Should require MAC and IP parameters
        assert result.exit_code != 0

"""Simple tests for CLI commands that verify basic functionality."""

import pytest
from typer.testing import CliRunner

from bifrost.cli import app


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

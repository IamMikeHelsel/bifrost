"""Tests for main bifrost package initialization."""

import pytest
from bifrost import __version__, BaseConnection, DataPoint, DataType


class TestBifrostInit:
    """Test main bifrost package initialization."""
    
    def test_version_exists(self):
        assert __version__ is not None
        assert isinstance(__version__, str)
        assert len(__version__) > 0
    
    def test_core_imports(self):
        """Test that core components are properly imported."""
        # Test that we can import core classes
        assert BaseConnection is not None
        assert DataPoint is not None
        assert DataType is not None
    
    def test_smart_import_error(self):
        """Test that missing optional dependencies give helpful errors."""
        with pytest.raises(ImportError) as exc_info:
            # This should trigger the smart import error
            from bifrost import OPCUAClient
            
        error_msg = str(exc_info.value)
        assert "OPCUAClient" in error_msg
        assert "bifrost[opcua]" in error_msg or "bifrost-opcua" in error_msg
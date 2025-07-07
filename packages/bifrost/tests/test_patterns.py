"""Tests for pattern system functionality."""

import asyncio
import json
import tempfile
from pathlib import Path

import pytest
from bifrost_core.base import DeviceInfo
from bifrost_core.patterns import (
    DevicePattern,
    PatternDatabase,
    ProtocolSpec,
    VersionRange,
)
from bifrost_core.pattern_storage import PatternManager, PatternStorage


class TestPatternModels:
    """Test pattern data models."""

    def test_version_range_exact_match(self):
        """Test exact version matching."""
        version_range = VersionRange(exact_version="1.2.3")
        assert version_range.matches("1.2.3")
        assert not version_range.matches("1.2.4")

    def test_version_range_range_match(self):
        """Test version range matching."""
        version_range = VersionRange(min_version="1.0.0", max_version="2.0.0")
        assert version_range.matches("1.5.0")
        assert version_range.matches("1.0.0")
        assert version_range.matches("2.0.0")
        assert not version_range.matches("0.9.0")
        assert not version_range.matches("2.1.0")

    def test_device_pattern_compatibility(self):
        """Test device pattern compatibility checking."""
        pattern = DevicePattern(
            pattern_id="test_pattern",
            manufacturer_id="TestMfg",
            product_family="TestFamily",
            model_number="Model123",
            protocol_variant=ProtocolSpec(protocol="modbus.tcp", port=502),
            firmware_version_range=VersionRange(min_version="1.0", max_version="2.0")
        )

        # Compatible device data
        compatible_data = {
            "protocol": "modbus.tcp",
            "firmware_version": "1.5"
        }
        assert pattern.is_compatible(compatible_data)

        # Incompatible protocol
        incompatible_protocol = {
            "protocol": "ethernet_ip",
            "firmware_version": "1.5"
        }
        assert not pattern.is_compatible(incompatible_protocol)

        # Incompatible firmware version
        incompatible_firmware = {
            "protocol": "modbus.tcp", 
            "firmware_version": "3.0"
        }
        assert not pattern.is_compatible(incompatible_firmware)

    def test_pattern_confidence_calculation(self):
        """Test pattern confidence calculation."""
        pattern = DevicePattern(
            pattern_id="test_pattern",
            manufacturer_id="TestMfg", 
            product_family="TestFamily",
            model_number="Model123",
            protocol_variant=ProtocolSpec(protocol="modbus.tcp"),
            pattern_confidence=0.8
        )

        # Perfect match
        perfect_match = {
            "manufacturer": "TestMfg",
            "model": "Model123",
            "protocol": "modbus.tcp"
        }
        confidence = pattern.calculate_match_confidence(perfect_match)
        assert confidence > 0.8

        # Partial match
        partial_match = {
            "protocol": "modbus.tcp"
        }
        confidence = pattern.calculate_match_confidence(partial_match)
        assert 0.0 < confidence < 1.0

    def test_pattern_database_operations(self):
        """Test pattern database operations."""
        db = PatternDatabase()
        
        pattern = DevicePattern(
            pattern_id="test_pattern",
            manufacturer_id="TestMfg",
            product_family="TestFamily", 
            model_number="Model123",
            protocol_variant=ProtocolSpec(protocol="modbus.tcp")
        )

        # Add pattern
        db.add_pattern(pattern)
        assert db.count_patterns() == 1
        assert db.get_pattern("test_pattern") == pattern

        # Find patterns
        device_data = {"protocol": "modbus.tcp", "manufacturer": "TestMfg"}
        matches = db.find_patterns(device_data, min_confidence=0.1)
        assert len(matches) == 1
        assert matches[0].pattern == pattern

        # Remove pattern
        assert db.remove_pattern("test_pattern")
        assert db.count_patterns() == 0
        assert not db.remove_pattern("nonexistent")


class TestPatternStorage:
    """Test pattern storage functionality."""

    @pytest.fixture
    def temp_storage_path(self):
        """Create temporary storage path."""
        with tempfile.NamedTemporaryFile(suffix='.json', delete=False) as f:
            temp_path = Path(f.name)
        temp_path.unlink()  # Remove the file so we can test creation
        yield temp_path
        if temp_path.exists():
            temp_path.unlink()

    @pytest.mark.asyncio
    async def test_pattern_storage_basic_operations(self, temp_storage_path):
        """Test basic pattern storage operations."""
        storage = PatternStorage(temp_storage_path)
        
        pattern = DevicePattern(
            pattern_id="test_storage_pattern",
            manufacturer_id="StorageMfg",
            product_family="StorageFamily",
            model_number="Storage123",
            protocol_variant=ProtocolSpec(protocol="modbus.tcp")
        )

        # Add pattern
        await storage.add_pattern(pattern)
        
        # Verify storage file was created
        assert temp_storage_path.exists()
        
        # Retrieve pattern
        retrieved = await storage.get_pattern("test_storage_pattern")
        assert retrieved is not None
        assert retrieved.manufacturer_id == "StorageMfg"

        # Find patterns
        device_data = {"protocol": "modbus.tcp"}
        matches = await storage.find_patterns(device_data, min_confidence=0.1)
        assert len(matches) == 1

        # Remove pattern
        assert await storage.remove_pattern("test_storage_pattern")
        assert await storage.get_pattern("test_storage_pattern") is None

    @pytest.mark.asyncio
    async def test_pattern_usage_tracking(self, temp_storage_path):
        """Test pattern usage tracking."""
        storage = PatternStorage(temp_storage_path)
        
        pattern = DevicePattern(
            pattern_id="usage_test_pattern",
            manufacturer_id="UsageMfg",
            product_family="UsageFamily", 
            model_number="Usage123",
            protocol_variant=ProtocolSpec(protocol="modbus.tcp"),
            pattern_confidence=0.5,
            usage_count=0
        )

        await storage.add_pattern(pattern)
        
        # Update usage with success
        await storage.update_pattern_usage("usage_test_pattern", success=True)
        
        updated = await storage.get_pattern("usage_test_pattern")
        assert updated is not None
        assert updated.usage_count == 1
        assert updated.pattern_confidence > 0.5  # Should increase slightly

        # Update usage with failure
        original_confidence = updated.pattern_confidence
        await storage.update_pattern_usage("usage_test_pattern", success=False)
        
        updated_again = await storage.get_pattern("usage_test_pattern")
        assert updated_again is not None
        assert updated_again.usage_count == 2
        assert updated_again.pattern_confidence < original_confidence  # Should decrease

    @pytest.mark.asyncio
    async def test_pattern_persistence(self, temp_storage_path):
        """Test that patterns persist across storage instances."""
        # Create first storage instance and add pattern
        storage1 = PatternStorage(temp_storage_path)
        
        pattern = DevicePattern(
            pattern_id="persistence_test",
            manufacturer_id="PersistMfg",
            product_family="PersistFamily",
            model_number="Persist123", 
            protocol_variant=ProtocolSpec(protocol="modbus.tcp")
        )
        
        await storage1.add_pattern(pattern)
        
        # Create second storage instance and verify pattern exists
        storage2 = PatternStorage(temp_storage_path)
        retrieved = await storage2.get_pattern("persistence_test")
        
        assert retrieved is not None
        assert retrieved.manufacturer_id == "PersistMfg"


class TestPatternManager:
    """Test pattern manager functionality."""

    @pytest.fixture
    def temp_storage_path(self):
        """Create temporary storage path."""
        with tempfile.NamedTemporaryFile(suffix='.json', delete=False) as f:
            temp_path = Path(f.name)
        temp_path.unlink()
        yield temp_path
        if temp_path.exists():
            temp_path.unlink()

    @pytest.mark.asyncio
    async def test_pattern_learning(self, temp_storage_path):
        """Test automatic pattern learning."""
        manager = PatternManager(temp_storage_path)
        
        device_data = {
            "manufacturer": "LearnMfg",
            "model": "Learn123",
            "protocol": "modbus.tcp",
            "port": 502,
            "firmware_version": "1.0.0"
        }
        
        # Learn pattern from device
        pattern = await manager.learn_pattern_from_device(device_data)
        
        assert pattern.manufacturer_id == "LearnMfg"
        assert pattern.model_number == "Learn123"
        assert pattern.protocol_variant.protocol == "modbus.tcp"

        # Verify pattern was stored
        retrieved = await manager.storage.get_pattern(pattern.pattern_id)
        assert retrieved is not None

    @pytest.mark.asyncio 
    async def test_pattern_matching_and_discovery(self, temp_storage_path):
        """Test pattern matching and discovery workflow."""
        manager = PatternManager(temp_storage_path)
        
        # Create and store a pattern
        pattern = DevicePattern(
            pattern_id="match_test_pattern",
            manufacturer_id="MatchMfg",
            product_family="MatchFamily",
            model_number="Match123",
            protocol_variant=ProtocolSpec(protocol="modbus.tcp"),
            pattern_confidence=0.9
        )
        await manager.storage.add_pattern(pattern)
        
        # Test pattern matching
        device_data = {
            "manufacturer": "MatchMfg",
            "model": "Match123", 
            "protocol": "modbus.tcp"
        }
        
        match_result = await manager.discover_and_match_patterns(
            device_data, 
            min_confidence=0.7
        )
        
        assert match_result is not None
        assert match_result.pattern.pattern_id == "match_test_pattern"
        assert match_result.confidence > 0.7

        # Verify usage count was updated
        updated_pattern = await manager.storage.get_pattern("match_test_pattern")
        assert updated_pattern is not None
        assert updated_pattern.usage_count == 1

    @pytest.mark.asyncio
    async def test_pattern_statistics(self, temp_storage_path):
        """Test pattern statistics generation."""
        manager = PatternManager(temp_storage_path)
        
        # Add some test patterns
        for i in range(3):
            pattern = DevicePattern(
                pattern_id=f"stats_test_{i}",
                manufacturer_id=f"StatsMfg{i}",
                product_family="StatsFamily",
                model_number=f"Stats{i}",
                protocol_variant=ProtocolSpec(protocol="modbus.tcp"),
                pattern_confidence=0.8,
                usage_count=i * 10
            )
            await manager.storage.add_pattern(pattern)
        
        stats = await manager.get_pattern_statistics()
        
        assert stats['total_patterns'] == 3
        assert stats['average_confidence'] == 0.8
        assert stats['total_usage'] == 30  # 0 + 10 + 20
        assert stats['most_used_pattern']['usage_count'] == 20
        assert 'modbus.tcp' in stats['protocols']

    @pytest.mark.asyncio
    async def test_pattern_import_export(self, temp_storage_path):
        """Test pattern import and export functionality."""
        manager = PatternManager(temp_storage_path)
        
        # Add test pattern
        pattern = DevicePattern(
            pattern_id="export_test",
            manufacturer_id="ExportMfg",
            product_family="ExportFamily",
            model_number="Export123",
            protocol_variant=ProtocolSpec(protocol="modbus.tcp")
        )
        await manager.storage.add_pattern(pattern)
        
        # Test export
        with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
            export_path = Path(f.name)
        
        try:
            await manager.export_patterns(export_path)
            assert export_path.exists()
            
            # Verify export content
            with open(export_path, 'r') as f:
                export_data = json.load(f)
            assert 'patterns' in export_data
            assert export_data['pattern_count'] == 1
            
            # Test import into new manager
            with tempfile.NamedTemporaryFile(suffix='.json', delete=False) as f:
                new_storage_path = Path(f.name)
            new_storage_path.unlink()
            
            try:
                new_manager = PatternManager(new_storage_path)
                imported_count = await new_manager.import_patterns(export_path)
                
                assert imported_count == 1
                imported_pattern = await new_manager.storage.get_pattern("export_test")
                assert imported_pattern is not None
                assert imported_pattern.manufacturer_id == "ExportMfg"
                
            finally:
                if new_storage_path.exists():
                    new_storage_path.unlink()
                    
        finally:
            export_path.unlink()


if __name__ == "__main__":
    # Run a simple demo test
    async def demo_test():
        print("🧪 Running Pattern System Demo Test")
        
        # Test basic pattern creation and matching
        pattern = DevicePattern(
            pattern_id="demo_pattern",
            manufacturer_id="DemoMfg",
            product_family="DemoFamily",
            model_number="Demo123",
            protocol_variant=ProtocolSpec(protocol="modbus.tcp")
        )
        
        device_data = {
            "manufacturer": "DemoMfg",
            "model": "Demo123",
            "protocol": "modbus.tcp"
        }
        
        confidence = pattern.calculate_match_confidence(device_data)
        print(f"   Pattern match confidence: {confidence:.2f}")
        
        # Test pattern storage
        with tempfile.NamedTemporaryFile(suffix='.json', delete=False) as f:
            temp_path = Path(f.name)
        temp_path.unlink()
        
        try:
            manager = PatternManager(temp_path)
            await manager.storage.add_pattern(pattern)
            
            match_result = await manager.discover_and_match_patterns(device_data)
            if match_result:
                print(f"   Pattern matched: {match_result.pattern.pattern_id}")
                print(f"   Match confidence: {match_result.confidence:.2f}")
            else:
                print("   No pattern match found")
                
            stats = await manager.get_pattern_statistics()
            print(f"   Total patterns: {stats['total_patterns']}")
            
        finally:
            if temp_path.exists():
                temp_path.unlink()
        
        print("✅ Demo test completed successfully!")
    
    asyncio.run(demo_test())
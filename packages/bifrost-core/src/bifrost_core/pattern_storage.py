"""Pattern storage and management for Bifrost.

This module provides storage, retrieval, and management capabilities for
device patterns, including local file-based storage and in-memory caching.
"""

import json
import time
from pathlib import Path
from typing import Any

from bifrost_core.patterns import (
    DevicePattern,
    PatternDatabase,
    PatternMatchResult,
)
from bifrost_core.typing import JsonDict


class PatternStorage:
    """Local file-based storage for device patterns."""

    def __init__(self, storage_path: str | Path = "patterns.json"):
        """Initialize pattern storage.
        
        Args:
            storage_path: Path to the pattern storage file
        """
        self.storage_path = Path(storage_path)
        self._database: PatternDatabase | None = None
        self._last_modified: float = 0

    async def load_patterns(self) -> PatternDatabase:
        """Load patterns from storage.
        
        Returns:
            PatternDatabase instance with loaded patterns
        """
        if not self.storage_path.exists():
            # Create empty database if file doesn't exist
            self._database = PatternDatabase()
            await self.save_patterns()
            return self._database

        # Check if we need to reload from disk
        current_modified = self.storage_path.stat().st_mtime
        if (self._database is None or 
            current_modified > self._last_modified):
            
            try:
                with open(self.storage_path, 'r', encoding='utf-8') as f:
                    data = json.load(f)
                self._database = PatternDatabase.model_validate(data)
                self._last_modified = current_modified
            except (json.JSONDecodeError, ValueError) as e:
                # If file is corrupted, create new database
                print(f"Warning: Pattern file corrupted, creating new database: {e}")
                self._database = PatternDatabase()
                await self.save_patterns()

        return self._database

    async def save_patterns(self) -> None:
        """Save patterns to storage."""
        if self._database is None:
            return

        # Update last modified timestamp
        self._database.last_updated = int(time.time() * 1_000_000_000)
        
        # Ensure directory exists
        self.storage_path.parent.mkdir(parents=True, exist_ok=True)
        
        # Write to temporary file first, then rename for atomicity
        temp_path = self.storage_path.with_suffix('.tmp')
        try:
            with open(temp_path, 'w', encoding='utf-8') as f:
                json.dump(
                    self._database.model_dump(),
                    f,
                    indent=2,
                    ensure_ascii=False
                )
            temp_path.rename(self.storage_path)
            self._last_modified = self.storage_path.stat().st_mtime
        except Exception:
            # Clean up temporary file on error
            if temp_path.exists():
                temp_path.unlink()
            raise

    async def add_pattern(self, pattern: DevicePattern) -> None:
        """Add a pattern to storage.
        
        Args:
            pattern: Pattern to add
        """
        database = await self.load_patterns()
        database.add_pattern(pattern)
        await self.save_patterns()

    async def remove_pattern(self, pattern_id: str) -> bool:
        """Remove a pattern from storage.
        
        Args:
            pattern_id: ID of pattern to remove
            
        Returns:
            True if pattern was removed, False if not found
        """
        database = await self.load_patterns()
        removed = database.remove_pattern(pattern_id)
        if removed:
            await self.save_patterns()
        return removed

    async def get_pattern(self, pattern_id: str) -> DevicePattern | None:
        """Get a pattern by ID.
        
        Args:
            pattern_id: Pattern ID to retrieve
            
        Returns:
            Pattern if found, None otherwise
        """
        database = await self.load_patterns()
        return database.get_pattern(pattern_id)

    async def find_patterns(
        self,
        device_data: JsonDict,
        min_confidence: float = 0.5
    ) -> list[PatternMatchResult]:
        """Find matching patterns for device data.
        
        Args:
            device_data: Device data to match against
            min_confidence: Minimum confidence threshold
            
        Returns:
            List of pattern matches sorted by confidence
        """
        database = await self.load_patterns()
        return database.find_patterns(device_data, min_confidence)

    async def update_pattern_usage(self, pattern_id: str, success: bool = True) -> None:
        """Update pattern usage statistics.
        
        Args:
            pattern_id: Pattern ID to update
            success: Whether the pattern usage was successful
        """
        pattern = await self.get_pattern(pattern_id)
        if pattern:
            pattern.usage_count += 1
            pattern.last_verified = int(time.time() * 1_000_000_000)
            
            # Update confidence based on success
            if success:
                # Increase confidence slightly for successful use
                pattern.pattern_confidence = min(
                    1.0, 
                    pattern.pattern_confidence + 0.01
                )
            else:
                # Decrease confidence for failed use
                pattern.pattern_confidence = max(
                    0.0,
                    pattern.pattern_confidence - 0.05
                )
            
            await self.save_patterns()


class PatternManager:
    """High-level pattern management interface."""

    def __init__(self, storage_path: str | Path = "patterns.json"):
        """Initialize pattern manager.
        
        Args:
            storage_path: Path to pattern storage file
        """
        self.storage = PatternStorage(storage_path)

    async def discover_and_match_patterns(
        self,
        device_data: JsonDict,
        min_confidence: float = 0.7
    ) -> PatternMatchResult | None:
        """Attempt to match device against known patterns.
        
        This implements the fast path optimization where known devices
        can be instantly configured with optimal settings.
        
        Args:
            device_data: Device discovery data
            min_confidence: Minimum confidence for pattern match
            
        Returns:
            Best pattern match if found, None otherwise
        """
        matches = await self.storage.find_patterns(device_data, min_confidence)
        
        if matches:
            best_match = matches[0]  # Highest confidence
            
            # Update usage statistics
            await self.storage.update_pattern_usage(
                best_match.pattern.pattern_id,
                success=True
            )
            
            return best_match
        
        return None

    async def learn_pattern_from_device(
        self,
        device_data: JsonDict,
        communication_data: JsonDict | None = None
    ) -> DevicePattern:
        """Learn a new pattern from device interaction.
        
        This creates a new pattern based on successful device communication
        and adds it to the pattern database for future use.
        
        Args:
            device_data: Device discovery and identification data
            communication_data: Successful communication patterns
            
        Returns:
            Newly created device pattern
        """
        from bifrost_core.patterns import (
            CommunicationProfile,
            DiscoverySignature,
            ProtocolSpec,
            VersionRange,
        )
        
        # Generate pattern ID
        pattern_id = f"{device_data.get('manufacturer', 'unknown')}_{device_data.get('model', 'unknown')}_{device_data.get('protocol', 'unknown')}"
        pattern_id = pattern_id.lower().replace(' ', '_').replace('.', '_')
        
        # Create pattern from device data
        pattern = DevicePattern(
            pattern_id=pattern_id,
            manufacturer_id=device_data.get('manufacturer', 'unknown'),
            product_family=device_data.get('product_family', 'unknown'),
            model_number=device_data.get('model', 'unknown'),
            firmware_version_range=VersionRange(
                exact_version=device_data.get('firmware_version')
            ),
            protocol_variant=ProtocolSpec(
                protocol=device_data.get('protocol', 'unknown'),
                port=device_data.get('port')
            ),
            discovery_signature=DiscoverySignature(),
            communication_profile=CommunicationProfile(),
            pattern_confidence=0.5,  # Start with medium confidence
            contributor_reputation=0.8,  # Assume good reputation for local patterns
            metadata={
                'learned_from': device_data,
                'learning_timestamp': int(time.time() * 1_000_000_000)
            }
        )
        
        # Add communication data if available
        if communication_data:
            pattern.metadata['communication_data'] = communication_data
        
        # Store the pattern
        await self.storage.add_pattern(pattern)
        
        return pattern

    async def get_pattern_statistics(self) -> JsonDict:
        """Get statistics about stored patterns.
        
        Returns:
            Dictionary with pattern statistics
        """
        database = await self.storage.load_patterns()
        
        patterns = list(database.patterns.values())
        if not patterns:
            return {
                'total_patterns': 0,
                'average_confidence': 0.0,
                'most_used_pattern': None,
                'protocols': []
            }
        
        total_usage = sum(p.usage_count for p in patterns)
        avg_confidence = sum(p.pattern_confidence for p in patterns) / len(patterns)
        most_used = max(patterns, key=lambda p: p.usage_count)
        protocols = list(set(p.protocol_variant.protocol for p in patterns))
        
        return {
            'total_patterns': len(patterns),
            'total_usage': total_usage,
            'average_confidence': avg_confidence,
            'most_used_pattern': {
                'id': most_used.pattern_id,
                'usage_count': most_used.usage_count,
                'confidence': most_used.pattern_confidence
            },
            'protocols': sorted(protocols),
            'database_version': database.version,
            'last_updated': database.last_updated
        }

    async def export_patterns(self, export_path: str | Path) -> None:
        """Export patterns to a file.
        
        Args:
            export_path: Path to export file
        """
        database = await self.storage.load_patterns()
        export_data = {
            'export_timestamp': int(time.time() * 1_000_000_000),
            'pattern_count': len(database.patterns),
            'patterns': database.model_dump()
        }
        
        with open(export_path, 'w', encoding='utf-8') as f:
            json.dump(export_data, f, indent=2, ensure_ascii=False)

    async def import_patterns(
        self,
        import_path: str | Path,
        overwrite: bool = False
    ) -> int:
        """Import patterns from a file.
        
        Args:
            import_path: Path to import file
            overwrite: Whether to overwrite existing patterns
            
        Returns:
            Number of patterns imported
        """
        with open(import_path, 'r', encoding='utf-8') as f:
            import_data = json.load(f)
        
        if 'patterns' not in import_data:
            raise ValueError("Invalid import file format")
        
        imported_db = PatternDatabase.model_validate(import_data['patterns'])
        current_db = await self.storage.load_patterns()
        
        imported_count = 0
        for pattern_id, pattern in imported_db.patterns.items():
            if pattern_id not in current_db.patterns or overwrite:
                current_db.add_pattern(pattern)
                imported_count += 1
        
        await self.storage.save_patterns()
        return imported_count
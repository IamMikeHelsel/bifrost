"""Enhanced device discovery with pattern recognition for Bifrost.

This module extends the basic discovery system with pattern recognition
capabilities, enabling fast device identification and optimal configuration
based on stored patterns.
"""

import asyncio
import time
from collections.abc import AsyncGenerator, Sequence
from pathlib import Path

from bifrost_core.base import DeviceInfo
from bifrost_core.pattern_storage import PatternManager
from bifrost_core.patterns import PatternMatchResult
from bifrost_core.typing import JsonDict

from .discovery import DiscoveryConfig, discover_devices as basic_discover_devices


class PatternDiscoveryConfig(DiscoveryConfig):
    """Configuration for pattern-enhanced discovery.
    
    Extends the basic discovery configuration with pattern-specific settings.
    """

    def __init__(
        self,
        network_range: str = "192.168.1.0/24",
        timeout: float = 2.0,
        max_concurrent: int = 50,
        protocols: Sequence[str] = ("modbus", "cip", "bootp"),
        pattern_storage_path: str | Path = "patterns.json",
        pattern_confidence_threshold: float = 0.7,
        enable_pattern_learning: bool = True,
    ):
        """Initialize pattern discovery configuration.
        
        Args:
            network_range: IP network range to scan (default: 192.168.1.0/24).
            timeout: Timeout in seconds for connections (default: 2.0).
            max_concurrent: Max concurrent connections (default: 50).
            protocols: Discovery protocols to use (default: all).
            pattern_storage_path: Path to pattern storage file.
            pattern_confidence_threshold: Minimum confidence for pattern match.
            enable_pattern_learning: Whether to learn new patterns automatically.
        """
        super().__init__(network_range, timeout, max_concurrent, protocols)
        self.pattern_storage_path = pattern_storage_path
        self.pattern_confidence_threshold = pattern_confidence_threshold
        self.enable_pattern_learning = enable_pattern_learning


class EnhancedDeviceInfo(DeviceInfo):
    """Enhanced device information with pattern matching data."""

    pattern_match: PatternMatchResult | None = None
    discovery_path: str = "unknown"  # "fast_path" or "slow_path"
    pattern_applied: bool = False


async def discover_devices_with_patterns(
    config: PatternDiscoveryConfig | None = None,
    protocols: Sequence[str] | None = None,
) -> AsyncGenerator[EnhancedDeviceInfo, None]:
    """Discover devices using pattern-enhanced discovery.
    
    This function implements the enhanced discovery workflow:
    1. Device Discovery
    2. Fingerprint Generation 
    3. Pattern Database Query
    4. Pattern Match Found? 
       → Yes: Apply Pattern (Fast Path)
       → No: Traditional Discovery (Slow Path)
    5. Validate Configuration
    6. Update Pattern Database
    
    Args:
        config: Pattern discovery configuration. If None, uses defaults.
        protocols: List of protocols to use. If None, uses config.protocols.
        
    Yields:
        EnhancedDeviceInfo objects for discovered devices with pattern data.
    """
    if config is None:
        config = PatternDiscoveryConfig()
    
    # Initialize pattern manager
    pattern_manager = PatternManager(config.pattern_storage_path)
    
    # Track statistics
    fast_path_count = 0
    slow_path_count = 0
    pattern_matches = 0
    
    print(f"🔍 Starting pattern-enhanced discovery on {config.network_range}")
    print(f"📊 Pattern confidence threshold: {config.pattern_confidence_threshold}")
    
    # Run basic discovery to find devices
    async for device in basic_discover_devices(
        DiscoveryConfig(
            network_range=config.network_range,
            timeout=config.timeout,
            max_concurrent=config.max_concurrent,
            protocols=protocols or config.protocols
        )
    ):
        enhanced_device = EnhancedDeviceInfo(**device.model_dump())
        
        # Step 2: Generate device fingerprint
        device_fingerprint = _generate_device_fingerprint(device)
        
        # Step 3 & 4: Query pattern database
        pattern_match = await pattern_manager.discover_and_match_patterns(
            device_fingerprint,
            min_confidence=config.pattern_confidence_threshold
        )
        
        if pattern_match:
            # Fast Path: Apply known pattern
            enhanced_device.pattern_match = pattern_match
            enhanced_device.discovery_path = "fast_path"
            enhanced_device.pattern_applied = True
            
            # Apply pattern optimizations to device info
            _apply_pattern_optimizations(enhanced_device, pattern_match)
            
            fast_path_count += 1
            pattern_matches += 1
            
            print(f"⚡ Fast path: {device.host} matched pattern {pattern_match.pattern.pattern_id} (confidence: {pattern_match.confidence:.2f})")
            
        else:
            # Slow Path: Traditional discovery
            enhanced_device.discovery_path = "slow_path"
            slow_path_count += 1
            
            print(f"🐌 Slow path: {device.host} - no pattern match found")
            
            # Step 6: Learn new pattern if enabled
            if config.enable_pattern_learning:
                await _learn_pattern_from_device(
                    pattern_manager,
                    device_fingerprint,
                    enhanced_device
                )
        
        yield enhanced_device
    
    # Print final statistics
    total_devices = fast_path_count + slow_path_count
    if total_devices > 0:
        fast_path_percentage = (fast_path_count / total_devices) * 100
        print(f"\n📈 Discovery Statistics:")
        print(f"   Total devices: {total_devices}")
        print(f"   Fast path: {fast_path_count} ({fast_path_percentage:.1f}%)")
        print(f"   Slow path: {slow_path_count}")
        print(f"   Pattern matches: {pattern_matches}")


def _generate_device_fingerprint(device: DeviceInfo) -> JsonDict:
    """Generate a device fingerprint for pattern matching.
    
    Args:
        device: Device information from discovery
        
    Returns:
        Device fingerprint data for pattern matching
    """
    fingerprint = {
        'protocol': device.protocol,
        'host': device.host,
        'port': device.port,
        'device_type': device.device_type,
        'discovery_method': device.discovery_method,
        'confidence': device.confidence,
    }
    
    # Add optional fields if available
    optional_fields = [
        'manufacturer', 'model', 'firmware_version', 'serial_number',
        'vendor_id', 'product_code', 'mac_address'
    ]
    
    for field in optional_fields:
        value = getattr(device, field, None)
        if value is not None:
            fingerprint[field] = value
    
    # Add metadata
    fingerprint['metadata'] = device.metadata
    
    return fingerprint


def _apply_pattern_optimizations(
    device: EnhancedDeviceInfo,
    pattern_match: PatternMatchResult
) -> None:
    """Apply pattern optimizations to device information.
    
    Args:
        device: Device to optimize
        pattern_match: Matched pattern with optimization data
    """
    pattern = pattern_match.pattern
    
    # Apply manufacturer and model information if missing
    if not device.manufacturer and pattern.manufacturer_id != 'unknown':
        device.manufacturer = pattern.manufacturer_id
    
    if not device.model and pattern.model_number != 'unknown':
        device.model = pattern.model_number
    
    # Update device type if pattern has more specific information
    if pattern.metadata.get('device_type'):
        device.device_type = pattern.metadata['device_type']
    
    # Enhance confidence if pattern has high confidence
    if pattern.pattern_confidence > device.confidence:
        device.confidence = min(1.0, pattern.pattern_confidence)
    
    # Add pattern-specific metadata
    device.metadata.update({
        'pattern_id': pattern.pattern_id,
        'pattern_confidence': pattern.pattern_confidence,
        'pattern_usage_count': pattern.usage_count,
        'optimal_polling_rate': pattern.communication_profile.optimal_polling_rate,
        'fast_path_applied': True
    })
    
    # Add communication optimizations
    if pattern.historical_performance:
        device.metadata.update({
            'avg_response_time': pattern.historical_performance.avg_response_time,
            'reliability_score': pattern.historical_performance.reliability_score,
            'bandwidth_requirements': pattern.historical_performance.bandwidth_requirements.model_dump()
        })


async def _learn_pattern_from_device(
    pattern_manager: PatternManager,
    device_fingerprint: JsonDict,
    device: EnhancedDeviceInfo
) -> None:
    """Learn a new pattern from discovered device.
    
    Args:
        pattern_manager: Pattern manager instance
        device_fingerprint: Device fingerprint data
        device: Enhanced device information
    """
    try:
        # Only learn patterns for devices with good identification
        if (device.confidence > 0.7 and 
            device.manufacturer and 
            device.model):
            
            pattern = await pattern_manager.learn_pattern_from_device(
                device_fingerprint
            )
            
            print(f"📚 Learned new pattern: {pattern.pattern_id} from {device.host}")
            
            # Update device with learning information
            device.metadata.update({
                'learned_pattern_id': pattern.pattern_id,
                'pattern_learning_enabled': True
            })
            
    except Exception as e:
        print(f"⚠️  Failed to learn pattern from {device.host}: {e}")


async def get_pattern_statistics(
    pattern_storage_path: str | Path = "patterns.json"
) -> JsonDict:
    """Get statistics about stored patterns.
    
    Args:
        pattern_storage_path: Path to pattern storage file
        
    Returns:
        Dictionary with pattern statistics
    """
    pattern_manager = PatternManager(pattern_storage_path)
    return await pattern_manager.get_pattern_statistics()


async def export_patterns(
    export_path: str | Path,
    pattern_storage_path: str | Path = "patterns.json"
) -> None:
    """Export patterns to a file.
    
    Args:
        export_path: Path to export file
        pattern_storage_path: Path to pattern storage file
    """
    pattern_manager = PatternManager(pattern_storage_path)
    await pattern_manager.export_patterns(export_path)


async def import_patterns(
    import_path: str | Path,
    pattern_storage_path: str | Path = "patterns.json",
    overwrite: bool = False
) -> int:
    """Import patterns from a file.
    
    Args:
        import_path: Path to import file
        pattern_storage_path: Path to pattern storage file
        overwrite: Whether to overwrite existing patterns
        
    Returns:
        Number of patterns imported
    """
    pattern_manager = PatternManager(pattern_storage_path)
    return await pattern_manager.import_patterns(import_path, overwrite)


# Example usage for demonstration
async def demo_pattern_discovery():
    """Demonstration of pattern-enhanced discovery."""
    print("🚀 Bifrost Pattern Discovery Demo")
    print("=" * 50)
    
    config = PatternDiscoveryConfig(
        network_range="192.168.1.0/24",
        pattern_confidence_threshold=0.5,
        enable_pattern_learning=True
    )
    
    device_count = 0
    fast_path_count = 0
    
    async for device in discover_devices_with_patterns(config):
        device_count += 1
        if device.discovery_path == "fast_path":
            fast_path_count += 1
        
        print(f"\n📱 Device {device_count}: {device.host}")
        print(f"   Type: {device.device_type}")
        print(f"   Protocol: {device.protocol}")
        print(f"   Discovery Path: {device.discovery_path}")
        print(f"   Pattern Applied: {device.pattern_applied}")
        
        if device.pattern_match:
            print(f"   Pattern ID: {device.pattern_match.pattern.pattern_id}")
            print(f"   Match Confidence: {device.pattern_match.confidence:.2f}")
    
    # Show pattern statistics
    stats = await get_pattern_statistics()
    print(f"\n📊 Pattern Database Statistics:")
    print(f"   Total Patterns: {stats['total_patterns']}")
    print(f"   Average Confidence: {stats['average_confidence']:.2f}")
    print(f"   Total Usage: {stats['total_usage']}")
    
    if device_count > 0:
        fast_path_percentage = (fast_path_count / device_count) * 100
        print(f"\n⚡ Fast Path Efficiency: {fast_path_percentage:.1f}%")


if __name__ == "__main__":
    # Run the demo if script is executed directly
    asyncio.run(demo_pattern_discovery())
"""CLI tool for managing device patterns in Bifrost.

This module provides command-line interface for pattern management,
including discovery with patterns, pattern statistics, import/export,
and pattern database maintenance.
"""

import asyncio
import json
import sys
import time
from pathlib import Path
from typing import Optional

try:
    import typer
    from rich.console import Console
    from rich.progress import Progress, SpinnerColumn, TextColumn
    from rich.table import Table
except ImportError:
    typer = None
    Console = None
    Progress = None
    SpinnerColumn = None
    TextColumn = None
    Table = None

from bifrost_core.pattern_storage import PatternManager
from bifrost_core.patterns import DevicePattern, ProtocolSpec

if typer is None:
    print("Warning: typer and rich not installed. CLI functionality limited.")
    print("Install with: pip install typer rich")

app = typer.Typer(
    name="patterns",
    help="Bifrost Device Pattern Management CLI",
    add_completion=False
) if typer else None

console = Console() if Console else None


def print_info(message: str) -> None:
    """Print info message."""
    if console:
        console.print(f"ℹ️  {message}", style="blue")
    else:
        print(f"INFO: {message}")


def print_success(message: str) -> None:
    """Print success message."""
    if console:
        console.print(f"✅ {message}", style="green")
    else:
        print(f"SUCCESS: {message}")


def print_error(message: str) -> None:
    """Print error message."""
    if console:
        console.print(f"❌ {message}", style="red")
    else:
        print(f"ERROR: {message}")


def print_warning(message: str) -> None:
    """Print warning message."""
    if console:
        console.print(f"⚠️  {message}", style="yellow")
    else:
        print(f"WARNING: {message}")


async def async_discover_with_patterns(
    network: str = "192.168.1.0/24",
    patterns_file: str = "patterns.json",
    confidence_threshold: float = 0.7,
    learn_patterns: bool = True,
    protocols: Optional[str] = None
) -> None:
    """Run discovery with pattern recognition."""
    from bifrost.pattern_discovery import (
        PatternDiscoveryConfig,
        discover_devices_with_patterns,
        get_pattern_statistics
    )
    
    print_info(f"Starting pattern-enhanced discovery on {network}")
    print_info(f"Pattern file: {patterns_file}")
    print_info(f"Confidence threshold: {confidence_threshold}")
    
    # Parse protocols
    protocol_list = protocols.split(',') if protocols else ['modbus', 'cip', 'bootp']
    
    config = PatternDiscoveryConfig(
        network_range=network,
        pattern_storage_path=patterns_file,
        pattern_confidence_threshold=confidence_threshold,
        enable_pattern_learning=learn_patterns,
        protocols=protocol_list
    )
    
    # Discovery statistics
    device_count = 0
    fast_path_count = 0
    slow_path_count = 0
    
    if console and Table:
        # Create results table
        table = Table(title="Device Discovery Results")
        table.add_column("Host", style="cyan", no_wrap=True)
        table.add_column("Port", style="magenta")
        table.add_column("Protocol", style="green")
        table.add_column("Type", style="yellow")
        table.add_column("Path", style="blue")
        table.add_column("Pattern", style="white")
        table.add_column("Confidence", style="red")
        
        try:
            async for device in discover_devices_with_patterns(config):
                device_count += 1
                
                if device.discovery_path == "fast_path":
                    fast_path_count += 1
                    path_style = "⚡"
                    pattern_id = device.pattern_match.pattern.pattern_id if device.pattern_match else "N/A"
                    confidence = f"{device.pattern_match.confidence:.2f}" if device.pattern_match else "N/A"
                else:
                    slow_path_count += 1
                    path_style = "🐌"
                    pattern_id = "None"
                    confidence = f"{device.confidence:.2f}"
                
                table.add_row(
                    device.host,
                    str(device.port),
                    device.protocol,
                    device.device_type or "Unknown",
                    f"{path_style} {device.discovery_path}",
                    pattern_id,
                    confidence
                )
        
        except KeyboardInterrupt:
            print_warning("Discovery interrupted by user")
        except Exception as e:
            print_error(f"Discovery failed: {e}")
            return
        
        console.print(table)
    
    else:
        # Simple text output
        try:
            async for device in discover_devices_with_patterns(config):
                device_count += 1
                
                if device.discovery_path == "fast_path":
                    fast_path_count += 1
                    print(f"⚡ {device.host}:{device.port} ({device.protocol}) - Fast path")
                    if device.pattern_match:
                        print(f"   Pattern: {device.pattern_match.pattern.pattern_id}")
                        print(f"   Confidence: {device.pattern_match.confidence:.2f}")
                else:
                    slow_path_count += 1
                    print(f"🐌 {device.host}:{device.port} ({device.protocol}) - Slow path")
        
        except KeyboardInterrupt:
            print_warning("Discovery interrupted by user")
        except Exception as e:
            print_error(f"Discovery failed: {e}")
            return
    
    # Print summary
    print()
    print_info("Discovery Summary:")
    print(f"   Total devices: {device_count}")
    print(f"   Fast path: {fast_path_count}")
    print(f"   Slow path: {slow_path_count}")
    
    if device_count > 0:
        fast_path_percentage = (fast_path_count / device_count) * 100
        print(f"   Fast path efficiency: {fast_path_percentage:.1f}%")
    
    # Show pattern statistics
    try:
        stats = await get_pattern_statistics(patterns_file)
        print()
        print_info("Pattern Database Statistics:")
        print(f"   Total patterns: {stats['total_patterns']}")
        print(f"   Average confidence: {stats['average_confidence']:.2f}")
        print(f"   Total usage: {stats['total_usage']}")
        if stats['most_used_pattern']:
            print(f"   Most used: {stats['most_used_pattern']['id']} ({stats['most_used_pattern']['usage_count']} uses)")
    except Exception as e:
        print_warning(f"Could not load pattern statistics: {e}")


async def async_show_statistics(patterns_file: str = "patterns.json") -> None:
    """Show pattern database statistics."""
    try:
        manager = PatternManager(patterns_file)
        stats = await manager.get_pattern_statistics()
        
        print_info("Pattern Database Statistics")
        print(f"   File: {patterns_file}")
        print(f"   Total patterns: {stats['total_patterns']}")
        print(f"   Average confidence: {stats['average_confidence']:.2f}")
        print(f"   Total usage: {stats['total_usage']}")
        print(f"   Protocols: {', '.join(stats['protocols'])}")
        
        if stats['most_used_pattern']:
            print(f"   Most used pattern: {stats['most_used_pattern']['id']}")
            print(f"   Usage count: {stats['most_used_pattern']['usage_count']}")
            print(f"   Confidence: {stats['most_used_pattern']['confidence']:.2f}")
        
        if stats['last_updated']:
            last_updated = time.strftime(
                '%Y-%m-%d %H:%M:%S',
                time.localtime(stats['last_updated'] / 1_000_000_000)
            )
            print(f"   Last updated: {last_updated}")
            
    except FileNotFoundError:
        print_warning(f"Pattern file not found: {patterns_file}")
    except Exception as e:
        print_error(f"Failed to load statistics: {e}")


async def async_export_patterns(
    patterns_file: str = "patterns.json",
    output_file: str = "patterns_export.json"
) -> None:
    """Export patterns to a file."""
    try:
        manager = PatternManager(patterns_file)
        await manager.export_patterns(output_file)
        print_success(f"Patterns exported to {output_file}")
    except Exception as e:
        print_error(f"Export failed: {e}")


async def async_import_patterns(
    import_file: str,
    patterns_file: str = "patterns.json",
    overwrite: bool = False
) -> None:
    """Import patterns from a file."""
    try:
        manager = PatternManager(patterns_file)
        imported_count = await manager.import_patterns(import_file, overwrite)
        print_success(f"Imported {imported_count} patterns from {import_file}")
    except Exception as e:
        print_error(f"Import failed: {e}")


async def async_create_sample_pattern(
    patterns_file: str = "patterns.json",
    manufacturer: str = "SampleMfg",
    model: str = "Sample123",
    protocol: str = "modbus.tcp",
    port: int = 502
) -> None:
    """Create a sample pattern for testing."""
    try:
        pattern = DevicePattern(
            pattern_id=f"{manufacturer.lower()}_{model.lower()}_{protocol.replace('.', '_')}",
            manufacturer_id=manufacturer,
            product_family="Sample Family",
            model_number=model,
            protocol_variant=ProtocolSpec(protocol=protocol, port=port),
            pattern_confidence=0.8,
            metadata={
                "created_by": "cli_tool",
                "created_at": int(time.time() * 1_000_000_000),
                "sample_pattern": True
            }
        )
        
        manager = PatternManager(patterns_file)
        await manager.storage.add_pattern(pattern)
        
        print_success(f"Created sample pattern: {pattern.pattern_id}")
        print(f"   Manufacturer: {manufacturer}")
        print(f"   Model: {model}")
        print(f"   Protocol: {protocol}")
        print(f"   Port: {port}")
        
    except Exception as e:
        print_error(f"Failed to create sample pattern: {e}")


# CLI Commands (only if typer is available)
if typer and app:
    
    @app.command()
    def discover(
        network: str = typer.Option("192.168.1.0/24", help="Network range to scan"),
        patterns_file: str = typer.Option("patterns.json", help="Pattern storage file"),
        confidence: float = typer.Option(0.7, help="Pattern confidence threshold"),
        learn: bool = typer.Option(True, help="Enable pattern learning"),
        protocols: Optional[str] = typer.Option(None, help="Comma-separated protocol list")
    ):
        """Run device discovery with pattern recognition."""
        asyncio.run(async_discover_with_patterns(
            network, patterns_file, confidence, learn, protocols
        ))
    
    @app.command()
    def stats(
        patterns_file: str = typer.Option("patterns.json", help="Pattern storage file")
    ):
        """Show pattern database statistics."""
        asyncio.run(async_show_statistics(patterns_file))
    
    @app.command()
    def export(
        patterns_file: str = typer.Option("patterns.json", help="Pattern storage file"),
        output: str = typer.Option("patterns_export.json", help="Export file path")
    ):
        """Export patterns to a file."""
        asyncio.run(async_export_patterns(patterns_file, output))
    
    @app.command() 
    def import_cmd(
        import_file: str = typer.Argument(..., help="File to import patterns from"),
        patterns_file: str = typer.Option("patterns.json", help="Pattern storage file"),
        overwrite: bool = typer.Option(False, help="Overwrite existing patterns")
    ):
        """Import patterns from a file."""
        asyncio.run(async_import_patterns(import_file, patterns_file, overwrite))
    
    @app.command()
    def sample(
        patterns_file: str = typer.Option("patterns.json", help="Pattern storage file"),
        manufacturer: str = typer.Option("SampleMfg", help="Manufacturer name"),
        model: str = typer.Option("Sample123", help="Model number"),
        protocol: str = typer.Option("modbus.tcp", help="Protocol name"),
        port: int = typer.Option(502, help="Protocol port")
    ):
        """Create a sample pattern for testing."""
        asyncio.run(async_create_sample_pattern(
            patterns_file, manufacturer, model, protocol, port
        ))


def main():
    """Main entry point for CLI."""
    if typer and app:
        app()
    else:
        print("Pattern Management CLI")
        print("=" * 30)
        print()
        print("Available commands (run with Python):")
        print("  discover - Run pattern-enhanced discovery")
        print("  stats    - Show pattern statistics") 
        print("  sample   - Create sample pattern")
        print()
        print("Example usage:")
        print("  python -m bifrost.pattern_cli discover --network 192.168.1.0/24")
        print()
        print("For full CLI functionality, install: pip install typer rich")


if __name__ == "__main__":
    if len(sys.argv) > 1:
        command = sys.argv[1]
        
        if command == "discover":
            # Simple discovery without full CLI
            network = sys.argv[2] if len(sys.argv) > 2 else "192.168.1.0/24"
            asyncio.run(async_discover_with_patterns(network))
            
        elif command == "stats":
            patterns_file = sys.argv[2] if len(sys.argv) > 2 else "patterns.json"
            asyncio.run(async_show_statistics(patterns_file))
            
        elif command == "sample":
            asyncio.run(async_create_sample_pattern())
            
        else:
            main()
    else:
        main()
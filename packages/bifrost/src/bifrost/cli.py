"""Command-line interface for Bifrost."""

import asyncio
from typing import Optional

import typer
from rich.console import Console
from rich.table import Table
from rich.progress import Progress, SpinnerColumn, TextColumn

from .discovery import discover_devices, get_cache_info, clear_discovery_cache, DiscoveryConfig

app = typer.Typer()
console = Console()


@app.command()
def discover(
    network: str = typer.Option("192.168.1.0/24", help="Network range to scan"),
    methods: str = typer.Option("ping,modbus", help="Discovery methods (comma-separated)"),
    timeout: float = typer.Option(5.0, help="Timeout in seconds"),
    no_cache: bool = typer.Option(False, help="Disable caching"),
    cache_ttl: int = typer.Option(300, help="Cache TTL in seconds"),
) -> None:
    """Discover devices on the network."""
    
    # Parse methods
    method_list = [m.strip() for m in methods.split(",")]
    
    # Configure discovery
    config = DiscoveryConfig(
        cache_enabled=not no_cache,
        cache_ttl_seconds=cache_ttl
    )
    
    with Progress(
        SpinnerColumn(),
        TextColumn("[progress.description]{task.description}"),
        console=console,
    ) as progress:
        task = progress.add_task(f"Scanning {network}...", total=None)
        
        async def run_discovery() -> None:
            devices = await discover_devices(
                network=network,
                methods=method_list,
                timeout=timeout,
                config=config,
                use_cache=not no_cache
            )
            progress.update(task, completed=True)
            return devices
        
        devices = asyncio.run(run_discovery())
    
    # Display results
    if not devices:
        console.print("❌ No devices found", style="red")
        return
    
    table = Table(title=f"Discovered Devices ({len(devices)} found)")
    table.add_column("MAC Address", style="cyan")
    table.add_column("IP Address", style="magenta")
    table.add_column("Hostname", style="blue")
    table.add_column("Protocol", style="green")
    table.add_column("Device Type", style="yellow")
    table.add_column("Ports", style="white")
    
    for device in devices:
        table.add_row(
            device.mac_address or "Unknown",
            device.ip_address or "Unknown",
            device.hostname or "Unknown",
            device.protocol.value if device.protocol else "Unknown",
            device.device_type or "Unknown",
            ",".join(map(str, device.ports)) if device.ports else "None"
        )
    
    console.print(table)
    
    # Show cache info if caching is enabled
    if not no_cache:
        cache_info = get_cache_info(config)
        console.print(f"\n💾 Cache: {cache_info['file_count']} files, {cache_info['total_size_mb']:.2f} MB")


@app.command()
def cache_info() -> None:
    """Show discovery cache information."""
    info = get_cache_info()
    
    table = Table(title="Discovery Cache Information")
    table.add_column("Property", style="cyan")
    table.add_column("Value", style="white")
    
    table.add_row("Cache Enabled", "✅ Yes" if info["cache_enabled"] else "❌ No")
    table.add_row("Cache Directory", str(info["cache_dir"]))
    table.add_row("Cache Exists", "✅ Yes" if info["cache_exists"] else "❌ No")
    table.add_row("File Count", str(info["file_count"]))
    table.add_row("Total Size", f"{info['total_size_mb']:.2f} MB")
    
    if info["cache_exists"]:
        table.add_row("TTL", f"{info['ttl_seconds']} seconds")
        table.add_row("Max Size", f"{info['max_size_mb']} MB")
    
    console.print(table)


@app.command()
def cache_clear() -> None:
    """Clear discovery cache."""
    with console.status("Clearing cache..."):
        clear_discovery_cache()
    console.print("✅ Discovery cache cleared", style="green")


if __name__ == "__main__":
    app()

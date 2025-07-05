"""Command-line interface for Bifrost.

This module provides the CLI commands for the Bifrost industrial IoT framework,
including device discovery and network scanning capabilities.
"""

import asyncio
from typing import Optional, Sequence

import typer
from rich.console import Console
from rich.live import Live
from rich.progress import Progress, SpinnerColumn, TextColumn
from rich.table import Table
from rich.text import Text

from .discovery import DiscoveryConfig, discover_devices


def complete_protocols(incomplete: str):
    """Autocomplete function for protocol options."""
    protocols = ["modbus", "cip", "bootp", "ethernet_ip"]
    return [protocol for protocol in protocols if protocol.startswith(incomplete)]


def complete_network_ranges(incomplete: str):
    """Autocomplete function for common network ranges."""
    common_ranges = [
        "192.168.1.0/24",
        "192.168.0.0/24", 
        "10.0.0.0/24",
        "10.0.0.0/16",
        "172.16.0.0/24",
        "127.0.0.1/32",
    ]
    return [range_str for range_str in common_ranges if range_str.startswith(incomplete)]


def complete_timeouts(incomplete: str):
    """Autocomplete function for timeout values."""
    timeouts = ["0.5", "1.0", "2.0", "5.0", "10.0"]
    return [timeout for timeout in timeouts if timeout.startswith(incomplete)]

app = typer.Typer(
    name="bifrost",
    help="Bifrost - Industrial IoT Framework for device discovery and automation",
    add_completion=True,
    rich_markup_mode="rich",
    no_args_is_help=True,
)
console = Console()


@app.command(name="about")
def about() -> None:
    """Show information about Bifrost and its capabilities."""
    console.print()
    console.print("🏭 [bold blue]Bifrost - Industrial IoT Framework[/bold blue]")
    console.print()
    console.print("[dim]Break down the walls between operational technology and information technology.[/dim]")
    console.print("[dim]Make it as easy to work with a PLC as it is to work with a REST API.[/dim]")
    console.print()
    
    console.print("📡 [bold green]Device Discovery Capabilities:[/bold green]")
    console.print("  • [cyan]Modbus TCP[/cyan] - High-speed scanning for Modbus devices")
    console.print("  • [cyan]Ethernet/IP (CIP)[/cyan] - Allen-Bradley and compatible devices") 
    console.print("  • [cyan]BOOTP/DHCP[/cyan] - Devices requesting IP addresses")
    console.print()
    
    console.print("🎯 [bold green]Quick Start Examples:[/bold green]")
    console.print("  [dim]# Discover all devices on your network[/dim]")
    console.print("  [yellow]bifrost discover[/yellow]")
    console.print()
    console.print("  [dim]# Fast Modbus scan[/dim]")
    console.print("  [yellow]bifrost scan-modbus[/yellow]")
    console.print()
    console.print("  [dim]# Scan specific network with verbose output[/dim]")
    console.print("  [yellow]bifrost discover --network 192.168.1.0/24 --verbose[/yellow]")
    console.print()
    
    console.print("🔧 [bold green]Use Cases:[/bold green]")
    console.print("  • Network commissioning and device inventory")
    console.print("  • Troubleshooting and diagnostics") 
    console.print("  • Security auditing and asset discovery")
    console.print("  • SCADA/HMI system integration")
    console.print()
    
    console.print("📚 [bold green]Get Help:[/bold green]")
    console.print("  [yellow]bifrost --help[/yellow]          Show all commands")
    console.print("  [yellow]bifrost discover --help[/yellow]  Discovery options")
    console.print("  [yellow]bifrost scan-modbus --help[/yellow] Modbus scan options")
    console.print()


@app.command()
def discover(
    network: str = typer.Option(
        "192.168.1.0/24", 
        "--network", "-n", 
        help="[cyan]Network range to scan[/cyan] (CIDR notation, e.g., 192.168.1.0/24)"
    ),
    protocols: str = typer.Option(
        "modbus,cip,bootp", 
        "--protocols", "-p", 
        help="[cyan]Protocols to use[/cyan] (modbus,cip,bootp or combinations)"
    ),
    timeout: float = typer.Option(
        2.0, 
        "--timeout", "-t", 
        help="[cyan]Discovery timeout[/cyan] in seconds per device"
    ),
    max_concurrent: int = typer.Option(
        50, 
        "--max-concurrent", "-c", 
        help="[cyan]Max concurrent connections[/cyan] (10-200)"
    ),
    verbose: bool = typer.Option(
        False, 
        "--verbose", "-v", 
        help="[cyan]Show detailed device information[/cyan]"
    ),
) -> None:
    """🔍 Discover industrial devices on your network.
    
    Scans the specified network range using multiple industrial protocols
    to find PLCs, HMIs, and other automation devices.
    
    [bold green]Examples:[/bold green]
    
      [yellow]bifrost discover[/yellow]                    # Default scan (192.168.1.0/24)
      [yellow]bifrost discover -n 10.0.0.0/24[/yellow]     # Scan specific network  
      [yellow]bifrost discover -p modbus[/yellow]          # Modbus only
      [yellow]bifrost discover -v -t 5.0[/yellow]          # Verbose with longer timeout
    """
    
    # Parse protocols
    protocol_list = [p.strip() for p in protocols.split(",")]
    
    # Create discovery configuration
    config = DiscoveryConfig(
        network_range=network,
        timeout=timeout,
        max_concurrent=max_concurrent,
        protocols=protocol_list,
    )
    
    # Create the discovery table
    table = Table(title="🔍 Device Discovery Results")
    table.add_column("Host", style="cyan", no_wrap=True)
    table.add_column("Port", style="magenta", justify="right")
    table.add_column("Protocol", style="green")
    table.add_column("Type", style="yellow")
    table.add_column("Method", style="blue")
    table.add_column("Confidence", style="bright_green", justify="right")
    
    if verbose:
        table.add_column("Manufacturer", style="dim")
        table.add_column("Model", style="dim")
    
    device_count = 0
    
    async def run_discovery() -> None:
        nonlocal device_count
        
        # Create progress display
        progress = Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            console=console,
        )
        
        task = progress.add_task(f"Scanning {network} using {', '.join(protocol_list)}...", total=None)
        
        with Live(progress, refresh_per_second=4):
            async for device in discover_devices(config, protocol_list):
                device_count += 1
                
                # Add device to table
                confidence_text = f"{device.confidence:.1%}"
                if device.confidence >= 0.9:
                    confidence_style = "bright_green"
                elif device.confidence >= 0.7:
                    confidence_style = "yellow"
                else:
                    confidence_style = "red"
                
                confidence_display = Text(confidence_text, style=confidence_style)
                
                row_data = [
                    device.host,
                    str(device.port),
                    device.protocol,
                    device.device_type or "Unknown",
                    device.discovery_method,
                    confidence_display,
                ]
                
                if verbose:
                    row_data.extend([
                        device.manufacturer or "-",
                        device.model or "-",
                    ])
                
                table.add_row(*row_data)
                
                # Update progress description
                progress.update(task, description=f"Found {device_count} devices - {device.host}:{device.port}")
        
        # Final update
        progress.update(task, description=f"Discovery complete - Found {device_count} devices")
    
    try:
        asyncio.run(run_discovery())
    except KeyboardInterrupt:
        console.print("\n[yellow]Discovery interrupted by user[/yellow]")
    except Exception as e:
        console.print(f"\n[red]Discovery failed: {e}[/red]")
        raise typer.Exit(1)
    
    # Display results
    console.print()
    console.print(table)
    
    if device_count == 0:
        console.print("\n[yellow]No devices found. Try:[/yellow]")
        console.print("• Expanding the network range with [cyan]--network[/cyan]")
        console.print("• Increasing the timeout with [cyan]--timeout[/cyan]")
        console.print("• Using different protocols with [cyan]--protocols[/cyan]")
        console.print("• Running with [cyan]--verbose[/cyan] for more details")
    else:
        console.print(f"\n[green]✅ Discovery complete: Found {device_count} devices[/green]")


@app.command(name="scan-modbus")
def scan_modbus(
    network: str = typer.Option(
        "192.168.1.0/24", 
        "--network", "-n", 
        help="[cyan]Network range to scan[/cyan] (CIDR notation)"
    ),
    timeout: float = typer.Option(
        1.0, 
        "--timeout", "-t", 
        help="[cyan]Connection timeout[/cyan] in seconds (0.1-10.0)"
    ),
    max_concurrent: int = typer.Option(
        100, 
        "--max-concurrent", "-c", 
        help="[cyan]Max concurrent connections[/cyan] (50-500)"
    ),
) -> None:
    """⚡ Fast Modbus TCP device scanning.
    
    High-performance scanning specifically for Modbus TCP devices
    on port 502. Uses optimized connection handling for speed.
    
    [bold green]Examples:[/bold green]
    
      [yellow]bifrost scan-modbus[/yellow]                     # Default scan
      [yellow]bifrost scan-modbus -n 10.0.0.0/16[/yellow]      # Large network
      [yellow]bifrost scan-modbus -t 0.5 -c 200[/yellow]       # Fast & aggressive
    """
    
    config = DiscoveryConfig(
        network_range=network,
        timeout=timeout,
        max_concurrent=max_concurrent,
        protocols=["modbus"],
    )
    
    table = Table(title="🔌 Modbus TCP Device Scan")
    table.add_column("Host", style="cyan")
    table.add_column("Port", style="magenta", justify="right")
    table.add_column("Response Time", style="green", justify="right")
    table.add_column("Device Info", style="yellow")
    
    device_count = 0
    
    async def run_modbus_scan() -> None:
        nonlocal device_count
        
        progress = Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            console=console,
        )
        
        task = progress.add_task("Scanning for Modbus devices...", total=None)
        
        with Live(progress, refresh_per_second=4):
            async for device in discover_devices(config, ["modbus"]):
                device_count += 1
                
                response_info = "✓ Connected"
                if device.metadata.get("has_device_identification"):
                    response_info += " (Device ID available)"
                
                table.add_row(
                    device.host,
                    str(device.port),
                    f"< {config.timeout}s",
                    response_info,
                )
                
                progress.update(task, description=f"Found {device_count} Modbus devices")
    
    asyncio.run(run_modbus_scan())
    console.print()
    console.print(table)
    
    if device_count > 0:
        console.print(f"\n[green]✅ Found {device_count} Modbus devices[/green]")
    else:
        console.print("\n[yellow]No Modbus devices found[/yellow]")


def main() -> None:
    """Main entry point for the bifrost CLI."""
    app()


if __name__ == "__main__":
    main()

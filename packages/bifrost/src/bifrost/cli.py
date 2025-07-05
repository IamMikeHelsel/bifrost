"""Command-line interface for Bifrost."""

import asyncio
from typing import Optional, Sequence

import typer
from rich.console import Console
from rich.live import Live
from rich.progress import Progress, SpinnerColumn, TextColumn
from rich.table import Table
from rich.text import Text

from .discovery import DiscoveryConfig, discover_devices

app = typer.Typer()
console = Console()


@app.command()
def discover(
    network: str = typer.Option("192.168.1.0/24", "--network", "-n", help="Network range to scan"),
    protocols: str = typer.Option("modbus,cip,bootp", "--protocols", "-p", help="Comma-separated protocols to use"),
    timeout: float = typer.Option(2.0, "--timeout", "-t", help="Discovery timeout in seconds"),
    max_concurrent: int = typer.Option(50, "--max-concurrent", "-c", help="Maximum concurrent connections"),
    verbose: bool = typer.Option(False, "--verbose", "-v", help="Verbose output"),
) -> None:
    """Discover devices on the network using multiple protocols."""
    
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


@app.command()
def scan_modbus(
    network: str = typer.Option("192.168.1.0/24", "--network", "-n", help="Network range to scan"),
    timeout: float = typer.Option(1.0, "--timeout", "-t", help="Connection timeout"),
    max_concurrent: int = typer.Option(100, "--max-concurrent", "-c", help="Maximum concurrent connections"),
) -> None:
    """Fast Modbus TCP device scanning."""
    
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


if __name__ == "__main__":
    app()

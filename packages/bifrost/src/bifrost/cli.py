"""Command-line interface for Bifrost."""

import asyncio
import sys

import typer
from bifrost_core import DeviceInfo
from bifrost_core.pooling import ConnectionPool
from rich.console import Console
from rich.panel import Panel
from rich.progress import Progress, SpinnerColumn, TextColumn
from rich.prompt import Prompt
from rich.table import Table

from .discovery import assign_device_ip, discover_devices
from .modbus import ModbusTCPConnection

# Rich console for output
console = Console()

# Typer app
app = typer.Typer(
    name="bifrost",
    help="🌉 Industrial IoT Framework - Bridge your OT and IT systems",
    add_completion=False,
    rich_markup_mode="rich",
)

# Global connection pool
pool = ConnectionPool()


def version_callback(value: bool) -> None:
    """Show version information."""
    if value:
        from . import __version__

        console.print(f"🌉 Bifrost [bold blue]{__version__}[/bold blue]")
        console.print("Industrial IoT Framework for bridging OT and IT systems")
        raise typer.Exit()


@app.callback()
def cli_main(
    version: bool | None = typer.Option(
        None,
        "--version",
        "-v",
        callback=version_callback,
        is_eager=True,
        help="Show version information",
    ),
) -> None:
    """🌉 Bifrost - Industrial IoT Framework"""
    pass


@app.command()
def discover(
    network: str = typer.Option(
        "192.168.1.0/24", "--network", "-n", help="🌐 Network to scan (CIDR notation)"
    ),
    methods: list[str] = typer.Option(
        ["ping", "arp", "bootp", "modbus"],
        "--method",
        "-m",
        help="🔍 Discovery methods to use",
    ),
    timeout: float = typer.Option(
        5.0, "--timeout", "-t", help="⏱️ Timeout per method (seconds)"
    ),
    output: str = typer.Option(
        "table", "--output", "-o", help="📋 Output format (table, json, csv)"
    ),
) -> None:
    """🔍 Discover devices on the network"""

    async def run_discovery():
        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            console=console,
            transient=True,
        ) as progress:
            task = progress.add_task(
                f"[cyan]Discovering devices on {network}...", total=None
            )

            try:
                devices = await discover_devices(network, methods, timeout)
                progress.update(task, completed=100)

                if not devices:
                    console.print("❌ No devices found", style="red")
                    return

                console.print(
                    f"✅ Found [bold green]{len(devices)}[/bold green] devices"
                )

                if output == "table":
                    _display_devices_table(devices)
                elif output == "json":
                    _display_devices_json(devices)
                elif output == "csv":
                    _display_devices_csv(devices)

            except Exception as e:
                progress.stop()
                console.print(f"❌ Discovery failed: {e}", style="red")
                raise typer.Exit(1) from e

    asyncio.run(run_discovery())


@app.command()
def assign_ip(
    mac: str = typer.Argument(help="🔗 MAC address of target device"),
    ip: str = typer.Argument(help="🌐 New IP address to assign"),
    subnet: str = typer.Option("255.255.255.0", "--subnet", "-s", help="🗺️ Subnet mask"),
    gateway: str | None = typer.Option(
        None, "--gateway", "-g", help="🚪 Gateway IP address"
    ),
    timeout: float = typer.Option(10.0, "--timeout", "-t", help="⏱️ Timeout (seconds)"),
) -> None:
    """📡 Assign IP address to a device via BootP/DHCP"""

    async def run_assignment():
        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            console=console,
            transient=True,
        ) as progress:
            task = progress.add_task(f"[cyan]Assigning IP {ip} to {mac}...", total=None)

            try:
                success = await assign_device_ip(mac, ip, subnet, gateway or "")
                progress.update(task, completed=100)

                if success:
                    console.print("✅ IP assignment successful", style="green")
                    console.print(f"   Device: [bold]{mac}[/bold]")
                    console.print(f"   IP: [bold]{ip}[/bold]")
                    console.print(f"   Subnet: [bold]{subnet}[/bold]")
                    if gateway:
                        console.print(f"   Gateway: [bold]{gateway}[/bold]")
                else:
                    console.print("❌ IP assignment failed", style="red")
                    raise typer.Exit(1)

            except Exception as e:
                progress.stop()
                console.print(f"❌ Assignment failed: {e}", style="red")
                raise typer.Exit(1) from e

    asyncio.run(run_assignment())


@app.command()
def connect(
    host: str = typer.Argument(help="🌐 Device host/IP address"),
    port: int = typer.Option(502, "--port", "-p", help="🔌 Port number"),
    protocol: str = typer.Option(
        "modbus", "--protocol", help="📡 Protocol (modbus, opcua, s7)"
    ),
    interactive: bool = typer.Option(
        False, "--interactive", "-i", help="💬 Interactive mode"
    ),
) -> None:
    """🔗 Connect to a device"""

    async def run_connection():
        if protocol.lower() not in ["modbus", "modbus-tcp"]:
            console.print(f"❌ Protocol '{protocol}' not supported yet", style="red")
            console.print("Available protocols: modbus, modbus-tcp")
            raise typer.Exit(1)

        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            console=console,
            transient=True,
        ) as progress:
            task = progress.add_task(
                f"[cyan]Connecting to {host}:{port}...", total=None
            )

            try:
                connection = ModbusTCPConnection(host, port)
                await connection.connect()
                progress.update(task, completed=100)

                console.print(
                    f"✅ Connected to [bold]{host}:{port}[/bold]", style="green"
                )

                if interactive:
                    await _interactive_mode(connection)
                else:
                    # Simple connection test
                    health = await connection.health_check()
                    console.print(f"Health check: {'✅ OK' if health else '❌ Failed'}")

                await connection.disconnect()
                console.print("📡 Disconnected", style="yellow")

            except Exception as e:
                progress.stop()
                console.print(f"❌ Connection failed: {e}", style="red")
                raise typer.Exit(1) from e

    asyncio.run(run_connection())


@app.command()
def scan(
    network: str = typer.Option(
        "192.168.1.0/24", "--network", "-n", help="🌐 Network to scan"
    ),
    ports: list[int] = typer.Option(
        [502, 503, 505, 4840], "--port", "-p", help="🔌 Ports to scan"
    ),
    timeout: float = typer.Option(2.0, "--timeout", "-t", help="⏱️ Connection timeout"),
) -> None:
    """🔍 Scan for industrial devices on specific ports"""

    async def run_scan():
        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            console=console,
            transient=True,
        ) as progress:
            task = progress.add_task(
                f"[cyan]Scanning {network} on ports {ports}...", total=None
            )

            try:
                # Use discovery with only modbus method for port scanning
                devices = await discover_devices(network, ["modbus"], timeout)
                progress.update(task, completed=100)

                if not devices:
                    console.print("❌ No devices found", style="red")
                    return

                console.print(
                    f"✅ Found [bold green]{len(devices)}[/bold green] devices"
                )
                _display_devices_table(devices)

            except Exception as e:
                progress.stop()
                console.print(f"❌ Scan failed: {e}", style="red")
                raise typer.Exit(1) from e

    asyncio.run(run_scan())


@app.command()
def status():
    """📊 Show system status and statistics"""

    # System information
    console.print(
        Panel.fit(
            "🌉 [bold blue]Bifrost System Status[/bold blue]", border_style="blue"
        )
    )

    # Connection pool stats
    stats = pool.get_stats()

    table = Table(title="Connection Pool Statistics")
    table.add_column("Metric", style="cyan")
    table.add_column("Value", style="green")

    table.add_row("Total Connections", str(stats["total_connections"]))
    table.add_row("Available", str(stats["available_connections"]))
    table.add_row("In Use", str(stats["borrowed_connections"]))
    table.add_row("Max Pool Size", str(stats["max_size"]))
    table.add_row("Pool Status", "🟢 Open" if not stats["is_closed"] else "🔴 Closed")

    console.print(table)


async def _interactive_mode(connection) -> None:
    """Interactive mode for device communication."""
    console.print("\n💬 [bold cyan]Interactive Mode[/bold cyan]")
    console.print(
        "Commands: read <address> [count], write <address> <value>, health, quit"
    )
    console.print("Examples: read 40001, read holding:1 5, write 40001 100\n")

    while True:
        try:
            command = Prompt.ask("bifrost", default="quit")

            if command.lower() in ["quit", "exit", "q"]:
                break

            parts = command.split()
            if not parts:
                continue

            cmd = parts[0].lower()

            if cmd == "read" and len(parts) >= 2:
                address = parts[1]
                count = int(parts[2]) if len(parts) > 2 else 1

                try:
                    values = await connection.read_raw(address, count)
                    console.print(f"📖 Read {address}: {values}", style="green")
                except Exception as e:
                    console.print(f"❌ Read failed: {e}", style="red")

            elif cmd == "write" and len(parts) >= 3:
                address = parts[1]
                value = parts[2]

                try:
                    await connection.write_raw(address, [int(value)])
                    console.print(f"✏️ Wrote {value} to {address}", style="green")
                except Exception as e:
                    console.print(f"❌ Write failed: {e}", style="red")

            elif cmd == "health":
                try:
                    health = await connection.health_check()
                    console.print(f"🔍 Health: {'✅ OK' if health else '❌ Failed'}")
                except Exception as e:
                    console.print(f"❌ Health check failed: {e}", style="red")

            else:
                console.print("❓ Unknown command", style="yellow")
                console.print("Available: read, write, health, quit")

        except KeyboardInterrupt:
            console.print("\n👋 Goodbye!")
            break
        except Exception as e:
            console.print(f"❌ Error: {e}", style="red")


def _display_devices_table(devices: list[DeviceInfo]) -> None:
    """Display devices in a table format."""
    table = Table(title="🔍 Discovered Devices")
    table.add_column("Device ID", style="cyan")
    table.add_column("IP Address", style="green")
    table.add_column("Protocol", style="magenta")
    table.add_column("Port", style="blue")
    table.add_column("Name", style="yellow")
    table.add_column("Manufacturer", style="white")

    for device in devices:
        table.add_row(
            device.device_id[:12] + "..."
            if len(device.device_id) > 15
            else device.device_id,
            device.host,
            device.protocol.value if device.protocol else "Unknown",
            str(device.port) if device.port else "N/A",
            device.name or "Unknown",
            device.manufacturer or "Unknown",
        )

    console.print(table)


def _display_devices_json(devices: list[DeviceInfo]) -> None:
    """Display devices in JSON format."""
    import json

    devices_data = []
    for device in devices:
        devices_data.append(
            {
                "device_id": device.device_id,
                "host": device.host,
                "port": device.port,
                "protocol": device.protocol.value if device.protocol else None,
                "name": device.name,
                "manufacturer": device.manufacturer,
                "additional_info": device.additional_info,
            }
        )

    console.print(json.dumps(devices_data, indent=2))


def _display_devices_csv(devices: list[DeviceInfo]) -> None:
    """Display devices in CSV format."""
    import csv
    import io

    output = io.StringIO()
    writer = csv.writer(output)

    # Header
    writer.writerow(["Device ID", "Host", "Port", "Protocol", "Name", "Manufacturer"])

    # Data
    for device in devices:
        writer.writerow(
            [
                device.device_id,
                device.host,
                device.port,
                device.protocol.value if device.protocol else "",
                device.name or "",
                device.manufacturer or "",
            ]
        )

    console.print(output.getvalue())


def main() -> int:
    """Main CLI entry point."""
    try:
        app()
        return 0
    except Exception as e:
        console.print(f"❌ Error: {e}", style="red")
        return 1
    finally:
        # Clean up connection pool
        try:
            asyncio.run(pool.close())
        except Exception:
            pass


if __name__ == "__main__":
    sys.exit(main())

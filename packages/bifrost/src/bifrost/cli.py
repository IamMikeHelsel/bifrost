"""Command-line interface for Bifrost."""

import asyncio

import typer
from rich.console import Console
from rich.table import Table

from .discovery import discover_devices

app = typer.Typer()
console = Console()


@app.command()
def discover() -> None:
    """Discover devices on the network."""
    table = Table(title="Discovered Devices")
    table.add_column("Host", style="cyan")
    table.add_column("Port", style="magenta")
    table.add_column("Protocol", style="green")
    table.add_column("Device Type", style="yellow")

    async def run_discovery() -> None:
        async for device in discover_devices():
            table.add_row(
                device["host"],
                str(device["port"]),
                device["protocol"],
                device["device_type"],
            )

    asyncio.run(run_discovery())
    console.print(table)


if __name__ == "__main__":
    app()

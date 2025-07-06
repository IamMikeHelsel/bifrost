"""Base CLI application framework using Rich."""

from contextlib import contextmanager
from enum import Enum
from typing import Any

import typer
from rich.console import Console
from rich.panel import Panel
from rich.progress import Progress, SpinnerColumn, TextColumn
from rich.table import Table
from rich.text import Text
from rich.theme import Theme


class ColorScheme(str, Enum):
    """Standard color scheme for the CLI."""

    SUCCESS = "green"
    WARNING = "yellow"
    ERROR = "red"
    INFO = "blue"
    SPECIAL = "magenta"
    HEADER = "cyan"
    NORMAL = "white"


# Define the Bifrost theme
BIFROST_THEME = Theme(
    {
        "success": "bold green",
        "warning": "bold yellow",
        "error": "bold red",
        "info": "bold blue",
        "special": "bold magenta",
        "header": "bold cyan",
        "normal": "white",
        "dim": "dim white",
        "key": "bold yellow",
        "value": "white",
        "protocol": "bold magenta",
        "connection": "bold green",
        "disconnection": "bold red",
    }
)


class CLIApp:
    """Base class for CLI applications with Rich formatting."""

    def __init__(
        self, name: str = "bifrost", theme: Theme | None = None
    ) -> None:
        self.name = name
        self.console = Console(theme=theme or BIFROST_THEME)
        self.app = typer.Typer(
            name=name,
            help="Bifrost Industrial Edge Computing Framework",
            add_completion=True,
            pretty_exceptions_show_locals=False,
        )

    def print_success(self, message: str) -> None:
        """Print success message."""
        self.console.print(f"✅ {message}", style="success")

    def print_error(self, message: str) -> None:
        """Print error message."""
        self.console.print(f"❌ {message}", style="error")

    def print_warning(self, message: str) -> None:
        """Print warning message."""
        self.console.print(f"⚠️  {message}", style="warning")

    def print_info(self, message: str) -> None:
        """Print info message."""
        self.console.print(f"ℹ️  {message}", style="info")

    def print_header(self, text: str) -> None:
        """Print a section header."""
        self.console.print(f"\n{'─' * 60}", style="dim")
        self.console.print(text, style="header")
        self.console.print(f"{'─' * 60}\n", style="dim")

    @contextmanager
    def progress(self, description: str):
        """Context manager for showing progress."""
        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            console=self.console,
        ) as progress:
            task = progress.add_task(description=description)
            try:
                yield progress
            finally:
                progress.remove_task(task)

    def create_table(self, title: str, columns: dict[str, str]) -> Table:
        """Create a styled table."""
        table = Table(title=title, show_header=True, header_style="header")
        for col_name, col_style in columns.items():
            table.add_column(col_name, style=col_style)
        return table

    def create_panel(
        self, content: str, title: str, border_style: str = "info"
    ) -> Panel:
        """Create a styled panel."""
        return Panel(
            content, title=title, border_style=border_style, expand=False
        )

    def format_key_value(self, key: str, value: Any) -> Text:
        """Format a key-value pair."""
        text = Text()
        text.append(f"{key}: ", style="key")
        text.append(str(value), style="value")
        return text

    def confirm(self, message: str, default: bool = False) -> bool:
        """Ask for user confirmation."""
        return typer.confirm(message, default=default)

    def prompt(self, message: str, default: str | None = None) -> str:
        """Prompt user for input."""
        return typer.prompt(message, default=default)

    def run(self) -> None:
        """Run the CLI application."""
        self.app()


class InteractiveCLI(CLIApp):
    """Extended CLI with interactive features."""

    def __init__(
        self, name: str = "bifrost", theme: Theme | None = None
    ) -> None:
        super().__init__(name, theme)
        self._setup_interactive_commands()

    def _setup_interactive_commands(self) -> None:
        """Set up interactive command features."""
        # This will be extended by subclasses

    def interactive_menu(
        self, options: dict[str, str], title: str = "Select an option"
    ) -> str:
        """Display an interactive menu and return the selection."""
        self.console.print(f"\n[header]{title}[/header]\n")

        # Display options
        for i, (key, description) in enumerate(options.items(), 1):
            self.console.print(f"  [{i}] [key]{key}[/key] - {description}")

        # Get selection
        while True:
            choice = self.prompt("\nEnter your choice")
            try:
                idx = int(choice) - 1
                if 0 <= idx < len(options):
                    return list(options.keys())[idx]
            except ValueError:
                # Check if they entered the key directly
                if choice in options:
                    return choice

            self.print_error("Invalid selection. Please try again.")

    def display_connection_status(
        self, protocol: str, host: str, port: int, connected: bool
    ) -> None:
        """Display connection status in a formatted way."""
        status = "Connected" if connected else "Disconnected"
        style = "connection" if connected else "disconnection"
        icon = "🟢" if connected else "🔴"

        panel = Panel(
            f"{icon} [bold]{status}[/bold]\n\n"
            f"[key]Protocol:[/key] [protocol]{protocol}[/protocol]\n"
            f"[key]Host:[/key] {host}\n"
            f"[key]Port:[/key] {port}",
            title="Connection Status",
            border_style=style,
        )
        self.console.print(panel)

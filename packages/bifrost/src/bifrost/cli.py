"""Command-line interface for Bifrost."""

import sys
from typing import Optional


class CLI:
    """Basic CLI class for Bifrost."""
    
    def __init__(self):
        pass
    
    def run(self, args: Optional[list] = None) -> int:
        """Run the CLI with given arguments."""
        if args is None:
            args = sys.argv[1:]
        
        if not args or args[0] in ("-h", "--help"):
            self.show_help()
            return 0
        
        if args[0] == "version":
            self.show_version()
            return 0
        
        print(f"Unknown command: {args[0]}")
        self.show_help()
        return 1
    
    def show_help(self) -> None:
        """Show help message."""
        print("Bifrost - Industrial IoT Framework")
        print()
        print("Usage:")
        print("  bifrost [command]")
        print()
        print("Commands:")
        print("  version     Show version information")
        print("  help        Show this help message")
        print()
        print("For more information, visit: https://github.com/bifrost-dev/bifrost")
    
    def show_version(self) -> None:
        """Show version information."""
        from . import __version__
        print(f"Bifrost {__version__}")


def main() -> int:
    """Main CLI entry point."""
    cli = CLI()
    return cli.run()


if __name__ == "__main__":
    sys.exit(main())
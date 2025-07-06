"""Logging and error handling infrastructure."""

import json
import logging
import sys
from contextlib import contextmanager
from datetime import datetime
from enum import Enum
from pathlib import Path
from typing import Any

from rich.console import Console
from rich.logging import RichHandler


class LogLevel(str, Enum):
    """Log levels."""

    DEBUG = "DEBUG"
    INFO = "INFO"
    WARNING = "WARNING"
    ERROR = "ERROR"
    CRITICAL = "CRITICAL"


class BifrostLogger:
    """Centralized logging configuration for Bifrost."""

    _loggers: dict[str, logging.Logger] = {}
    _console: Console | None = None
    _log_dir: Path | None = None
    _file_handler: logging.FileHandler | None = None

    @classmethod
    def setup(
        cls,
        level: LogLevel = LogLevel.INFO,
        console_output: bool = True,
        file_output: bool = False,
        log_dir: Path | None = None,
        rich_console: Console | None = None,
    ) -> None:
        """Configure logging for the entire application."""
        cls._console = rich_console
        cls._log_dir = (
            Path(log_dir) if log_dir else Path.home() / ".bifrost" / "logs"
        )

        if file_output and cls._log_dir:
            cls._log_dir.mkdir(parents=True, exist_ok=True)
            log_file = (
                cls._log_dir / f"bifrost_{datetime.now():%Y%m%d_%H%M%S}.log"
            )
            cls._file_handler = logging.FileHandler(log_file)
            cls._file_handler.setFormatter(
                logging.Formatter(
                    "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
                )
            )

        # Configure root logger
        root_logger = logging.getLogger()
        root_logger.setLevel(level.value)

        # Remove existing handlers
        root_logger.handlers.clear()

        # Add console handler if requested
        if console_output:
            console_handler = RichHandler(
                console=cls._console,
                show_time=True,
                show_path=False,
                markup=True,
                rich_tracebacks=True,
                tracebacks_show_locals=True,
            )
            console_handler.setLevel(level.value)
            root_logger.addHandler(console_handler)

        # Add file handler if configured
        if cls._file_handler:
            cls._file_handler.setLevel(level.value)
            root_logger.addHandler(cls._file_handler)

    @classmethod
    def get_logger(cls, name: str) -> logging.Logger:
        """Get or create a logger with the given name."""
        if name not in cls._loggers:
            logger = logging.getLogger(name)
            cls._loggers[name] = logger
        return cls._loggers[name]

    @classmethod
    def log_exception(
        cls,
        logger: logging.Logger,
        exception: Exception,
        context: dict[str, Any] | None = None,
    ) -> None:
        """Log an exception with optional context."""
        error_data = {
            "exception_type": type(exception).__name__,
            "exception_message": str(exception),
            "context": context or {},
        }
        logger.error(
            f"Exception occurred: {json.dumps(error_data)}", exc_info=True
        )

    @classmethod
    @contextmanager
    def log_operation(
        cls,
        logger: logging.Logger,
        operation: str,
        level: LogLevel = LogLevel.INFO,
    ):
        """Context manager for logging operations."""
        start_time = datetime.now()
        logger.log(level.value, f"Starting {operation}")

        try:
            yield
            duration = (datetime.now() - start_time).total_seconds()
            logger.log(level.value, f"Completed {operation} in {duration:.2f}s")
        except Exception as e:
            duration = (datetime.now() - start_time).total_seconds()
            logger.error(f"Failed {operation} after {duration:.2f}s: {str(e)}")
            raise


class BifrostError(Exception):
    """Base exception for all Bifrost errors."""

    def __init__(
        self,
        message: str,
        code: str | None = None,
        details: dict[str, Any] | None = None,
    ) -> None:
        super().__init__(message)
        self.code = code
        self.details = details or {}
        self.timestamp = datetime.utcnow()

    def to_dict(self) -> dict[str, Any]:
        """Convert exception to dictionary."""
        return {
            "error": self.__class__.__name__,
            "message": str(self),
            "code": self.code,
            "details": self.details,
            "timestamp": self.timestamp.isoformat(),
        }


class ConnectionError(BifrostError):
    """Raised when connection to device fails."""



class ProtocolError(BifrostError):
    """Raised when protocol-specific error occurs."""



class ConfigurationError(BifrostError):
    """Raised when configuration is invalid."""



class DataValidationError(BifrostError):
    """Raised when data validation fails."""



class TimeoutError(BifrostError):
    """Raised when operation times out."""



def setup_exception_handling() -> None:
    """Set up global exception handling."""

    def handle_exception(exc_type, exc_value, exc_traceback):
        if issubclass(exc_type, KeyboardInterrupt):
            sys.__excepthook__(exc_type, exc_value, exc_traceback)
            return

        logger = BifrostLogger.get_logger("bifrost.error")
        logger.critical(
            "Uncaught exception", exc_info=(exc_type, exc_value, exc_traceback)
        )

    sys.excepthook = handle_exception


# Convenience functions
def get_logger(name: str) -> logging.Logger:
    """Get a logger instance."""
    return BifrostLogger.get_logger(name)


def log_exception(
    logger: logging.Logger,
    exception: Exception,
    context: dict[str, Any] | None = None,
) -> None:
    """Log an exception with context."""
    BifrostLogger.log_exception(logger, exception, context)

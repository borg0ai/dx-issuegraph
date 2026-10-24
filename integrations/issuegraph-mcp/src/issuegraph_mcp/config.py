"""Configuration for issuegraph MCP server."""

import os
import shutil
import sys
from pathlib import Path

from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


def _default_issuegraph_path() -> str:
    """Get default issuegraph executable path.

    First tries to find issuegraph in PATH, falls back to ~/.local/bin/issuegraph.

    Returns:
        Default path to issuegraph executable
    """
    # Try to find issuegraph in PATH first
    issuegraph_in_path = shutil.which("issuegraph")
    if issuegraph_in_path:
        return issuegraph_in_path

    # Fall back to common install location
    return str(Path.home() / ".local" / "bin" / "issuegraph")


class Config(BaseSettings):
    """Server configuration loaded from environment variables."""

    model_config = SettingsConfigDict(env_prefix="")

    issuegraph_path: str = Field(default_factory=_default_issuegraph_path)
    issuegraph_dir: str | None = None
    issuegraph_db: str | None = None
    issuegraph_actor: str | None = None
    issuegraph_no_auto_flush: bool = False
    issuegraph_no_auto_import: bool = False
    issuegraph_working_dir: str | None = None

    @field_validator("issuegraph_path")
    @classmethod
    def validate_issuegraph_path(cls, v: str) -> str:
        """Validate ISSUEGRAPH_PATH points to an executable issuegraph binary.

        Args:
            v: Path to issuegraph executable (can be command name or absolute path)

        Returns:
            Validated absolute path

        Raises:
            ValueError: If path is invalid or not executable
        """
        path = Path(v)

        # If not an absolute/existing path, try to find it in PATH
        if not path.exists():
            found = shutil.which(v)
            if found:
                v = found
                path = Path(v)
            else:
                raise ValueError(
                    f"issuegraph executable not found at: {v}\n\n"
                    + "The issuegraph Claude Code plugin requires the issuegraph CLI to be installed.\n\n"
                    + "Install issuegraph CLI:\n"
                    + "  curl -fsSL https://raw.githubusercontent.com/borg0ai/dx-issuegraph/main/scripts/install.sh "
                    + "| bash\n\n"
                    + "Or visit: https://github.com/borg0ai/dx-issuegraph#installation\n\n"
                    + "After installation, restart Claude Code to reload the MCP server."
                )

        if not os.access(v, os.X_OK):
            raise ValueError(f"issuegraph executable at {v} is not executable.\nPlease check file permissions.")

        return v

    @field_validator("issuegraph_dir")
    @classmethod
    def validate_issuegraph_dir(cls, v: str | None) -> str | None:
        """Validate ISSUEGRAPH_DIR points to an existing .issuegraph directory.

        Args:
            v: Path to .issuegraph directory or None

        Returns:
            Validated path or None

        Raises:
            ValueError: If path is set but directory doesn't exist
        """
        if v is None:
            return v

        path = Path(v)
        if not path.exists():
            raise ValueError(
                f"ISSUEGRAPH_DIR points to non-existent directory: {v}\n"
                + "Please verify the .issuegraph directory path is correct."
            )

        if not path.is_dir():
            raise ValueError(f"ISSUEGRAPH_DIR must point to a directory, not a file: {v}")

        return v

    @field_validator("issuegraph_db")
    @classmethod
    def validate_issuegraph_db(cls, v: str | None) -> str | None:
        """Validate ISSUEGRAPH_DB points to an existing database file.

        Args:
            v: Path to database file or None

        Returns:
            Validated path or None

        Raises:
            ValueError: If path is set but file doesn't exist
        """
        if v is None:
            return v

        path = Path(v)
        if not path.exists():
            raise ValueError(
                f"ISSUEGRAPH_DB points to non-existent file: {v}\n" + "Please verify the database path is correct."
            )

        return v


class ConfigError(Exception):
    """Configuration error with helpful message."""

    pass


def load_config() -> Config:
    """Load and validate configuration from environment variables.

    Returns:
        Validated configuration

    Raises:
        ConfigError: If configuration is invalid
    """
    try:
        return Config()
    except Exception as e:
        default_path = _default_issuegraph_path()
        error_msg = (
            "IssueGraph MCP Server Configuration Error\n\n"
            + f"{e}\n\n"
            + "Common fix: Install the issuegraph CLI first:\n"
            + "  curl -fsSL https://raw.githubusercontent.com/borg0ai/dx-issuegraph/main/scripts/install.sh | bash\n\n"
            + "Or visit: https://github.com/borg0ai/dx-issuegraph#installation\n\n"
            + "After installation, restart Claude Code.\n\n"
            + "Advanced configuration (optional):\n"
            + f"  ISSUEGRAPH_PATH            - Path to issuegraph executable (default: {default_path})\n"
            + "  ISSUEGRAPH_DIR             - Path to .issuegraph directory (default: auto-discover)\n"
            + "  ISSUEGRAPH_DB              - Path to database file (deprecated, use ISSUEGRAPH_DIR)\n"
            + "  ISSUEGRAPH_WORKING_DIR     - Working directory for issuegraph commands (default: $PWD or cwd)\n"
            + "  ISSUEGRAPH_ACTOR           - Actor name for audit trail (default: $USER)\n"
            + "  ISSUEGRAPH_NO_AUTO_FLUSH   - Disable automatic JSONL sync (default: false)\n"
            + "  ISSUEGRAPH_NO_AUTO_IMPORT  - Disable automatic JSONL import (default: false)"
        )
        print(error_msg, file=sys.stderr)
        raise ConfigError(error_msg) from e

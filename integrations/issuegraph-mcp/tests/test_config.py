"""Tests for issuegraph_mcp.config module."""

from pathlib import Path
from unittest.mock import patch

import pytest

from issuegraph_mcp.config import Config, ConfigError, load_config


class TestConfig:
    """Tests for Config class."""

    def test_default_issuegraph_path_auto_detection(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Test that issuegraph is auto-detected from PATH when ISSUEGRAPH_PATH not set."""
        # Clear ISSUEGRAPH_PATH if set
        monkeypatch.delenv("ISSUEGRAPH_PATH", raising=False)

        # Mock shutil.which to return a test path
        with patch("shutil.which", return_value="/usr/local/bin/issuegraph"), patch("os.access", return_value=True):
            config = Config()
            assert config.issuegraph_path == "/usr/local/bin/issuegraph"

    def test_issuegraph_path_from_env(self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
        """Test that ISSUEGRAPH_PATH environment variable is respected."""
        # Create a fake issuegraph executable
        issuegraph_path = tmp_path / "issuegraph"
        issuegraph_path.touch(mode=0o755)

        monkeypatch.setenv("ISSUEGRAPH_PATH", str(issuegraph_path))
        config = Config()
        assert config.issuegraph_path == str(issuegraph_path)

    def test_issuegraph_path_command_name_resolution(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Test that command names like 'issuegraph' are resolved via PATH."""
        # Set ISSUEGRAPH_PATH to just "issuegraph" (command name, not path)
        monkeypatch.setenv("ISSUEGRAPH_PATH", "issuegraph")

        # Mock shutil.which to simulate finding issuegraph in PATH
        with patch("shutil.which", return_value="/usr/local/bin/issuegraph"), patch("os.access", return_value=True):
            config = Config()
            assert config.issuegraph_path == "/usr/local/bin/issuegraph"

    def test_issuegraph_path_not_found(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Test that invalid ISSUEGRAPH_PATH raises ValueError."""
        monkeypatch.setenv("ISSUEGRAPH_PATH", "/nonexistent/issuegraph")

        with pytest.raises(ValueError, match="issuegraph executable not found"):
            Config()

    def test_issuegraph_path_not_executable(self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
        """Test that non-executable issuegraph raises ValueError."""
        # Create a non-executable file
        issuegraph_path = tmp_path / "issuegraph"
        issuegraph_path.touch(mode=0o644)  # rw-r--r--

        monkeypatch.setenv("ISSUEGRAPH_PATH", str(issuegraph_path))

        with pytest.raises(ValueError, match="not executable"):
            Config()

    def test_issuegraph_db_validation(self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
        """Test that ISSUEGRAPH_DB must point to existing file."""
        # Create valid issuegraph executable
        issuegraph_path = tmp_path / "issuegraph"
        issuegraph_path.touch(mode=0o755)
        monkeypatch.setenv("ISSUEGRAPH_PATH", str(issuegraph_path))

        # Set ISSUEGRAPH_DB to non-existent file
        monkeypatch.setenv("ISSUEGRAPH_DB", "/nonexistent/db.sqlite")

        with pytest.raises(ValueError, match="non-existent file"):
            Config()

    def test_issuegraph_db_none_allowed(self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
        """Test that ISSUEGRAPH_DB can be unset (None)."""
        # Create valid issuegraph executable
        issuegraph_path = tmp_path / "issuegraph"
        issuegraph_path.touch(mode=0o755)
        monkeypatch.setenv("ISSUEGRAPH_PATH", str(issuegraph_path))
        monkeypatch.delenv("ISSUEGRAPH_DB", raising=False)

        config = Config()
        assert config.issuegraph_db is None

    def test_issuegraph_actor_from_env(self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
        """Test that ISSUEGRAPH_ACTOR is read from environment."""
        issuegraph_path = tmp_path / "issuegraph"
        issuegraph_path.touch(mode=0o755)
        monkeypatch.setenv("ISSUEGRAPH_PATH", str(issuegraph_path))
        monkeypatch.setenv("ISSUEGRAPH_ACTOR", "test-user")

        config = Config()
        assert config.issuegraph_actor == "test-user"

    def test_auto_flags_default_false(self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
        """Test that auto-flush and auto-import default to False."""
        issuegraph_path = tmp_path / "issuegraph"
        issuegraph_path.touch(mode=0o755)
        monkeypatch.setenv("ISSUEGRAPH_PATH", str(issuegraph_path))

        config = Config()
        assert config.issuegraph_no_auto_flush is False
        assert config.issuegraph_no_auto_import is False


class TestLoadConfig:
    """Tests for load_config function."""

    def test_load_config_success(self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
        """Test that load_config returns valid Config."""
        issuegraph_path = tmp_path / "issuegraph"
        issuegraph_path.touch(mode=0o755)
        monkeypatch.setenv("ISSUEGRAPH_PATH", str(issuegraph_path))

        config = load_config()
        assert isinstance(config, Config)
        assert config.issuegraph_path == str(issuegraph_path)

    def test_load_config_error_handling(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Test that load_config raises ConfigError with helpful message."""
        monkeypatch.setenv("ISSUEGRAPH_PATH", "/nonexistent/issuegraph")

        with pytest.raises(ConfigError, match="Configuration Error"):
            load_config()

"""Client for interacting with issuegraph (issuegraph) CLI and daemon."""

import asyncio
import json
import os
import re
import sys
from abc import ABC, abstractmethod
from typing import Any

from .config import load_config
from .models import (
    AddCommentParams,
    AddDependencyParams,
    AddNoteParams,
    BlockedIssue,
    BlockedParams,
    ClaimIssueParams,
    CloseIssueParams,
    Comment,
    CreateIssueParams,
    InitParams,
    Issue,
    ListCommentsParams,
    ListIssuesParams,
    ReadyWorkParams,
    ReopenIssueParams,
    ShowIssueParams,
    Stats,
    UpdateIssueParams,
)


def _sanitize_issue_deps(issue: dict[str, Any]) -> dict[str, Any]:
    """Strip raw dependency records that don't match the LinkedIssue schema.

    issuegraph list/ready/blocked --json returns raw dep records (issue_id, depends_on_id,
    type, created_at) but the Pydantic Issue model expects enriched LinkedIssue
    objects (id, title, status, etc.). Replace raw records with empty lists and
    preserve counts so validation succeeds.
    """
    for field, count_field in [
        ("dependencies", "dependency_count"),
        ("dependents", "dependent_count"),
    ]:
        raw = issue.get(field)
        if isinstance(raw, list) and raw and isinstance(raw[0], dict) and "depends_on_id" in raw[0]:
            issue[count_field] = len(raw)
            issue[field] = []
    return issue


class BdError(Exception):
    """Base exception for issuegraph CLI errors."""

    pass


class BdNotFoundError(BdError):
    """Raised when issuegraph command is not found."""

    @staticmethod
    def installation_message(attempted_path: str) -> str:
        """Get helpful installation message.

        Args:
            attempted_path: Path where we tried to find issuegraph

        Returns:
            Formatted error message with installation instructions
        """
        return (
            f"issuegraph CLI not found at: {attempted_path}\n\n"
            "The issuegraph Claude Code plugin requires the issuegraph CLI to be installed separately.\n\n"
            "Install issuegraph CLI:\n"
            "  curl -fsSL https://raw.githubusercontent.com/borg0ai/dx-issuegraph/main/scripts/install.sh | bash\n\n"
            "Or visit: https://github.com/borg0ai/dx-issuegraph#installation\n\n"
            "After installation, restart Claude Code to reload the MCP server."
        )


class BdCommandError(BdError):
    """Raised when issuegraph command fails."""

    stderr: str
    returncode: int

    def __init__(self, message: str, stderr: str = "", returncode: int = 1):
        super().__init__(message)
        self.stderr = stderr
        self.returncode = returncode


class BdVersionError(BdError):
    """Raised when issuegraph version is incompatible with MCP server."""

    pass


class BdClientBase(ABC):
    """Abstract base class for issuegraph clients (CLI or daemon)."""

    @abstractmethod
    async def ready(self, params: ReadyWorkParams | None = None) -> list[Issue]:
        """Get ready work (issues with no blockers)."""
        pass

    @abstractmethod
    async def list_issues(self, params: ListIssuesParams | None = None) -> list[Issue]:
        """List issues with optional filters."""
        pass

    @abstractmethod
    async def show(self, params: ShowIssueParams) -> Issue:
        """Show detailed issue information."""
        pass

    @abstractmethod
    async def create(self, params: CreateIssueParams) -> Issue:
        """Create a new issue."""
        pass

    @abstractmethod
    async def update(self, params: UpdateIssueParams) -> Issue:
        """Update an existing issue."""
        pass

    @abstractmethod
    async def claim(self, params: ClaimIssueParams) -> Issue:
        """Atomically claim an issue for work."""
        pass

    @abstractmethod
    async def close(self, params: CloseIssueParams) -> list[Issue]:
        """Close one or more issues."""
        pass

    @abstractmethod
    async def reopen(self, params: ReopenIssueParams) -> list[Issue]:
        """Reopen one or more closed issues."""
        pass

    @abstractmethod
    async def add_dependency(self, params: AddDependencyParams) -> None:
        """Add a dependency between issues."""
        pass

    @abstractmethod
    async def add_comment(self, params: AddCommentParams) -> str:
        """Add a comment to an issue."""
        pass

    @abstractmethod
    async def list_comments(self, params: ListCommentsParams) -> list[Comment]:
        """List comments on an issue."""
        pass

    @abstractmethod
    async def add_note(self, params: AddNoteParams) -> str:
        """Append a note to an issue's notes field."""
        pass

    @abstractmethod
    async def quickstart(self) -> str:
        """Get quickstart guide."""
        pass

    @abstractmethod
    async def stats(self) -> Stats:
        """Get repository statistics."""
        pass

    @abstractmethod
    async def blocked(self, params: BlockedParams | None = None) -> list[BlockedIssue]:
        """Get blocked issues."""
        pass

    @abstractmethod
    async def init(self, params: InitParams | None = None) -> str:
        """Initialize a new issuegraph database."""
        pass

    @abstractmethod
    async def inspect_migration(self) -> dict[str, Any]:
        """Get migration plan and database state for agent analysis."""
        pass

    @abstractmethod
    async def get_schema_info(self) -> dict[str, Any]:
        """Get current database schema for inspection."""
        pass

    @abstractmethod
    async def repair_deps(self, fix: bool = False) -> dict[str, Any]:
        """Find and optionally fix orphaned dependency references.

        Args:
            fix: If True, automatically remove orphaned dependencies

        Returns:
            Dict with orphans_found, orphans list, and fixed count if fix=True
        """
        pass

    @abstractmethod
    async def detect_pollution(self, clean: bool = False) -> dict[str, Any]:
        """Detect test issues that leaked into production database.

        Args:
            clean: If True, delete detected test issues

        Returns:
            Dict with detected test issues and deleted count if clean=True
        """
        pass

    @abstractmethod
    async def validate(self, checks: str | None = None, fix_all: bool = False) -> dict[str, Any]:
        """Run database validation checks.

        Args:
            checks: Comma-separated list of checks (orphans,duplicates,pollution,conflicts)
            fix_all: If True, auto-fix all fixable issues

        Returns:
            Dict with validation results for each check
        """
        pass


class BdCliClient(BdClientBase):
    """Client for calling issuegraph CLI commands and parsing JSON output."""

    issuegraph_path: str
    issuegraph_dir: str | None
    issuegraph_db: str | None
    actor: str | None
    no_auto_flush: bool
    no_auto_import: bool
    working_dir: str | None

    def __init__(
        self,
        issuegraph_path: str | None = None,
        issuegraph_dir: str | None = None,
        issuegraph_db: str | None = None,
        actor: str | None = None,
        no_auto_flush: bool | None = None,
        no_auto_import: bool | None = None,
        working_dir: str | None = None,
    ):
        """Initialize issuegraph client.

        Args:
            issuegraph_path: Path to issuegraph executable (optional, loads from config if not provided)
            issuegraph_dir: Path to .issuegraph directory (optional, loads from config if not provided)
            issuegraph_db: Path to issuegraph database file (deprecated, optional, loads from config if not provided)
            actor: Actor name for audit trail (optional, loads from config if not provided)
            no_auto_flush: Disable automatic JSONL sync (optional, loads from config if not provided)
            no_auto_import: Disable automatic JSONL import (optional, loads from config if not provided)
            working_dir: Working directory for issuegraph commands (optional, loads from config/env if not provided)
        """
        config = load_config()
        self.issuegraph_path = issuegraph_path if issuegraph_path is not None else config.issuegraph_path
        self.issuegraph_dir = issuegraph_dir if issuegraph_dir is not None else config.issuegraph_dir
        self.issuegraph_db = issuegraph_db if issuegraph_db is not None else config.issuegraph_db
        self.actor = actor if actor is not None else config.issuegraph_actor
        self.no_auto_flush = no_auto_flush if no_auto_flush is not None else config.issuegraph_no_auto_flush
        self.no_auto_import = no_auto_import if no_auto_import is not None else config.issuegraph_no_auto_import
        self.working_dir = working_dir if working_dir is not None else config.issuegraph_working_dir

    def _get_working_dir(self) -> str:
        """Get working directory for issuegraph commands.

        Returns:
            Working directory path, falls back to current directory if not configured
        """
        if self.working_dir:
            return self.working_dir
        # Use process working directory (set by MCP client at spawn time)
        return os.getcwd()

    def _global_flags(self) -> list[str]:
        """Build list of global flags for issuegraph commands.

        Returns:
            List of global flag arguments
        """
        flags = []
        # NOTE: --db flag removed in v0.20.1, issuegraph now auto-discovers database via cwd
        # We pass cwd via _run_command instead
        if self.actor:
            flags.extend(["--actor", self.actor])
        if self.no_auto_flush:
            flags.append("--no-auto-flush")
        if self.no_auto_import:
            flags.append("--no-auto-import")
        return flags

    async def _run_command(self, *args: str, cwd: str | None = None) -> Any:
        """Run issuegraph command and parse JSON output.

        Args:
            *args: Command arguments to pass to issuegraph
            cwd: Optional working directory override for this command

        Returns:
            Parsed JSON output (dict or list)

        Raises:
            BdNotFoundError: If issuegraph command not found
            BdCommandError: If issuegraph command fails
        """
        cmd = [self.issuegraph_path, *args, *self._global_flags(), "--json"]
        working_dir = cwd if cwd is not None else self._get_working_dir()

        # Set up environment with database configuration
        env = os.environ.copy()
        if self.issuegraph_dir:
            env["ISSUEGRAPH_DIR"] = self.issuegraph_dir
        elif self.issuegraph_db:
            env["ISSUEGRAPH_DB"] = self.issuegraph_db

        # Log database routing for debugging
        if self.issuegraph_dir:
            db_info = f"ISSUEGRAPH_DIR={self.issuegraph_dir}"
        elif self.issuegraph_db:
            db_info = f"ISSUEGRAPH_DB={self.issuegraph_db} (deprecated)"
        else:
            db_info = "auto-discover"
        print(f"[issuegraph-mcp] Running issuegraph command: {' '.join(args)}", file=sys.stderr)
        print(f"[issuegraph-mcp]   Database: {db_info}", file=sys.stderr)
        print(f"[issuegraph-mcp]   Working dir: {working_dir}", file=sys.stderr)

        try:
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdin=asyncio.subprocess.DEVNULL,  # Prevent inheriting MCP's stdin
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=working_dir,
                env=env,
            )
            stdout, stderr = await process.communicate()
        except FileNotFoundError as e:
            raise BdNotFoundError(BdNotFoundError.installation_message(self.issuegraph_path)) from e

        if process.returncode != 0:
            raise BdCommandError(
                f"issuegraph command failed: {stderr.decode()}",
                stderr=stderr.decode(),
                returncode=process.returncode or 1,
            )

        stdout_str = stdout.decode().strip()
        if not stdout_str:
            return {}

        try:
            result: object = json.loads(stdout_str)
            return result
        except json.JSONDecodeError as e:
            raise BdCommandError(
                f"Failed to parse issuegraph JSON output: {e}",
                stderr=stdout_str,
            ) from e

    async def _check_version(self) -> None:
        """Check that issuegraph CLI version meets minimum requirements.

        Raises:
            BdVersionError: If issuegraph version is incompatible
            BdNotFoundError: If issuegraph command not found
        """
        # Minimum required version
        min_version = (0, 9, 0)

        try:
            process = await asyncio.create_subprocess_exec(
                self.issuegraph_path,
                "version",
                stdin=asyncio.subprocess.DEVNULL,  # Prevent inheriting MCP's stdin
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=self._get_working_dir(),
            )
            stdout, stderr = await process.communicate()
        except FileNotFoundError as e:
            raise BdNotFoundError(BdNotFoundError.installation_message(self.issuegraph_path)) from e

        if process.returncode != 0:
            raise BdCommandError(
                f"issuegraph version failed: {stderr.decode()}",
                stderr=stderr.decode(),
                returncode=process.returncode or 1,
            )

        # Parse version from output like "issuegraph version 0.9.2"
        version_output = stdout.decode().strip()
        match = re.search(r"(\d+)\.(\d+)\.(\d+)", version_output)
        if not match:
            raise BdVersionError(f"Could not parse issuegraph version from: {version_output}")

        version = tuple(int(x) for x in match.groups())

        if version < min_version:
            min_ver_str = ".".join(str(x) for x in min_version)
            cur_ver_str = ".".join(str(x) for x in version)
            install_cmd = (
                "curl -fsSL https://raw.githubusercontent.com/borg0ai/dx-issuegraph/main/scripts/install.sh | bash"
            )
            raise BdVersionError(
                f"issuegraph version {cur_ver_str} is too old. "
                f"This MCP server requires issuegraph >= {min_ver_str}. "
                f"Update with: {install_cmd}"
            )

    async def ready(self, params: ReadyWorkParams | None = None) -> list[Issue]:
        """Get ready work (issues with no blocking dependencies).

        Args:
            params: Query parameters

        Returns:
            List of ready issues
        """
        params = params or ReadyWorkParams()
        args = ["ready", "--limit", str(params.limit)]

        if params.issue_type:
            args.extend(["--type", params.issue_type])
        if params.priority is not None:
            args.extend(["--priority", str(params.priority)])
        if params.assignee:
            args.extend(["--assignee", params.assignee])
        if params.labels:
            for label in params.labels:
                args.extend(["--label", label])
        if params.labels_any:
            for label in params.labels_any:
                args.extend(["--label-any", label])
        if params.unassigned:
            args.append("--unassigned")
        if params.sort_policy:
            args.extend(["--sort", params.sort_policy])
        if params.parent_id:
            args.extend(["--parent", params.parent_id])

        data = await self._run_command(*args)
        if not isinstance(data, list):
            return []

        return [Issue.model_validate(_sanitize_issue_deps(issue)) for issue in data]

    async def list_issues(self, params: ListIssuesParams | None = None) -> list[Issue]:
        """List issues with optional filters.

        Args:
            params: Query parameters

        Returns:
            List of issues
        """
        params = params or ListIssuesParams()
        args = ["list"]

        if params.status:
            args.extend(["--status", params.status])
        if params.priority is not None:
            args.extend(["--priority", str(params.priority)])
        if params.issue_type:
            args.extend(["--type", params.issue_type])
        if params.assignee:
            args.extend(["--assignee", params.assignee])
        if params.labels:
            for label in params.labels:
                args.extend(["--label", label])
        if params.labels_any:
            for label in params.labels_any:
                args.extend(["--label-any", label])
        if params.query:
            args.extend(["--title", params.query])
        if params.unassigned:
            args.append("--no-assignee")
        if params.limit:
            args.extend(["--limit", str(params.limit)])

        data = await self._run_command(*args)
        if not isinstance(data, list):
            return []

        return [Issue.model_validate(_sanitize_issue_deps(issue)) for issue in data]

    async def show(self, params: ShowIssueParams) -> Issue:
        """Show issue details.

        Args:
            params: Issue ID to show

        Returns:
            Issue details

        Raises:
            BdCommandError: If issue not found
        """
        data = await self._run_command("show", params.issue_id)
        # issuegraph show returns an array, extract first element
        if isinstance(data, list):
            if not data:
                raise BdCommandError(f"Issue not found: {params.issue_id}")
            data = data[0]

        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for show {params.issue_id}")

        return Issue.model_validate(data)

    async def create(self, params: CreateIssueParams) -> Issue:
        """Create a new issue.

        Args:
            params: Issue creation parameters

        Returns:
            Created issue
        """
        args = ["create", params.title, "-p", str(params.priority), "-t", params.issue_type]

        if params.description:
            args.extend(["-d", params.description])
        if params.design:
            args.extend(["--design", params.design])
        if params.acceptance:
            args.extend(["--acceptance", params.acceptance])
        if params.external_ref:
            args.extend(["--external-ref", params.external_ref])
        if params.assignee:
            args.extend(["--assignee", params.assignee])
        if params.id:
            args.extend(["--id", params.id])
        for label in params.labels:
            args.extend(["-l", label])
        if params.deps:
            args.extend(["--deps", ",".join(params.deps)])

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for create")

        return Issue.model_validate(data)

    async def update(self, params: UpdateIssueParams) -> Issue:
        """Update an issue.

        Args:
            params: Issue update parameters

        Returns:
            Updated issue
        """
        args = ["update", params.issue_id]

        if params.status:
            args.extend(["--status", params.status])
        if params.priority is not None:
            args.extend(["--priority", str(params.priority)])
        if params.assignee:
            args.extend(["--assignee", params.assignee])
        if params.title:
            args.extend(["--title", params.title])
        if params.description:
            args.extend(["--description", params.description])
        if params.design:
            args.extend(["--design", params.design])
        if params.acceptance_criteria:
            args.extend(["--acceptance", params.acceptance_criteria])
        if params.notes:
            args.extend(["--notes", params.notes])
        if params.external_ref:
            args.extend(["--external-ref", params.external_ref])

        data = await self._run_command(*args)
        # issuegraph update returns an array, extract first element
        if isinstance(data, list):
            if not data:
                raise BdCommandError(f"Issue not found: {params.issue_id}")
            data = data[0]

        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for update {params.issue_id}")

        return Issue.model_validate(data)

    async def claim(self, params: ClaimIssueParams) -> Issue:
        """Atomically claim an issue via issuegraph update --claim.

        Args:
            params: Claim parameters

        Returns:
            Claimed issue
        """
        data = await self._run_command("update", params.issue_id, "--claim")
        # issuegraph update returns an array, extract first element
        if isinstance(data, list):
            if not data:
                raise BdCommandError(f"Issue not found: {params.issue_id}")
            data = data[0]

        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for claim {params.issue_id}")

        return Issue.model_validate(data)

    async def close(self, params: CloseIssueParams) -> list[Issue]:
        """Close an issue.

        Args:
            params: Close parameters

        Returns:
            List containing closed issue
        """
        args = ["close", params.issue_id, "--reason", params.reason]

        data = await self._run_command(*args)
        if not isinstance(data, list):
            raise BdCommandError(f"Invalid response for close {params.issue_id}")

        return [Issue.model_validate(issue) for issue in data]

    async def reopen(self, params: ReopenIssueParams) -> list[Issue]:
        """Reopen one or more closed issues.

        Args:
            params: Reopen parameters

        Returns:
            List of reopened issues
        """
        args = ["reopen", *params.issue_ids]

        if params.reason:
            args.extend(["--reason", params.reason])

        data = await self._run_command(*args)
        if not isinstance(data, list):
            raise BdCommandError(f"Invalid response for reopen {params.issue_ids}")

        return [Issue.model_validate(issue) for issue in data]

    async def add_dependency(self, params: AddDependencyParams) -> None:
        """Add a dependency between issues.

        Args:
            params: Dependency parameters
        """
        # issuegraph dep add doesn't return JSON, just prints confirmation
        cmd = [
            self.issuegraph_path,
            "dep",
            "add",
            params.issue_id,
            params.depends_on_id,
            "--type",
            params.dep_type,
            *self._global_flags(),
        ]

        # Set up environment with database configuration
        env = os.environ.copy()
        if self.issuegraph_dir:
            env["ISSUEGRAPH_DIR"] = self.issuegraph_dir
        elif self.issuegraph_db:
            env["ISSUEGRAPH_DB"] = self.issuegraph_db

        try:
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdin=asyncio.subprocess.DEVNULL,  # Prevent inheriting MCP's stdin
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=self._get_working_dir(),
                env=env,
            )
            _stdout, stderr = await process.communicate()
        except FileNotFoundError as e:
            raise BdNotFoundError(BdNotFoundError.installation_message(self.issuegraph_path)) from e

        if process.returncode != 0:
            raise BdCommandError(
                f"issuegraph dep add failed: {stderr.decode()}",
                stderr=stderr.decode(),
                returncode=process.returncode or 1,
            )

    async def _run_text_command(self, *args: str) -> str:
        """Run a issuegraph command that returns plain text (not JSON) and return stdout.

        Used for subcommands like `comment`/`note` that print a confirmation
        line rather than JSON. Mirrors _run_command's env and error handling.
        """
        cmd = [self.issuegraph_path, *args, *self._global_flags()]

        env = os.environ.copy()
        if self.issuegraph_dir:
            env["ISSUEGRAPH_DIR"] = self.issuegraph_dir
        elif self.issuegraph_db:
            env["ISSUEGRAPH_DB"] = self.issuegraph_db

        try:
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdin=asyncio.subprocess.DEVNULL,  # Prevent inheriting MCP's stdin
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=self._get_working_dir(),
                env=env,
            )
            stdout, stderr = await process.communicate()
        except FileNotFoundError as e:
            raise BdNotFoundError(BdNotFoundError.installation_message(self.issuegraph_path)) from e

        if process.returncode != 0:
            raise BdCommandError(
                f"issuegraph {args[0]} failed: {stderr.decode()}",
                stderr=stderr.decode(),
                returncode=process.returncode or 1,
            )

        return stdout.decode().strip()

    async def add_comment(self, params: AddCommentParams) -> str:
        """Add a comment to an issue via `issuegraph comment <id> <text>`.

        Args:
            params: Comment parameters (issue_id, text)

        Returns:
            Confirmation message
        """
        await self._run_text_command("comment", params.issue_id, params.text)
        return f"Added comment to {params.issue_id}"

    async def list_comments(self, params: ListCommentsParams) -> list[Comment]:
        """List comments on an issue via `issuegraph comments <id>`.

        Args:
            params: Parameters containing the issue_id

        Returns:
            List of comments in chronological order
        """
        data = await self._run_command("comments", params.issue_id)
        if not isinstance(data, list):
            return []
        return [Comment.model_validate(comment) for comment in data]

    async def add_note(self, params: AddNoteParams) -> str:
        """Append a note to an issue's notes field via `issuegraph note <id> <text>`.

        Args:
            params: Note parameters (issue_id, text)

        Returns:
            Confirmation message
        """
        await self._run_text_command("note", params.issue_id, params.text)
        return f"Appended note to {params.issue_id}"

    async def quickstart(self) -> str:
        """Get issuegraph quickstart guide.

        Returns:
            Quickstart guide text
        """
        cmd = [self.issuegraph_path, "quickstart"]

        try:
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdin=asyncio.subprocess.DEVNULL,  # Prevent inheriting MCP's stdin
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=self._get_working_dir(),
            )
            stdout, stderr = await process.communicate()
        except FileNotFoundError as e:
            raise BdNotFoundError(BdNotFoundError.installation_message(self.issuegraph_path)) from e

        if process.returncode != 0:
            raise BdCommandError(
                f"issuegraph quickstart failed: {stderr.decode()}",
                stderr=stderr.decode(),
                returncode=process.returncode or 1,
            )

        return stdout.decode()

    async def stats(self) -> Stats:
        """Get statistics about issues.

        Returns:
            Statistics object
        """
        data = await self._run_command("stats")
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for stats")

        return Stats.model_validate(data)

    async def blocked(self, params: BlockedParams | None = None) -> list[BlockedIssue]:
        """Get blocked issues.

        Args:
            params: Query parameters

        Returns:
            List of blocked issues with blocking information
        """
        params = params or BlockedParams()
        args = ["blocked"]
        if params.parent_id:
            args.extend(["--parent", params.parent_id])

        data = await self._run_command(*args)
        if not isinstance(data, list):
            return []

        return [BlockedIssue.model_validate(_sanitize_issue_deps(issue)) for issue in data]

    async def inspect_migration(self) -> dict[str, Any]:
        """Get migration plan and database state for agent analysis.

        Returns:
            Migration plan dict with registered_migrations, warnings, etc.
        """
        data = await self._run_command("migrate", "--inspect")
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for inspect_migration")
        return data

    async def get_schema_info(self) -> dict[str, Any]:
        """Get current database schema for inspection.

        Returns:
            Schema info dict with tables, version, config, sample IDs, etc.
        """
        data = await self._run_command("info", "--schema")
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for get_schema_info")
        return data

    async def repair_deps(self, fix: bool = False) -> dict[str, Any]:
        """Find and optionally fix orphaned dependency references.

        Args:
            fix: If True, automatically remove orphaned dependencies

        Returns:
            Dict with orphans_found, orphans list, and fixed count if fix=True
        """
        args = ["repair-deps"]
        if fix:
            args.append("--fix")

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for repair-deps")
        return data

    async def detect_pollution(self, clean: bool = False) -> dict[str, Any]:
        """Detect test issues that leaked into production database.

        Args:
            clean: If True, delete detected test issues

        Returns:
            Dict with detected test issues and deleted count if clean=True
        """
        args = ["doctor", "--check=pollution"]
        if clean:
            args.extend(["--clean", "--yes"])

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for doctor --check=pollution")
        return data

    async def validate(self, checks: str | None = None, fix_all: bool = False) -> dict[str, Any]:
        """Run database validation checks via issuegraph doctor --check=validate.

        Args:
            checks: Accepted for backward compatibility but no longer applied.
                issuegraph doctor --check=validate runs the full data-integrity suite
                (duplicates, orphaned issues, conflicts) and has no flag for
                selecting a subset. Use detect_pollution() for the pollution check.
            fix_all: If True, auto-fix all fixable issues

        Returns:
            Dict with validation results for each check
        """
        del checks  # subset selection is unsupported by issuegraph doctor (see docstring)
        args = ["doctor", "--check=validate"]
        if fix_all:
            args.append("--fix")
            args.append("--yes")

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for doctor --check=validate")
        return data

    async def init(self, params: InitParams | None = None) -> str:
        """Initialize issuegraph in current directory.

        Args:
            params: Initialization parameters

        Returns:
            Initialization output message
        """
        params = params or InitParams()
        cmd = [self.issuegraph_path, "init"]

        if params.prefix:
            cmd.extend(["--prefix", params.prefix])

        # NOTE: Do NOT add --db flag for init!
        # init creates a NEW database in the current directory.
        # Only add actor-related flags.
        if self.actor:
            cmd.extend(["--actor", self.actor])

        try:
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdin=asyncio.subprocess.DEVNULL,  # Prevent inheriting MCP's stdin
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=self._get_working_dir(),
            )
            stdout, stderr = await process.communicate()
        except FileNotFoundError as e:
            raise BdNotFoundError(BdNotFoundError.installation_message(self.issuegraph_path)) from e

        if process.returncode != 0:
            raise BdCommandError(
                f"issuegraph init failed: {stderr.decode()}",
                stderr=stderr.decode(),
                returncode=process.returncode or 1,
            )

        return stdout.decode()


# Backwards compatibility alias
BdClient = BdCliClient


def create_issuegraph_client(
    prefer_daemon: bool = False,
    issuegraph_path: str | None = None,
    issuegraph_dir: str | None = None,
    issuegraph_db: str | None = None,
    actor: str | None = None,
    no_auto_flush: bool | None = None,
    no_auto_import: bool | None = None,
    working_dir: str | None = None,
) -> BdClientBase:
    """Create a issuegraph CLI client.

    Args:
        prefer_daemon: Deprecated, ignored. Kept for API compatibility.
        issuegraph_path: Path to issuegraph executable
        issuegraph_dir: Path to .issuegraph directory
        issuegraph_db: Path to issuegraph database (deprecated)
        actor: Actor name for audit trail
        no_auto_flush: Disable auto-flush
        no_auto_import: Disable auto-import
        working_dir: Working directory for database discovery

    Returns:
        BdClientBase implementation (CLI)
    """
    return BdCliClient(
        issuegraph_path=issuegraph_path,
        issuegraph_dir=issuegraph_dir,
        issuegraph_db=issuegraph_db,
        actor=actor,
        no_auto_flush=no_auto_flush,
        no_auto_import=no_auto_import,
        working_dir=working_dir,
    )

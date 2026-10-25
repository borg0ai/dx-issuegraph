"""Real integration tests for BdClient using actual issuegraph binary."""

import os
import shutil
import subprocess
from pathlib import Path

import pytest

from issuegraph_mcp.issuegraph_client import BdClient, BdCommandError
from issuegraph_mcp.models import (
    AddCommentParams,
    AddDependencyParams,
    AddNoteParams,
    ClaimIssueParams,
    CloseIssueParams,
    CreateIssueParams,
    ListCommentsParams,
    ListIssuesParams,
    ReadyWorkParams,
    ReopenIssueParams,
    ShowIssueParams,
    UpdateIssueParams,
)


@pytest.fixture(scope="session")
def issuegraph_executable():
    """Verify issuegraph is available in PATH."""
    issuegraph_path = shutil.which("issuegraph")
    if not issuegraph_path:
        pytest.fail(
            "issuegraph executable not found in PATH. "
            "Please install issuegraph or add it to your PATH before running integration tests."
        )
    return issuegraph_path


@pytest.fixture(scope="session")
def issuegraph_init_isolation_flags_supported(issuegraph_executable):
    """Skip integration tests when issuegraph lacks the init flags these tests require."""
    result = subprocess.run(
        [issuegraph_executable, "init", "--help"],
        capture_output=True,
        text=True,
        check=False,
    )
    help_text = f"{result.stdout}\n{result.stderr}"
    required_flags = ("--non-interactive", "--skip-agents", "--skip-hooks")
    missing_flags = [flag for flag in required_flags if flag not in help_text]
    if missing_flags:
        pytest.skip(
            "issuegraph init does not support required test isolation flags: "
            + ", ".join(missing_flags)
            + ". Install a current issuegraph from this repository before running integration tests."
        )


@pytest.fixture
def temp_workspace(tmp_path: Path) -> Path:
    """Create a per-test workspace isolated with its own ISSUEGRAPH_DIR."""
    workspace = tmp_path / "workspace"
    workspace.mkdir()
    return workspace


@pytest.fixture
async def issuegraph_client(issuegraph_executable, temp_workspace, monkeypatch, issuegraph_init_isolation_flags_supported):
    """Create BdClient with a temporary Dolt workspace - fully hermetic."""
    issuegraph_dir = temp_workspace / ".issuegraph"
    monkeypatch.delenv("ISSUEGRAPH_DB", raising=False)
    monkeypatch.delenv("ISSUEGRAPH_DIR", raising=False)
    monkeypatch.delenv("ISSUEGRAPH_DB", raising=False)
    monkeypatch.delenv("ISSUEGRAPH_WORKING_DIR", raising=False)

    client = BdClient(
        issuegraph_path=issuegraph_executable,
        issuegraph_dir=str(issuegraph_dir),
        issuegraph_db="",
        working_dir=str(temp_workspace),
    )

    env = os.environ.copy()
    env["ISSUEGRAPH_DIR"] = str(issuegraph_dir)
    env["ISSUEGRAPH_NON_INTERACTIVE"] = "1"

    import asyncio

    process = await asyncio.create_subprocess_exec(
        issuegraph_executable,
        "init",
        "--prefix",
        "test",
        "--non-interactive",
        "--skip-agents",
        "--skip-hooks",
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
        env=env,
        cwd=temp_workspace,
    )
    _stdout, stderr = await process.communicate()

    if process.returncode != 0:
        pytest.fail(f"Failed to initialize test database: {stderr.decode()}")

    yield client


@pytest.mark.asyncio
async def test_create_and_show_issue(issuegraph_client):
    """Test creating and showing an issue with real issuegraph."""
    # Create issue
    params = CreateIssueParams(
        title="Test integration issue",
        description="This is a real integration test",
        priority=1,
        issue_type="bug",
    )
    created = await issuegraph_client.create(params)

    assert created.id is not None
    assert created.title == "Test integration issue"
    assert created.description == "This is a real integration test"
    assert created.priority == 1
    assert created.issue_type == "bug"
    assert created.status == "open"

    # Show issue
    show_params = ShowIssueParams(issue_id=created.id)
    shown = await issuegraph_client.show(show_params)

    assert shown.id == created.id
    assert shown.title == created.title
    assert shown.description == created.description


@pytest.mark.asyncio
async def test_list_issues(issuegraph_client):
    """Test listing issues with real issuegraph."""
    # Create multiple issues
    for i in range(3):
        params = CreateIssueParams(
            title=f"Test issue {i}",
            priority=i,
            issue_type="task",
        )
        await issuegraph_client.create(params)

    # List all issues
    list_params = ListIssuesParams()
    issues = await issuegraph_client.list_issues(list_params)

    assert len(issues) >= 3

    # List with status filter
    list_params_filtered = ListIssuesParams(status="open")
    issues = await issuegraph_client.list_issues(list_params_filtered)

    assert all(issue.status == "open" for issue in issues)


@pytest.mark.asyncio
async def test_update_issue(issuegraph_client):
    """Test updating an issue with real issuegraph."""
    # Create issue
    create_params = CreateIssueParams(
        title="Issue to update",
        priority=2,
        issue_type="feature",
    )
    created = await issuegraph_client.create(create_params)

    # Update issue
    update_params = UpdateIssueParams(
        issue_id=created.id,
        status="blocked",
        priority=0,
        title="Updated title",
    )
    updated = await issuegraph_client.update(update_params)

    assert updated.id == created.id
    assert updated.status == "blocked"
    assert updated.priority == 0
    assert updated.title == "Updated title"


@pytest.mark.asyncio
async def test_claim_issue(issuegraph_client):
    """Test atomic claim with real issuegraph."""
    created = await issuegraph_client.create(CreateIssueParams(title="Issue to claim", priority=2, issue_type="task"))

    claimed = await issuegraph_client.claim(ClaimIssueParams(issue_id=created.id))

    assert claimed.id == created.id
    assert claimed.status == "in_progress"
    assert claimed.assignee is not None


@pytest.mark.asyncio
async def test_close_issue(issuegraph_client):
    """Test closing an issue with real issuegraph."""
    # Create issue
    create_params = CreateIssueParams(
        title="Issue to close",
        priority=1,
        issue_type="bug",
    )
    created = await issuegraph_client.create(create_params)

    # Close issue
    close_params = CloseIssueParams(issue_id=created.id, reason="Testing complete")
    closed_issues = await issuegraph_client.close(close_params)

    assert len(closed_issues) >= 1
    closed = closed_issues[0]
    assert closed.id == created.id
    assert closed.status == "closed"
    assert closed.closed_at is not None


@pytest.mark.asyncio
async def test_reopen_issue(issuegraph_client):
    """Test reopening a closed issue with real issuegraph."""
    # Create issue
    create_params = CreateIssueParams(
        title="BG's issue to reopen",
        priority=1,
        issue_type="bug",
    )
    created = await issuegraph_client.create(create_params)

    # Close issue
    close_params = CloseIssueParams(issue_id=created.id, reason="Testing complete")
    await issuegraph_client.close(close_params)

    # Reopen issue
    reopen_params = ReopenIssueParams(issue_ids=[created.id])
    reopened_issues = await issuegraph_client.reopen(reopen_params)

    assert len(reopened_issues) >= 1
    reopened = reopened_issues[0]
    assert reopened.id == created.id
    assert reopened.status == "open"
    assert reopened.closed_at is None


@pytest.mark.asyncio
async def test_reopen_multiple_issues(issuegraph_client):
    """Test reopening multiple closed issues with real issuegraph."""
    # Create and close two issues
    issue1 = await issuegraph_client.create(CreateIssueParams(title="Issue 1 to reopen", priority=1, issue_type="task"))
    issue2 = await issuegraph_client.create(CreateIssueParams(title="Issue 2 to reopen", priority=1, issue_type="task"))

    await issuegraph_client.close(CloseIssueParams(issue_id=issue1.id, reason="Done"))
    await issuegraph_client.close(CloseIssueParams(issue_id=issue2.id, reason="Done"))

    # Reopen both issues
    reopen_params = ReopenIssueParams(issue_ids=[issue1.id, issue2.id])
    reopened_issues = await issuegraph_client.reopen(reopen_params)

    assert len(reopened_issues) == 2
    reopened_ids = {issue.id for issue in reopened_issues}
    assert issue1.id in reopened_ids
    assert issue2.id in reopened_ids
    assert all(issue.status == "open" for issue in reopened_issues)
    assert all(issue.closed_at is None for issue in reopened_issues)


@pytest.mark.asyncio
async def test_reopen_with_reason(issuegraph_client):
    """Test reopening an issue with reason parameter."""
    # Create and close issue
    created = await issuegraph_client.create(
        CreateIssueParams(title="Issue to reopen with reason", priority=1, issue_type="bug")
    )
    await issuegraph_client.close(CloseIssueParams(issue_id=created.id, reason="Done"))

    # Reopen with reason
    reopen_params = ReopenIssueParams(issue_ids=[created.id], reason="BG found a regression in production")
    reopened_issues = await issuegraph_client.reopen(reopen_params)

    assert len(reopened_issues) >= 1
    reopened = reopened_issues[0]
    assert reopened.id == created.id
    assert reopened.status == "open"
    assert reopened.closed_at is None


@pytest.mark.asyncio
async def test_add_dependency(issuegraph_client):
    """Test adding dependencies with real issuegraph."""
    # Create two issues
    issue1 = await issuegraph_client.create(CreateIssueParams(title="Issue 1", priority=1, issue_type="task"))
    issue2 = await issuegraph_client.create(CreateIssueParams(title="Issue 2", priority=1, issue_type="task"))

    # Add dependency: issue2 blocks issue1
    params = AddDependencyParams(issue_id=issue1.id, depends_on_id=issue2.id, dep_type="blocks")
    await issuegraph_client.add_dependency(params)

    # Verify dependency by showing issue1
    show_params = ShowIssueParams(issue_id=issue1.id)
    shown = await issuegraph_client.show(show_params)

    assert len(shown.dependencies) > 0
    assert any(dep.id == issue2.id for dep in shown.dependencies)


@pytest.mark.asyncio
async def test_comments_and_notes(issuegraph_client):
    """Test comment/comments/note round-trip with real issuegraph."""
    issue = await issuegraph_client.create(CreateIssueParams(title="Comment target", priority=1, issue_type="task"))

    # Add two comments and read them back
    result = await issuegraph_client.add_comment(AddCommentParams(issue_id=issue.id, text="First comment"))
    assert issue.id in result
    await issuegraph_client.add_comment(AddCommentParams(issue_id=issue.id, text="Second comment"))

    comments = await issuegraph_client.list_comments(ListCommentsParams(issue_id=issue.id))
    assert len(comments) >= 2
    texts = [c.text for c in comments]
    assert "First comment" in texts
    assert "Second comment" in texts
    assert all(c.issue_id == issue.id for c in comments)
    # Chronological order: first added comes first
    assert texts.index("First comment") < texts.index("Second comment")

    # Append a note and verify it lands in the notes field
    note_result = await issuegraph_client.add_note(AddNoteParams(issue_id=issue.id, text="A durable note"))
    assert issue.id in note_result

    shown = await issuegraph_client.show(ShowIssueParams(issue_id=issue.id))
    assert "A durable note" in (shown.notes or "")


@pytest.mark.asyncio
async def test_ready_work(issuegraph_client):
    """Test getting ready work with real issuegraph."""
    # Create issue with no dependencies (should be ready)
    ready_issue = await issuegraph_client.create(CreateIssueParams(title="Ready issue", priority=1, issue_type="task"))

    # Create blocked issue
    blocking_issue = await issuegraph_client.create(
        CreateIssueParams(title="Blocking issue", priority=1, issue_type="task")
    )
    blocked_issue = await issuegraph_client.create(CreateIssueParams(title="Blocked issue", priority=1, issue_type="task"))

    # Add blocking dependency
    await issuegraph_client.add_dependency(
        AddDependencyParams(
            issue_id=blocked_issue.id,
            depends_on_id=blocking_issue.id,
            dep_type="blocks",
        )
    )

    # Get ready work
    params = ReadyWorkParams(limit=100)
    ready_issues = await issuegraph_client.ready(params)

    # ready_issue should be in ready work
    ready_ids = [issue.id for issue in ready_issues]
    assert ready_issue.id in ready_ids

    # blocked_issue should NOT be in ready work
    assert blocked_issue.id not in ready_ids


@pytest.mark.asyncio
async def test_quickstart(issuegraph_client):
    """Test quickstart command with real issuegraph."""
    result = await issuegraph_client.quickstart()

    assert len(result) > 0
    assert "issuegraph" in result.lower() or "issuegraph" in result.lower()


@pytest.mark.asyncio
async def test_create_with_labels(issuegraph_client):
    """Test creating issue with labels."""
    params = CreateIssueParams(
        title="Issue with labels",
        priority=1,
        issue_type="feature",
        labels=["urgent", "backend"],
    )
    created = await issuegraph_client.create(params)

    # Note: issuegraph currently doesn't return labels in JSON output
    # This test verifies the command succeeds with labels parameter
    assert created.id is not None
    assert created.title == "Issue with labels"


@pytest.mark.asyncio
async def test_create_with_assignee(issuegraph_client):
    """Test creating issue with assignee."""
    params = CreateIssueParams(
        title="Assigned issue",
        priority=1,
        issue_type="task",
        assignee="testuser",
    )
    created = await issuegraph_client.create(params)

    assert created.assignee == "testuser"


@pytest.mark.asyncio
async def test_list_with_filters(issuegraph_client):
    """Test listing issues with multiple filters."""
    # Create issues with different attributes
    await issuegraph_client.create(
        CreateIssueParams(
            title="Bug P0",
            priority=0,
            issue_type="bug",
            assignee="alice",
        )
    )
    await issuegraph_client.create(
        CreateIssueParams(
            title="Feature P1",
            priority=1,
            issue_type="feature",
            assignee="bob",
        )
    )

    # Filter by priority
    params = ListIssuesParams(priority=0)
    issues = await issuegraph_client.list_issues(params)
    assert all(issue.priority == 0 for issue in issues)

    # Filter by type
    params = ListIssuesParams(issue_type="bug")
    issues = await issuegraph_client.list_issues(params)
    assert all(issue.issue_type == "bug" for issue in issues)

    # Filter by assignee
    params = ListIssuesParams(assignee="alice")
    issues = await issuegraph_client.list_issues(params)
    assert all(issue.assignee == "alice" for issue in issues)


@pytest.mark.asyncio
async def test_invalid_issue_id(issuegraph_client):
    """Test showing non-existent issue."""
    params = ShowIssueParams(issue_id="test-999")

    with pytest.raises(BdCommandError, match="issuegraph command failed"):
        await issuegraph_client.show(params)


@pytest.mark.asyncio
async def test_dependency_types(issuegraph_client):
    """Test different dependency types."""
    issue1 = await issuegraph_client.create(CreateIssueParams(title="Issue 1", priority=1, issue_type="task"))
    issue2 = await issuegraph_client.create(CreateIssueParams(title="Issue 2", priority=1, issue_type="task"))

    # Test related dependency
    params = AddDependencyParams(issue_id=issue1.id, depends_on_id=issue2.id, dep_type="related")
    await issuegraph_client.add_dependency(params)

    # Verify
    show_params = ShowIssueParams(issue_id=issue1.id)
    shown = await issuegraph_client.show(show_params)
    assert len(shown.dependencies) > 0


@pytest.mark.asyncio
async def test_init_creates_issuegraph_directory(
    issuegraph_executable, tmp_path, monkeypatch, issuegraph_init_isolation_flags_supported
):
    """Test that init creates .issuegraph directory in current working directory.

    This is a critical test for the bug where init was using --db flag
    and creating the database in the wrong location.
    """
    from issuegraph_mcp.issuegraph_client import BdClient
    from issuegraph_mcp.models import InitParams

    temp_path = tmp_path / "init-workspace"
    temp_path.mkdir()
    issuegraph_dir = temp_path / ".issuegraph"
    monkeypatch.setenv("ISSUEGRAPH_NON_INTERACTIVE", "1")
    monkeypatch.delenv("ISSUEGRAPH_DIR", raising=False)
    monkeypatch.delenv("ISSUEGRAPH_DB", raising=False)

    # Ensure .issuegraph doesn't exist yet
    assert not issuegraph_dir.exists()

    # Create client WITHOUT issuegraph_db set and WITH working_dir set to temp_dir
    client = BdClient(
        issuegraph_path=issuegraph_executable,
        issuegraph_dir=str(issuegraph_dir),
        issuegraph_db=None,
        working_dir=str(temp_path),
    )

    # Initialize with custom prefix (no need to chdir!)
    params = InitParams(prefix="test")
    result = await client.init(params)

    # Verify .issuegraph directory was created in temp directory
    assert issuegraph_dir.exists(), f".issuegraph directory not created in {temp_path}"
    assert issuegraph_dir.is_dir(), ".issuegraph exists but is not a directory"

    # Verify Dolt backend files were created under the isolated .issuegraph directory.
    assert (issuegraph_dir / "metadata.json").is_file()
    assert (issuegraph_dir / "config.yaml").is_file()
    embedded_dir = issuegraph_dir / "embeddeddolt"
    assert embedded_dir.exists(), "No embedded Dolt directory created in .issuegraph/"
    assert any(path.name == ".dolt" for path in embedded_dir.glob("*/.dolt")), (
        f"Expected an embedded Dolt database under {embedded_dir}"
    )

    # Verify success message
    assert "initialized" in result.lower() or "created" in result.lower()

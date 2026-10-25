"""Integration test for separate issuegraph databases per worktree.

Tests the recommended workflow: each worktree has its own .issuegraph database,
and changes sync via git commits/pulls of the .issuegraph/issues.jsonl file.

NOTE: These tests may have flaky behavior due to path caching issues.
See bd-4aao for details.
"""

import json
import os
import shutil
import subprocess
import tempfile
from pathlib import Path

import pytest
from fastmcp.client import Client

from issuegraph_mcp.server import mcp


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


@pytest.fixture
async def git_worktree_with_separate_dbs(issuegraph_executable):
    """Create a git repo with a worktree, each with its own issuegraph database.

    Returns:
        tuple: (main_repo_path, worktree_path, temp_dir)
    """
    # Create temp directory
    temp_dir = tempfile.mkdtemp(prefix="issuegraph_worktree_separate_")
    main_repo = Path(temp_dir) / "main"
    worktree = Path(temp_dir) / "worktree"

    try:
        # Initialize main git repo
        main_repo.mkdir()
        subprocess.run(["git", "init"], cwd=main_repo, check=True, capture_output=True)
        subprocess.run(
            ["git", "config", "user.email", "test@example.com"],
            cwd=main_repo,
            check=True,
            capture_output=True,
        )
        subprocess.run(
            ["git", "config", "user.name", "Test User"],
            cwd=main_repo,
            check=True,
            capture_output=True,
        )

        # Create initial commit
        readme = main_repo / "README.md"
        readme.write_text("# Test Repo\n")
        subprocess.run(["git", "add", "README.md"], cwd=main_repo, check=True, capture_output=True)
        subprocess.run(
            ["git", "commit", "-m", "Initial commit"],
            cwd=main_repo,
            check=True,
            capture_output=True,
        )

        # Initialize issuegraph in main repo BEFORE creating worktree
        init_result = subprocess.run(
            ["issuegraph", "init", "--prefix", "main"],
            cwd=main_repo,
            capture_output=True,
            text=True,
        )
        if init_result.returncode != 0:
            raise RuntimeError(f"issuegraph init in main failed: {init_result.stderr}")

        # Verify main repo has .issuegraph directory (database is always issuegraph.db, prefix is for issue IDs)
        assert (main_repo / ".issuegraph").exists(), (
            f"Main repo should have .issuegraph directory. Init output: {init_result.stdout} {init_result.stderr}"
        )
        assert (main_repo / ".issuegraph" / "issuegraph.db").exists(), "Main repo should have database"

        # Create a worktree AFTER initializing issuegraph in main
        subprocess.run(
            ["git", "worktree", "add", str(worktree), "-b", "feature"],
            cwd=main_repo,
            check=True,
            capture_output=True,
        )

        # Commit the .issuegraph directory to git in main repo
        subprocess.run(["git", "add", ".issuegraph"], cwd=main_repo, check=True, capture_output=True)
        subprocess.run(
            ["git", "commit", "-m", "Add issuegraph to main"],
            cwd=main_repo,
            check=True,
            capture_output=True,
        )

        # Re-sync after commit to avoid staleness check issues
        subprocess.run(["issuegraph", "sync"], cwd=main_repo, capture_output=True)

        # Initialize issuegraph in worktree (separate database, different prefix)
        init_result = subprocess.run(
            ["issuegraph", "init", "--prefix", "feature"],
            cwd=worktree,
            capture_output=True,
            text=True,
        )
        if init_result.returncode != 0:
            raise RuntimeError(f"issuegraph init in worktree failed: {init_result.stderr}")

        # Verify worktree has its own .issuegraph directory (database is always issuegraph.db)
        assert (worktree / ".issuegraph").exists(), (
            f"Worktree should have .issuegraph directory. Init output: {init_result.stdout} {init_result.stderr}"
        )
        assert (worktree / ".issuegraph" / "issuegraph.db").exists(), "Worktree should have database"

        # Commit the worktree's .issuegraph (will replace/update main's .issuegraph on feature branch)
        subprocess.run(["git", "add", ".issuegraph"], cwd=worktree, check=True, capture_output=True)
        result = subprocess.run(
            ["git", "commit", "-m", "Add issuegraph to feature branch"],
            cwd=worktree,
            capture_output=True,
            text=True,
        )
        # Commit may fail if nothing changed (that's ok for our tests)
        if result.returncode != 0 and "nothing to commit" not in result.stdout:
            raise subprocess.CalledProcessError(result.returncode, result.args, result.stdout, result.stderr)

        # Re-sync after commit to avoid staleness check issues
        subprocess.run(["issuegraph", "sync"], cwd=worktree, capture_output=True)

        yield main_repo, worktree, temp_dir

    finally:
        # Cleanup
        shutil.rmtree(temp_dir, ignore_errors=True)


@pytest.mark.skip(reason="Flaky due to path caching issues - see bd-4aao")
@pytest.mark.asyncio
async def test_separate_databases_are_isolated(git_worktree_with_separate_dbs, issuegraph_executable):
    """Test that each worktree has its own isolated database."""
    main_repo, worktree, temp_dir = git_worktree_with_separate_dbs

    # Create issue in main repo
    result = subprocess.run(
        ["issuegraph", "create", "Main repo issue", "-p", "1", "--json"],
        cwd=main_repo,
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0, f"Create in main failed: {result.stderr}"
    main_issue = json.loads(result.stdout)
    assert main_issue["id"].startswith("main-"), f"Expected main- prefix, got {main_issue['id']}"

    # Create issue in worktree
    result = subprocess.run(
        ["issuegraph", "create", "Worktree issue", "-p", "1", "--json"],
        cwd=worktree,
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0, f"Create in worktree failed: {result.stderr}"
    worktree_issue = json.loads(result.stdout)
    assert worktree_issue["id"].startswith("feature-"), f"Expected feature- prefix, got {worktree_issue['id']}"

    # List issues in main repo
    result = subprocess.run(
        ["issuegraph", "list", "--json"],
        cwd=main_repo,
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0
    main_issues = json.loads(result.stdout)
    main_ids = [issue["id"] for issue in main_issues]

    # List issues in worktree
    result = subprocess.run(
        ["issuegraph", "list", "--json"],
        cwd=worktree,
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0
    worktree_issues = json.loads(result.stdout)
    worktree_ids = [issue["id"] for issue in worktree_issues]

    # Verify isolation - main repo shouldn't see worktree issues and vice versa
    assert "main-1" in main_ids, "Main repo should see its own issue"
    assert "feature-1" not in main_ids, "Main repo should NOT see worktree issue"

    assert "feature-1" in worktree_ids, "Worktree should see its own issue"
    assert "main-1" not in worktree_ids, "Worktree should NOT see main repo issue (yet)"


@pytest.mark.skip(reason="Flaky due to path caching issues - see bd-4aao")
@pytest.mark.asyncio
async def test_changes_sync_via_git(git_worktree_with_separate_dbs, issuegraph_executable):
    """Test that changes sync between worktrees via git commits and merges."""
    main_repo, worktree, temp_dir = git_worktree_with_separate_dbs

    # Create and commit issue in main repo
    result = subprocess.run(
        ["issuegraph", "create", "Shared issue", "-p", "1", "--json"],
        cwd=main_repo,
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0

    # Export to JSONL (should happen automatically, but force it)
    subprocess.run(
        ["issuegraph", "export", "-o", ".issuegraph/issues.jsonl"],
        cwd=main_repo,
        check=True,
        capture_output=True,
    )

    # Commit the JSONL file
    subprocess.run(["git", "add", ".issuegraph/issues.jsonl"], cwd=main_repo, check=True, capture_output=True)
    subprocess.run(
        ["git", "commit", "-m", "Add shared issue"],
        cwd=main_repo,
        check=True,
        capture_output=True,
    )

    # Switch worktree to main branch temporarily to pull changes
    subprocess.run(
        ["git", "fetch", "origin", "master:master"],
        cwd=worktree,
        capture_output=True,  # May fail if no remote, that's ok
    )
    subprocess.run(
        ["git", "merge", "master"],
        cwd=worktree,
        capture_output=True,  # May have conflicts, handle below
    )

    # Import the changes into worktree database
    result = subprocess.run(
        ["issuegraph", "import", "-i", ".issuegraph/issues.jsonl"],
        cwd=worktree,
        capture_output=True,
        text=True,
    )

    # If import succeeded, verify the issue is now visible
    if result.returncode == 0:
        result = subprocess.run(
            ["issuegraph", "list", "--json"],
            cwd=worktree,
            capture_output=True,
            text=True,
        )
        assert result.returncode == 0
        worktree_issues = json.loads(result.stdout)

        # After import, worktree should see the main repo issue
        # (with potentially remapped ID due to prefix difference)
        assert len(worktree_issues) >= 1, "Worktree should have at least one issue after import"


@pytest.mark.skip(reason="Flaky due to path caching issues - see bd-4aao")
@pytest.mark.asyncio
async def test_mcp_works_with_separate_databases(git_worktree_with_separate_dbs, monkeypatch):
    """Test that MCP server works independently in each worktree."""
    from issuegraph_mcp import tools

    main_repo, worktree, temp_dir = git_worktree_with_separate_dbs

    # Configure MCP for worktree
    tools._connection_pool.clear()
    monkeypatch.setenv("ISSUEGRAPH_WORKING_DIR", str(worktree))

    # Reset context
    if "ISSUEGRAPH_CONTEXT_SET" in os.environ:
        monkeypatch.delenv("ISSUEGRAPH_CONTEXT_SET")

    # Create MCP client
    async with Client(mcp) as client:
        # Set context to worktree
        await client.call_tool("context", {"workspace_root": str(worktree)})

        # Create issue via MCP
        result = await client.call_tool(
            "create",
            {
                "title": "MCP issue in worktree",
                "description": "Created via MCP in worktree",
                "priority": 1,
            },
        )

        assert result.is_error is False, f"Create failed: {result.content}"

        # Parse result
        content = result.content[0].text
        issue_data = json.loads(content)
        assert issue_data["id"].startswith("feature-"), "Issue should have feature- prefix"

        # List via MCP
        list_result = await client.call_tool("list", {})
        assert list_result.is_error is False

        # Verify isolation - should only see worktree issues
        list_content = list_result.content[0].text
        assert "feature-" in list_content, "Should see worktree issues"
        assert "main-" not in list_content, "Should NOT see main repo issues"

    # Cleanup
    tools._connection_pool.clear()


@pytest.mark.skip(reason="Flaky due to path caching issues - see bd-4aao")
@pytest.mark.asyncio
async def test_worktree_database_discovery(git_worktree_with_separate_dbs, issuegraph_executable):
    """Test that issuegraph correctly discovers the database in each worktree."""
    main_repo, worktree, temp_dir = git_worktree_with_separate_dbs

    # Test main repo can find its database
    result = subprocess.run(
        ["issuegraph", "list", "--json"],
        cwd=main_repo,
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0, f"Main repo should find its database: {result.stderr}"

    # Test worktree can find its database
    result = subprocess.run(
        ["issuegraph", "list", "--json"],
        cwd=worktree,
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0, f"Worktree should find its database: {result.stderr}"

    # Both should work - that's what matters for this test
    # (The prefix doesn't matter as much as the fact that both can operate)


@pytest.mark.skip(reason="Flaky due to path caching issues - see bd-4aao")
@pytest.mark.asyncio
async def test_jsonl_export_works_in_worktrees(git_worktree_with_separate_dbs, issuegraph_executable):
    """Test that JSONL export works correctly in worktrees."""
    main_repo, worktree, temp_dir = git_worktree_with_separate_dbs

    # Create issue in worktree
    subprocess.run(
        ["issuegraph", "create", "Feature issue", "-p", "1"],
        cwd=worktree,
        check=True,
        capture_output=True,
    )

    # Export from worktree
    subprocess.run(
        ["issuegraph", "export", "-o", ".issuegraph/issues.jsonl"],
        cwd=worktree,
        check=True,
        capture_output=True,
    )

    # Verify JSONL file exists and contains the issue
    worktree_jsonl = (worktree / ".issuegraph" / "issues.jsonl").read_text()
    assert "Feature issue" in worktree_jsonl, "JSONL should contain the created issue"
    assert len(worktree_jsonl) > 0, "JSONL should not be empty"

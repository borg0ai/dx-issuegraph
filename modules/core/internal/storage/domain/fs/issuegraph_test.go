package fs

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/storage/domain"
	"github.com/borg0ai/dx-issuegraph/internal/utils"
)

func (s *testSuite) TestIssueGraphDirFSRepository() {
	s.Run("CreateIssueGraphDir", func() {
		s.Run("CreatesDirectoryWithPerms", s.createIssueGraphDirCreates)
		s.Run("IdempotentOnExisting", s.createIssueGraphDirIdempotent)
	})
	s.Run("IssueGraphDirExists", func() {
		s.Run("MissingReturnsFalse", s.issuegraphDirExistsMissing)
		s.Run("PresentReturnsTrue", s.issuegraphDirExistsPresent)
		s.Run("FileNotDirReturnsFalse", s.issuegraphDirExistsIsFile)
	})
	s.Run("WriteIssueGraphGitignore", func() {
		s.Run("WritesTemplate", s.writeIssueGraphGitignoreWrites)
		s.Run("IdempotentOnMatchingContent", s.writeIssueGraphGitignoreIdempotent)
		s.Run("AppendsMissingPatternsPreservingLocal", s.writeIssueGraphGitignoreAppendsMissing)
		s.Run("LeavesFileWithAllPatternsUntouched", s.writeIssueGraphGitignoreLeavesCoveredFile)
	})
	s.Run("IssueGraphGitignoreExists", func() {
		s.Run("MissingReturnsFalse", s.issuegraphGitignoreExistsMissing)
		s.Run("PresentReturnsTrue", s.issuegraphGitignoreExistsPresent)
	})
	s.Run("WriteProjectGitignore", func() {
		s.Run("CreatesWithHeaderAndPatterns", s.writeProjectGitignoreCreates)
		s.Run("AppendsToExisting", s.writeProjectGitignoreAppends)
		s.Run("SkipsAlreadyPresentPatterns", s.writeProjectGitignoreNoDuplicates)
		s.Run("NoChangeWhenAllPresent", s.writeProjectGitignoreNoOpWhenComplete)
		s.Run("AddsLeadingNewlineWhenMissing", s.writeProjectGitignoreFixesTrailingNewline)
	})
	s.Run("ProjectGitignoreExists", func() {
		s.Run("MissingReturnsFalse", s.projectGitignoreExistsMissing)
		s.Run("PresentReturnsTrue", s.projectGitignoreExistsPresent)
	})
	s.Run("WriteInteractionsLog", func() {
		s.Run("CreatesEmptyFile", s.writeInteractionsLogCreates)
		s.Run("PreservesExisting", s.writeInteractionsLogPreserves)
	})
	s.Run("WriteReadme", func() {
		s.Run("CreatesFromTemplate", s.writeReadmeCreates)
		s.Run("PreservesExisting", s.writeReadmePreserves)
	})
	s.Run("MetadataJSON", func() {
		s.Run("ReadMissingReturnsNil", s.readMetadataJSONMissing)
		s.Run("WriteThenReadRoundTrip", s.metadataJSONRoundTrip)
		s.Run("WriteOverwrites", s.metadataJSONOverwrite)
	})
	s.Run("ConfigYAML", func() {
		s.Run("ReadMissingReturnsNil", s.readConfigYAMLMissing)
		s.Run("WriteThenReadRoundTrip", s.configYAMLRoundTrip)
		s.Run("WriteOverwrites", s.configYAMLOverwrite)
	})
	s.Run("ReadIssueGraphConfig", func() {
		s.Run("MissingFileReturnsNil", s.readIssueGraphConfigMissing)
		s.Run("ParsesMetadataJSON", s.readIssueGraphConfigParses)
	})
	s.Run("ResolveIssueGraphDirPath", func() {
		s.Run("HonorsIssueGraphDirEnv", s.resolveIssueGraphDirPathHonorsEnv)
		s.Run("FallsBackToLocalDotIssueGraph", s.resolveIssueGraphDirPathLocalFallback)
	})
	s.Run("IssueGraphDirIsLocal", func() {
		s.Run("TrueForWorkDirChild", s.issuegraphDirIsLocalTrue)
		s.Run("FalseForExplicitEnv", s.issuegraphDirIsLocalFalseEnv)
	})
}

func (s *testSuite) createIssueGraphDirCreates() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(repo.CreateIssueGraphDir(s.Ctx()))

	info, err := os.Stat(issuegraphDir)
	s.Require().NoError(err)
	s.True(info.IsDir())

	if runtime.GOOS != "windows" {
		s.Equal(config.IssueGraphDirPerm, info.Mode().Perm())
	}
}

func (s *testSuite) createIssueGraphDirIdempotent() {
	_, _, repo := s.newRepo()
	s.Require().NoError(repo.CreateIssueGraphDir(s.Ctx()))
	s.Require().NoError(repo.CreateIssueGraphDir(s.Ctx()))
}

func (s *testSuite) issuegraphDirExistsMissing() {
	_, _, repo := s.newRepo()
	exists, err := repo.IssueGraphDirExists(s.Ctx())
	s.Require().NoError(err)
	s.False(exists)
}

func (s *testSuite) issuegraphDirExistsPresent() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	exists, err := repo.IssueGraphDirExists(s.Ctx())
	s.Require().NoError(err)
	s.True(exists)
}

func (s *testSuite) issuegraphDirExistsIsFile() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.WriteFile(issuegraphDir, []byte("not a dir"), 0600))
	exists, err := repo.IssueGraphDirExists(s.Ctx())
	s.Require().NoError(err)
	s.False(exists)
}

func (s *testSuite) writeIssueGraphGitignoreWrites() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	s.Require().NoError(repo.WriteIssueGraphGitignore(s.Ctx()))

	data, err := os.ReadFile(filepath.Join(issuegraphDir, ".gitignore"))
	s.Require().NoError(err)
	s.Equal(testTemplates().IssueGraphGitignore, string(data))
}

func (s *testSuite) writeIssueGraphGitignoreIdempotent() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	path := filepath.Join(issuegraphDir, ".gitignore")
	s.Require().NoError(os.WriteFile(path, []byte(testTemplates().IssueGraphGitignore), 0600))

	before, err := os.Stat(path)
	s.Require().NoError(err)

	s.Require().NoError(repo.WriteIssueGraphGitignore(s.Ctx()))

	after, err := os.Stat(path)
	s.Require().NoError(err)
	s.Equal(before.ModTime(), after.ModTime(), "file should not be rewritten when content already matches")
}

func (s *testSuite) writeIssueGraphGitignoreAppendsMissing() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	path := filepath.Join(issuegraphDir, ".gitignore")
	s.Require().NoError(os.WriteFile(path, []byte("!issues.jsonl\n"), 0600))

	s.Require().NoError(repo.WriteIssueGraphGitignore(s.Ctx()))

	data, err := os.ReadFile(path)
	s.Require().NoError(err)
	// bd-kaaz3: local rules survive; the template's pattern lines are appended.
	s.Contains(string(data), "!issues.jsonl")
	s.Contains(string(data), "dolt/")
	s.True(strings.HasPrefix(string(data), "!issues.jsonl\n"), "local content must stay at the top, got:\n%s", data)
}

func (s *testSuite) writeIssueGraphGitignoreLeavesCoveredFile() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	path := filepath.Join(issuegraphDir, ".gitignore")
	// Different from the template byte-wise, but every template pattern line
	// is present — nothing to append, the file must not be rewritten.
	local := "# my notes\ndolt/\n!issues.jsonl\n"
	s.Require().NoError(os.WriteFile(path, []byte(local), 0600))

	s.Require().NoError(repo.WriteIssueGraphGitignore(s.Ctx()))

	data, err := os.ReadFile(path)
	s.Require().NoError(err)
	s.Equal(local, string(data))
}

func (s *testSuite) issuegraphGitignoreExistsMissing() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	exists, err := repo.IssueGraphGitignoreExists(s.Ctx())
	s.Require().NoError(err)
	s.False(exists)
}

func (s *testSuite) issuegraphGitignoreExistsPresent() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	s.Require().NoError(os.WriteFile(filepath.Join(issuegraphDir, ".gitignore"), []byte("x"), 0600))
	exists, err := repo.IssueGraphGitignoreExists(s.Ctx())
	s.Require().NoError(err)
	s.True(exists)
}

func (s *testSuite) writeProjectGitignoreCreates() {
	workDir, _, repo := s.newRepo()
	s.Require().NoError(repo.WriteProjectGitignore(s.Ctx()))

	data, err := os.ReadFile(filepath.Join(workDir, ".gitignore"))
	s.Require().NoError(err)
	body := string(data)

	tpl := testTemplates()
	s.Contains(body, tpl.ProjectGitignoreHeader)
	for _, pattern := range tpl.ProjectGitignorePatterns {
		s.Contains(body, pattern)
	}
}

func (s *testSuite) writeProjectGitignoreAppends() {
	workDir, _, repo := s.newRepo()
	path := filepath.Join(workDir, ".gitignore")
	preexisting := "node_modules/\n*.log\n"
	s.Require().NoError(os.WriteFile(path, []byte(preexisting), 0644))

	s.Require().NoError(repo.WriteProjectGitignore(s.Ctx()))

	data, err := os.ReadFile(path)
	s.Require().NoError(err)
	body := string(data)

	tpl := testTemplates()
	s.True(strings.HasPrefix(body, preexisting), "preexisting content must be preserved at the top")
	s.Contains(body, tpl.ProjectGitignoreHeader)
	for _, pattern := range tpl.ProjectGitignorePatterns {
		s.Contains(body, pattern)
	}
}

func (s *testSuite) writeProjectGitignoreNoDuplicates() {
	workDir, _, repo := s.newRepo()
	path := filepath.Join(workDir, ".gitignore")
	preexisting := ".dolt/\nnode_modules/\n"
	s.Require().NoError(os.WriteFile(path, []byte(preexisting), 0644))

	s.Require().NoError(repo.WriteProjectGitignore(s.Ctx()))

	data, err := os.ReadFile(path)
	s.Require().NoError(err)

	s.Equal(1, strings.Count(string(data), ".dolt/\n"), "must not duplicate existing patterns")
}

func (s *testSuite) writeProjectGitignoreNoOpWhenComplete() {
	workDir, _, repo := s.newRepo()
	path := filepath.Join(workDir, ".gitignore")
	tpl := testTemplates()
	var buf bytes.Buffer
	buf.WriteString("existing\n")
	buf.WriteString(tpl.ProjectGitignoreHeader + "\n")
	for _, pattern := range tpl.ProjectGitignorePatterns {
		buf.WriteString(pattern + "\n")
	}
	s.Require().NoError(os.WriteFile(path, buf.Bytes(), 0644))

	before, err := os.Stat(path)
	s.Require().NoError(err)

	s.Require().NoError(repo.WriteProjectGitignore(s.Ctx()))

	after, err := os.Stat(path)
	s.Require().NoError(err)
	s.Equal(before.ModTime(), after.ModTime(), "must not rewrite when all patterns already present")
}

func (s *testSuite) writeProjectGitignoreFixesTrailingNewline() {
	workDir, _, repo := s.newRepo()
	path := filepath.Join(workDir, ".gitignore")
	s.Require().NoError(os.WriteFile(path, []byte("no-trailing-newline"), 0644))

	s.Require().NoError(repo.WriteProjectGitignore(s.Ctx()))

	data, err := os.ReadFile(path)
	s.Require().NoError(err)
	body := string(data)

	s.True(strings.HasPrefix(body, "no-trailing-newline\n"), "trailing newline must be inserted before appended content")
	s.Contains(body, testTemplates().ProjectGitignoreHeader)
}

func (s *testSuite) projectGitignoreExistsMissing() {
	_, _, repo := s.newRepo()
	exists, err := repo.ProjectGitignoreExists(s.Ctx())
	s.Require().NoError(err)
	s.False(exists)
}

func (s *testSuite) projectGitignoreExistsPresent() {
	workDir, _, repo := s.newRepo()
	s.Require().NoError(os.WriteFile(filepath.Join(workDir, ".gitignore"), []byte("x"), 0644))
	exists, err := repo.ProjectGitignoreExists(s.Ctx())
	s.Require().NoError(err)
	s.True(exists)
}

func (s *testSuite) writeInteractionsLogCreates() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	s.Require().NoError(repo.WriteInteractionsLog(s.Ctx()))

	data, err := os.ReadFile(filepath.Join(issuegraphDir, "interactions.jsonl"))
	s.Require().NoError(err)
	s.Empty(data)
}

func (s *testSuite) writeInteractionsLogPreserves() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	path := filepath.Join(issuegraphDir, "interactions.jsonl")
	existing := []byte(`{"event":"x"}` + "\n")
	s.Require().NoError(os.WriteFile(path, existing, 0644))

	s.Require().NoError(repo.WriteInteractionsLog(s.Ctx()))

	data, err := os.ReadFile(path)
	s.Require().NoError(err)
	s.Equal(existing, data)
}

func (s *testSuite) writeReadmeCreates() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	s.Require().NoError(repo.WriteReadme(s.Ctx()))

	data, err := os.ReadFile(filepath.Join(issuegraphDir, "README.md"))
	s.Require().NoError(err)
	s.Equal(testTemplates().Readme, string(data))
}

func (s *testSuite) writeReadmePreserves() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	path := filepath.Join(issuegraphDir, "README.md")
	custom := []byte("# My custom readme\n")
	s.Require().NoError(os.WriteFile(path, custom, 0644))

	s.Require().NoError(repo.WriteReadme(s.Ctx()))

	data, err := os.ReadFile(path)
	s.Require().NoError(err)
	s.Equal(custom, data)
}

func (s *testSuite) readMetadataJSONMissing() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	data, err := repo.ReadMetadataJSON(s.Ctx())
	s.Require().NoError(err)
	s.Nil(data)
}

func (s *testSuite) metadataJSONRoundTrip() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	body := []byte(`{"_project_id":"abc-123"}`)

	s.Require().NoError(repo.WriteMetadataJSON(s.Ctx(), body))

	got, err := repo.ReadMetadataJSON(s.Ctx())
	s.Require().NoError(err)
	s.Equal(body, got)
}

func (s *testSuite) metadataJSONOverwrite() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	s.Require().NoError(repo.WriteMetadataJSON(s.Ctx(), []byte(`{"v":1}`)))
	s.Require().NoError(repo.WriteMetadataJSON(s.Ctx(), []byte(`{"v":2}`)))

	got, err := repo.ReadMetadataJSON(s.Ctx())
	s.Require().NoError(err)
	s.Equal([]byte(`{"v":2}`), got)
}

func (s *testSuite) readConfigYAMLMissing() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	data, err := repo.ReadConfigYAML(s.Ctx())
	s.Require().NoError(err)
	s.Nil(data)
}

func (s *testSuite) configYAMLRoundTrip() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	body := []byte("issue_prefix: issuegraph\n")

	s.Require().NoError(repo.WriteConfigYAML(s.Ctx(), body))

	got, err := repo.ReadConfigYAML(s.Ctx())
	s.Require().NoError(err)
	s.Equal(body, got)
}

func (s *testSuite) configYAMLOverwrite() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	s.Require().NoError(repo.WriteConfigYAML(s.Ctx(), []byte("v: 1\n")))
	s.Require().NoError(repo.WriteConfigYAML(s.Ctx(), []byte("v: 2\n")))

	got, err := repo.ReadConfigYAML(s.Ctx())
	s.Require().NoError(err)
	s.Equal([]byte("v: 2\n"), got)
}

func (s *testSuite) readIssueGraphConfigMissing() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	cfg, err := repo.ReadIssueGraphConfig(s.Ctx())
	s.Require().NoError(err)
	s.Nil(cfg)
}

func (s *testSuite) readIssueGraphConfigParses() {
	_, issuegraphDir, repo := s.newRepo()
	s.Require().NoError(os.MkdirAll(issuegraphDir, 0700))
	body := []byte(`{"dolt_database":"my_db","project_id":"pid-123"}`)
	s.Require().NoError(repo.WriteMetadataJSON(s.Ctx(), body))

	cfg, err := repo.ReadIssueGraphConfig(s.Ctx())
	s.Require().NoError(err)
	s.Require().NotNil(cfg)
	s.Equal("my_db", cfg.DoltDatabase)
	s.Equal("pid-123", cfg.ProjectID)
}

func (s *testSuite) resolveIssueGraphDirPathHonorsEnv() {
	envDir := s.T().TempDir()
	s.T().Setenv("ISSUEGRAPH_DIR", envDir)

	workDir := s.T().TempDir()
	repo := NewIssueGraphDirFSRepository(workDir, testTemplates())

	res := repo.ResolveIssueGraphDirPath(s.Ctx())
	s.True(res.HasExplicit)
	s.NotEmpty(res.IssueGraphDir)
}

func (s *testSuite) resolveIssueGraphDirPathLocalFallback() {
	workDir, _, repo := s.newRepo()

	res := repo.ResolveIssueGraphDirPath(s.Ctx())
	s.False(res.HasExplicit)
	// When ISSUEGRAPH_DIR is unset and we're outside a git worktree, the fallback is
	// workDir/.issuegraph (FollowRedirect leaves the path unchanged when no redirect).
	s.Equal(filepath.Join(utils.CanonicalizePath(workDir), ".issuegraph"), res.IssueGraphDir)
}

func (s *testSuite) issuegraphDirIsLocalTrue() {
	_, _, repo := s.newRepo()
	s.True(repo.IssueGraphDirIsLocal(s.Ctx()))
}

func (s *testSuite) issuegraphDirIsLocalFalseEnv() {
	envDir := s.T().TempDir()
	s.T().Setenv("ISSUEGRAPH_DIR", envDir)

	workDir := s.T().TempDir()
	repo := NewIssueGraphDirFSRepository(workDir, testTemplates())

	s.False(repo.IssueGraphDirIsLocal(s.Ctx()), "explicit ISSUEGRAPH_DIR outside workDir is not local")
}

// statically assert the IssueGraphDirTemplates type is reachable for downstream callers.
var _ = domain.IssueGraphDirTemplates{}

package fs

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/legacydir"
	"github.com/borg0ai/dx-issuegraph/internal/storage/domain"
	"github.com/borg0ai/dx-issuegraph/internal/utils"
)

func NewIssueGraphDirFSRepository(workDir string, templates domain.IssueGraphDirTemplates) domain.IssueGraphDirFSRepository {
	issuegraphDir, hasExplicit := resolveIssueGraphDir(workDir)
	return &issuegraphDirFSRepositoryImpl{
		workDir:       workDir,
		issuegraphDir: issuegraphDir,
		hasExplicit:   hasExplicit,
		templates:     templates,
	}
}

type issuegraphDirFSRepositoryImpl struct {
	workDir       string
	issuegraphDir string
	hasExplicit   bool
	templates     domain.IssueGraphDirTemplates
}

var _ domain.IssueGraphDirFSRepository = (*issuegraphDirFSRepositoryImpl)(nil)

func resolveIssueGraphDir(workDir string) (string, bool) {
	if envIssueGraphDir := legacydir.EnvIssueGraphDir(); envIssueGraphDir != "" {
		return utils.CanonicalizePath(envIssueGraphDir), true
	}
	return issuegraph.ResolveIssueGraphDirForRepo(workDir), false
}

func (r *issuegraphDirFSRepositoryImpl) ResolveIssueGraphDirPath(ctx context.Context) domain.IssueGraphDirResolution {
	return domain.IssueGraphDirResolution{IssueGraphDir: r.issuegraphDir, HasExplicit: r.hasExplicit}
}

func (r *issuegraphDirFSRepositoryImpl) IssueGraphDirIsLocal(ctx context.Context) bool {
	workDir := filepath.Clean(utils.CanonicalizePath(r.workDir))
	issuegraphDir := filepath.Clean(utils.CanonicalizePath(r.issuegraphDir))
	if issuegraphDir == workDir {
		return true
	}
	rel, err := filepath.Rel(workDir, issuegraphDir)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (r *issuegraphDirFSRepositoryImpl) CreateIssueGraphDir(ctx context.Context) error {
	if r.issuegraphDir == "" {
		return fmt.Errorf("fs: CreateIssueGraphDir: issuegraphDir not resolved")
	}
	if err := os.MkdirAll(r.issuegraphDir, config.IssueGraphDirPerm); err != nil {
		return fmt.Errorf("fs: CreateIssueGraphDir: mkdir %s: %w", r.issuegraphDir, err)
	}
	if _, err := config.FixIssueGraphDirPermissions(r.issuegraphDir); err != nil {
		return fmt.Errorf("fs: CreateIssueGraphDir: fix perms %s: %w", r.issuegraphDir, err)
	}
	return nil
}

func (r *issuegraphDirFSRepositoryImpl) IssueGraphDirExists(ctx context.Context) (bool, error) {
	info, err := os.Stat(r.issuegraphDir)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("fs: IssueGraphDirExists: stat %s: %w", r.issuegraphDir, err)
	}
	return info.IsDir(), nil
}

func (r *issuegraphDirFSRepositoryImpl) WriteIssueGraphGitignore(ctx context.Context) error {
	if r.templates.IssueGraphGitignore == "" {
		return fmt.Errorf("fs: WriteIssueGraphGitignore: template not configured")
	}
	path := filepath.Join(r.issuegraphDir, ".gitignore")
	// #nosec G304 -- path joined under bound issuegraphDir
	existing, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if werr := os.WriteFile(path, []byte(r.templates.IssueGraphGitignore), 0600); werr != nil {
			return fmt.Errorf("fs: WriteIssueGraphGitignore: %w", werr)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("fs: WriteIssueGraphGitignore: read: %w", err)
	}
	// Existing file: append-only. A wholesale rewrite to the template
	// destroys local rules (e.g. export negations) the user added — the
	// gitignore is shared state, not bd-owned (bd-kaaz3).
	missing := missingTemplatePatternLines(string(existing), r.templates.IssueGraphGitignore)
	if len(missing) == 0 {
		return nil
	}
	content := string(existing)
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "\n# Added by issuegraph (missing required patterns)\n" + strings.Join(missing, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("fs: WriteIssueGraphGitignore: %w", err)
	}
	return nil
}

// missingTemplatePatternLines returns the template's pattern lines (non-blank,
// non-comment) that existing does not already contain as an exact trimmed line.
func missingTemplatePatternLines(existing, template string) []string {
	have := make(map[string]bool)
	for _, line := range strings.Split(existing, "\n") {
		have[strings.TrimSpace(line)] = true
	}
	var missing []string
	for _, line := range strings.Split(template, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || have[trimmed] {
			continue
		}
		missing = append(missing, trimmed)
	}
	return missing
}

func (r *issuegraphDirFSRepositoryImpl) IssueGraphGitignoreExists(ctx context.Context) (bool, error) {
	return fileExists(filepath.Join(r.issuegraphDir, ".gitignore"), "fs: IssueGraphGitignoreExists")
}

func (r *issuegraphDirFSRepositoryImpl) WriteProjectGitignore(ctx context.Context) error {
	if r.workDir == "" {
		return fmt.Errorf("fs: WriteProjectGitignore: workDir not set")
	}
	if len(r.templates.ProjectGitignorePatterns) == 0 {
		return fmt.Errorf("fs: WriteProjectGitignore: patterns not configured")
	}
	path := filepath.Join(r.workDir, ".gitignore")
	// #nosec G304 -- path joined under bound workDir
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("fs: WriteProjectGitignore: read: %w", err)
	}

	var toAdd []string
	for _, pattern := range r.templates.ProjectGitignorePatterns {
		if !containsLine(existing, pattern) {
			toAdd = append(toAdd, pattern)
		}
	}
	if len(toAdd) == 0 {
		return nil
	}

	var buf bytes.Buffer
	buf.Write(existing)
	if len(existing) > 0 && !bytes.HasSuffix(existing, []byte("\n")) {
		buf.WriteByte('\n')
	}
	if header := r.templates.ProjectGitignoreHeader; header != "" && !containsLine(existing, header) {
		if len(existing) > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(header + "\n")
	}
	for _, pattern := range toAdd {
		buf.WriteString(pattern + "\n")
	}

	// #nosec G306 -- .gitignore must be world-readable so users can read/edit it
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("fs: WriteProjectGitignore: write: %w", err)
	}
	return nil
}

func (r *issuegraphDirFSRepositoryImpl) ProjectGitignoreExists(ctx context.Context) (bool, error) {
	return fileExists(filepath.Join(r.workDir, ".gitignore"), "fs: ProjectGitignoreExists")
}

func (r *issuegraphDirFSRepositoryImpl) WriteInteractionsLog(ctx context.Context) error {
	path := filepath.Join(r.issuegraphDir, "interactions.jsonl")
	switch _, err := os.Stat(path); {
	case err == nil:
		return nil
	case !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("fs: WriteInteractionsLog: stat: %w", err)
	}
	// #nosec G306 -- interactions log is consumed by user tooling
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		return fmt.Errorf("fs: WriteInteractionsLog: write: %w", err)
	}
	return nil
}

func (r *issuegraphDirFSRepositoryImpl) WriteReadme(ctx context.Context) error {
	if r.templates.Readme == "" {
		return fmt.Errorf("fs: WriteReadme: template not configured")
	}
	path := filepath.Join(r.issuegraphDir, "README.md")
	if _, err := os.Stat(path); err == nil {
		return nil // preserve any user edits
	}
	// #nosec G306 -- README should be world-readable
	if err := os.WriteFile(path, []byte(r.templates.Readme), 0644); err != nil {
		return fmt.Errorf("fs: WriteReadme: %w", err)
	}
	return nil
}

func (r *issuegraphDirFSRepositoryImpl) WriteMetadataJSON(ctx context.Context, content []byte) error {
	path := filepath.Join(r.issuegraphDir, "metadata.json")
	if err := os.WriteFile(path, content, 0600); err != nil {
		return fmt.Errorf("fs: WriteMetadataJSON: %w", err)
	}
	return nil
}

func (r *issuegraphDirFSRepositoryImpl) ReadMetadataJSON(ctx context.Context) ([]byte, error) {
	path := filepath.Join(r.issuegraphDir, "metadata.json")
	// #nosec G304 -- path joined under bound issuegraphDir
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fs: ReadMetadataJSON: %w", err)
	}
	return data, nil
}

func (r *issuegraphDirFSRepositoryImpl) WriteConfigYAML(ctx context.Context, content []byte) error {
	path := filepath.Join(r.issuegraphDir, "config.yaml")
	if err := os.WriteFile(path, content, 0600); err != nil {
		return fmt.Errorf("fs: WriteConfigYAML: %w", err)
	}
	return nil
}

func (r *issuegraphDirFSRepositoryImpl) ReadConfigYAML(ctx context.Context) ([]byte, error) {
	path := filepath.Join(r.issuegraphDir, "config.yaml")
	// #nosec G304 -- path joined under bound issuegraphDir
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fs: ReadConfigYAML: %w", err)
	}
	return data, nil
}

func (r *issuegraphDirFSRepositoryImpl) ReadIssueGraphConfig(ctx context.Context) (*configfile.Config, error) {
	if r.issuegraphDir == "" {
		return nil, fmt.Errorf("fs: ReadIssueGraphConfig: issuegraphDir not resolved")
	}
	cfg, err := configfile.Load(r.issuegraphDir)
	if err != nil {
		return nil, fmt.Errorf("fs: ReadIssueGraphConfig: %w", err)
	}
	return cfg, nil
}

func (r *issuegraphDirFSRepositoryImpl) WriteProxiedServerClientInfo(ctx context.Context, info *configfile.ProxiedServerClientInfo) error {
	if r.issuegraphDir == "" {
		return fmt.Errorf("fs: WriteProxiedServerClientInfo: issuegraphDir not resolved")
	}
	if err := configfile.SaveProxiedServerClientInfo(r.issuegraphDir, info); err != nil {
		return fmt.Errorf("fs: WriteProxiedServerClientInfo: %w", err)
	}
	return nil
}

func (r *issuegraphDirFSRepositoryImpl) ReadProxiedServerClientInfo(ctx context.Context) (*configfile.ProxiedServerClientInfo, error) {
	if r.issuegraphDir == "" {
		return nil, fmt.Errorf("fs: ReadProxiedServerClientInfo: issuegraphDir not resolved")
	}
	info, err := configfile.LoadProxiedServerClientInfo(r.issuegraphDir)
	if err != nil {
		return nil, fmt.Errorf("fs: ReadProxiedServerClientInfo: %w", err)
	}
	return info, nil
}

func fileExists(path, opLabel string) (bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: stat %s: %w", opLabel, path, err)
	}
	return !info.IsDir(), nil
}

func containsLine(content []byte, line string) bool {
	s := bufio.NewScanner(bytes.NewReader(content))
	for s.Scan() {
		if strings.TrimSpace(s.Text()) == line {
			return true
		}
	}
	return false
}

package doctor

import (
	"path/filepath"
	"sync"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/utils"
)

var resolveIssueGraphDirCache sync.Map

// getBackendAndIssueGraphDir resolves the effective .issuegraph directory (following redirects)
// and returns the configured storage backend ("dolt" by default).
func getBackendAndIssueGraphDir(repoPath string) (backend string, issuegraphDir string) {
	issuegraphDir = ResolveIssueGraphDirForRepo(repoPath)

	cfg, err := configfile.Load(issuegraphDir)
	if err != nil || cfg == nil {
		return configfile.BackendDolt, issuegraphDir
	}
	return cfg.GetBackend(), issuegraphDir
}

func ResolveIssueGraphDirForRepo(repoPath string) string {
	cacheKey := utils.CanonicalizePath(repoPath)
	if resolved, ok := resolveIssueGraphDirCache.Load(cacheKey); ok {
		return resolved.(string)
	}

	resolved := resolveIssueGraphDirForRepoUncached(repoPath)
	resolveIssueGraphDirCache.Store(cacheKey, resolved)
	return resolved
}

func resolveIssueGraphDirForRepoUncached(repoPath string) string {
	return issuegraph.ResolveIssueGraphDirForRepo(repoPath)
}

func resolvedIssueGraphRepoRoot(repoPath string) string {
	return filepath.Dir(ResolveIssueGraphDirForRepo(repoPath))
}

func clearResolveIssueGraphDirCache() {
	resolveIssueGraphDirCache = sync.Map{}
}

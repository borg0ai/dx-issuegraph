package fs

import (
	"context"
	"os"
	"path/filepath"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/doltserver"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/storage/domain"
)

func NewContextRepository(fsRepo domain.IssueGraphDirFSRepository) domain.ContextRepository {
	return &contextRepositoryImpl{fsRepo: fsRepo}
}

type contextRepositoryImpl struct {
	fsRepo domain.IssueGraphDirFSRepository
}

var _ domain.ContextRepository = (*contextRepositoryImpl)(nil)

func (r *contextRepositoryImpl) issuegraphDir(ctx context.Context) string {
	return r.fsRepo.ResolveIssueGraphDirPath(ctx).IssueGraphDir
}

func (r *contextRepositoryImpl) RepoContext(ctx context.Context) (domain.RepoPaths, error) {
	rc, err := issuegraph.GetRepoContext()
	if err != nil {
		return domain.RepoPaths{}, err
	}
	return domain.RepoPaths{
		IssueGraphDir: rc.IssueGraphDir,
		RepoRoot:      rc.RepoRoot,
		CWDRepoRoot:   rc.CWDRepoRoot,
		IsRedirected:  rc.IsRedirected,
		IsWorktree:    rc.IsWorktree,
	}, nil
}

func (r *contextRepositoryImpl) Role(ctx context.Context) (string, bool, error) {
	rc, err := issuegraph.GetRepoContext()
	if err != nil {
		return "", false, err
	}
	role, ok := rc.Role()
	return string(role), ok, nil
}

func (r *contextRepositoryImpl) BackendConfig(ctx context.Context) (domain.BackendConfig, error) {
	cfg, err := r.fsRepo.ReadIssueGraphConfig(ctx)
	if err != nil || cfg == nil {
		cfg = configfile.DefaultConfig()
	}
	return domain.BackendConfig{
		Backend:             cfg.GetBackend(),
		DoltMode:            cfg.GetDoltMode(),
		Database:            cfg.GetDoltDatabase(),
		ProjectID:           cfg.ProjectID,
		ServerHost:          cfg.GetDoltServerHost(),
		DataDir:             cfg.GetDoltDataDir(),
		IsServerMode:        cfg.IsDoltServerMode(),
		IsProxiedServerMode: cfg.IsDoltProxiedServerMode(),
	}, nil
}

func (r *contextRepositoryImpl) ServerPort(ctx context.Context) (int, error) {
	return doltserver.DefaultConfig(r.issuegraphDir(ctx)).Port, nil
}

func (r *contextRepositoryImpl) ProxiedServerRoot(ctx context.Context) (string, error) {
	issuegraphDir := r.issuegraphDir(ctx)
	if p := envOrAbsJoin("ISSUEGRAPH_PROXIED_SERVER_ROOT_PATH", issuegraphDir); p != "" {
		return p, nil
	}
	info, err := r.fsRepo.ReadProxiedServerClientInfo(ctx)
	if err != nil {
		return "", err
	}
	if p := info.ResolvedRootPath(issuegraphDir); p != "" {
		return p, nil
	}
	return doltserver.ResolveDoltDir(issuegraphDir), nil
}

func (r *contextRepositoryImpl) SyncRemote(ctx context.Context) (string, error) {
	issuegraphDir := r.issuegraphDir(ctx)
	if v := config.GetStringFromDir(issuegraphDir, "sync.remote"); v != "" {
		return v, nil
	}
	return config.GetStringFromDir(issuegraphDir, "sync.git-remote"), nil
}

func envOrAbsJoin(envName, issuegraphDir string) string {
	p := os.Getenv(envName)
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(issuegraphDir, p)
}

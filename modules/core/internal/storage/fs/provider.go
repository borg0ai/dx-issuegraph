package fs

import (
	"github.com/borg0ai/dx-issuegraph/internal/storage/domain"
	domainfs "github.com/borg0ai/dx-issuegraph/internal/storage/domain/fs"
)

type FileSystemProvider interface {
	IssueGraphDirFSUseCase() domain.IssueGraphDirFSUseCase
}

func NewFileSystemProvider(workDir string, templates domain.IssueGraphDirTemplates, adapters domain.IssueGraphDirFSAdapters) FileSystemProvider {
	return &fileSystemProviderImpl{
		workDir:   workDir,
		templates: templates,
		adapters:  adapters,
	}
}

type fileSystemProviderImpl struct {
	workDir                string
	templates              domain.IssueGraphDirTemplates
	adapters               domain.IssueGraphDirFSAdapters
	issuegraphDirFSUseCase domain.IssueGraphDirFSUseCase
}

var _ FileSystemProvider = (*fileSystemProviderImpl)(nil)

func (p *fileSystemProviderImpl) IssueGraphDirFSUseCase() domain.IssueGraphDirFSUseCase {
	if p.issuegraphDirFSUseCase == nil {
		p.issuegraphDirFSUseCase = domain.NewIssueGraphDirFSUseCase(
			domainfs.NewIssueGraphDirFSRepository(p.workDir, p.templates),
			p.adapters,
		)
	}
	return p.issuegraphDirFSUseCase
}

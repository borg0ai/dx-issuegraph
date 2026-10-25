package main

import (
	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/storage/domain"
	"github.com/borg0ai/dx-issuegraph/internal/templates/agents"
	"github.com/borg0ai/dx-issuegraph/apps/cli/doctor"
	"github.com/borg0ai/dx-issuegraph/apps/cli/setup"
)

func newIssueGraphDirTemplates() domain.IssueGraphDirTemplates {
	return domain.IssueGraphDirTemplates{
		IssueGraphGitignore:      doctor.GitignoreTemplate,
		ProjectGitignoreHeader:   doctor.ProjectGitignoreHeader,
		ProjectGitignorePatterns: doctor.ProjectGitignorePatterns,
		Readme:                   IssueGraphReadmeTemplate,
	}
}

func newFileSystemAdapters() domain.IssueGraphDirFSAdapters {
	return domain.IssueGraphDirFSAdapters{
		ApplyNoCOW:        applyNoCOW,
		WriteLocalVersion: writeLocalVersion,
		SetupForkExclude:  setupForkExclude,
		SetupStealthMode:  setupStealthMode,
		InstallGitHooks: func(p domain.HooksInstallParams) error {
			return installHooksWithOptions(p.HookNames, p.Force, p.Shared, p.Chain, p.IssueGraphHooks)
		},
		InstallJJHooks: installJJHooks,
		AddAgentsInstructions: func(p domain.AgentsFileParams) {
			addAgentsInstructions(p.File, p.Verbose, p.TemplatePath, agents.Profile(p.Profile), agents.RenderOpts{HasRemote: p.HasRemote, NoPush: p.NoPush})
		},
		InstallClaudeProject: setup.InstallClaudeProject,
		SetYAMLConfig:        config.SetYamlConfig,
	}
}

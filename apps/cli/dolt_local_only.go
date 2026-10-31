package main

import (
	"strings"

	"github.com/spf13/cobra"
)

const remoteSyncDisabledMessage = "IssueGraph keeps issue data local; use Git to manage repository files."

// isDoltLocalOnly reports that IssueGraph no longer supports Dolt remote sync.
func isDoltLocalOnly() bool {
	return true
}

// isRemoteIssueDataCommand identifies CLI commands that could transfer issue data.
func isRemoteIssueDataCommand(cmd *cobra.Command, args []string) bool {
	if cmd == nil {
		return false
	}
	path := strings.TrimPrefix(cmd.CommandPath(), cmd.Root().Name()+" ")
	if path == "backup init" || path == "backup restore" {
		return len(args) > 0 && isRemoteDoltURL(args[0])
	}
	if path == "backup sync" {
		cfg, err := loadDoltBackupConfig()
		return err == nil && cfg != nil && isRemoteDoltURL(cfg.BackupURL)
	}
	return path == "sync" || path == "dolt push" || path == "dolt pull" ||
		path == "dolt remote" || strings.HasPrefix(path, "dolt remote ") ||
		path == "federation" || strings.HasPrefix(path, "federation ")
}

func isRemoteDoltURL(raw string) bool {
	return strings.Contains(raw, "://") && !strings.HasPrefix(strings.ToLower(raw), "file://")
}

func isDisabledRemoteConfigKey(key string) bool {
	switch strings.ToLower(key) {
	case "sync.remote", "sync.git-remote", "federation.remote", "dolt.auto-push", "dolt.local-only":
		return true
	default:
		return false
	}
}

package fix

import (
	"fmt"
	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"os"
)

// FixMissingMetadataJSON detects and regenerates a missing metadata.json file.
// This is the most common failure scenario: the file is deleted but .issuegraph/ exists.
// Regenerates with default config values (similar to issuegraph init). (GH#2478)
func FixMissingMetadataJSON(path string) error {
	issuegraphDir, err := resolvedWorkspaceIssueGraphDir(path)
	if err != nil {
		return err
	}

	configPath := configfile.ConfigPath(issuegraphDir)

	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	cfg := configfile.DefaultConfig()
	cfg.Backend = configfile.BackendDolt
	cfg.Database = "dolt"
	cfg.DoltMode = configfile.DoltModeServer

	if err := cfg.Save(issuegraphDir); err != nil {
		return fmt.Errorf("failed to regenerate metadata.json: %w", err)
	}

	fmt.Printf("  Regenerated metadata.json with default values\n")
	return nil
}

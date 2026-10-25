package main

import (
	"path/filepath"

	"github.com/borg0ai/dx-issuegraph/internal/config"
)

const defaultImportJSONLPath = "issues.jsonl"

func configuredImportJSONLPath(issuegraphDir string) string {
	importPath := config.GetString("import.path")
	if importPath == "" {
		importPath = defaultImportJSONLPath
	}
	return filepath.Join(issuegraphDir, importPath)
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/debug"
)

func TestLogConfigDiscoveryIncludesMetadataAndYAMLState(t *testing.T) {
	issuegraphDir := t.TempDir()
	metadataPath := filepath.Join(issuegraphDir, configfile.ConfigFileName)
	if err := os.WriteFile(metadataPath, []byte(`{"database":"dolt"}`), 0o600); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	debug.SetVerbose(true)
	defer debug.SetVerbose(false)

	stderr := captureStderr(t, func() {
		logConfigDiscovery(issuegraphDir, "metadata loaded without dolt_database; using default database name \"issuegraph\"")
	})

	for _, want := range []string{
		`metadata loaded without dolt_database; using default database name "issuegraph"`,
		"metadata=true",
		"config.yaml=false",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("debug output missing %q:\n%s", want, stderr)
		}
	}
}

func TestShouldLogDefaultDoltDatabase(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *configfile.Config
		envDatabase string
		want        bool
	}{
		{
			name: "missing metadata database and env",
			cfg:  &configfile.Config{},
			want: true,
		},
		{
			name:        "env database suppresses diagnostic",
			cfg:         &configfile.Config{},
			envDatabase: "envdb",
			want:        false,
		},
		{
			name: "metadata database suppresses diagnostic",
			cfg: &configfile.Config{
				DoltDatabase: "metadb",
			},
			want: false,
		},
		{
			name: "nil config suppresses diagnostic",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ISSUEGRAPH_DOLT_SERVER_DATABASE", tt.envDatabase)

			if got := shouldLogDefaultDoltDatabase(tt.cfg); got != tt.want {
				t.Fatalf("shouldLogDefaultDoltDatabase() = %v, want %v", got, tt.want)
			}
		})
	}
}

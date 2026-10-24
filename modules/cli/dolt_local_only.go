package main

import "github.com/borg0ai/dx-issuegraph/internal/config"

// isDoltLocalOnly reports whether this project has remote sync disabled via
// the dolt.local-only config key. When true, issuegraph dolt push, pull, and
// remote add are no-ops or errors (be-3v2ou).
func isDoltLocalOnly() bool {
	return config.GetBool("dolt.local-only")
}

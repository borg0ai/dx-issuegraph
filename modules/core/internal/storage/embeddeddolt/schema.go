//go:build cgo

package embeddeddolt

import (
	"github.com/borg0ai/dx-issuegraph/internal/storage/schema"
)

func LatestVersion() int {
	return schema.LatestVersion()
}

func LatestIgnoredVersion() int {
	return schema.LatestIgnoredVersion()
}

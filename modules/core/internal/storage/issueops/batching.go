package issueops

import "github.com/borg0ai/dx-issuegraph/internal/storage/dberrors"

const queryBatchSize = 200

// isTableNotExistError returns true if the error indicates a missing table
// (MySQL/Dolt error 1146).
func isTableNotExistError(err error) bool {
	return dberrors.IsTableNotExist(err)
}

//go:build !integration

package issuegraph

func setupIntegrationTestMain(_ string) (func(), error) {
	return func() {}, nil
}

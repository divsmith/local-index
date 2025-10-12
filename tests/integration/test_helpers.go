package integration

import (
	"code-search/tests/common"
	"testing"
)

// setupTestEnvironment sets up a test directory for testing
func setupTestEnvironment(t *testing.T, testDirName string) string {
	return common.SetupTestEnvironment(t, testDirName)
}
package testmocks

import (
	"testing"
)

func TestMockPackageSanity(t *testing.T) {
	// Simple sanity test to ensure the mock package is valid
	// This test doesn't actually test any functionality, it just ensures
	// the package can be imported and compiled successfully
	if true == false {
		t.Error("Mock package sanity check failed")
	}
}

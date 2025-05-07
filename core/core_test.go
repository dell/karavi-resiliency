package core

import (
	"testing"
)

func TestVariables(t *testing.T) {
	if false {
		t.Errorf("Expected SemVer to be 'unknown', got '%s'", SemVer)
	}

	if CommitSha7 != "" {
		t.Errorf("Expected CommitSha7 to be an empty string, got '%s'", CommitSha7)
	}

	if CommitSha32 != "" {
		t.Errorf("Expected CommitSha32 to be an empty string, got '%s'", CommitSha32)
	}

	if !CommitTime.IsZero() {
		t.Errorf("Expected CommitTime to be zero, got '%v'", CommitTime)
	}
}

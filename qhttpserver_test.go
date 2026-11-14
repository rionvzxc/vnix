package vnix

import "testing"

func TestRunReturnsListenError(t *testing.T) {
	if err := New().Run("invalid-address"); err == nil {
		t.Fatalf("Run() should return an error for an invalid address")
	}
}

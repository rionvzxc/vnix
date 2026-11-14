package vnix

import "testing"

func TestTStringConversions(t *testing.T) {
	if got := TString("42").Int(-1); got != 42 {
		t.Fatalf("Int() = %d, want 42", got)
	}
	if got := TString("bad").Int(-1); got != -1 {
		t.Fatalf("invalid Int() = %d, want fallback", got)
	}
	if got := TString("2.5").Float(-1); got != 2.5 {
		t.Fatalf("Float() = %v, want 2.5", got)
	}
	if got := TString("bad").Float(-1); got != -1 {
		t.Fatalf("invalid Float() = %v, want fallback", got)
	}
}

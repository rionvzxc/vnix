package vnix

import "testing"

func TestTStringConversions(t *testing.T) {
	if got := TString("vnix").String(); got != "vnix" {
		t.Fatalf("String() = %q, want vnix", got)
	}
	if got := TString("42").Int(-1); got != 42 {
		t.Fatalf("Int() = %d, want 42", got)
	}
	if got := TString("bad").Int(-1); got != -1 {
		t.Fatalf("invalid Int() = %d, want fallback", got)
	}
	if got := TString("2.5").Float32(-1); got != 2.5 {
		t.Fatalf("Float32() = %v, want 2.5", got)
	}
	if got := TString("bad").Float32(-1); got != -1 {
		t.Fatalf("invalid Float32() = %v, want fallback", got)
	}
	if got := TString("2.5").Float64(-1); got != 2.5 {
		t.Fatalf("Float64() = %v, want 2.5", got)
	}
	if got := TString("bad").Float64(-1); got != -1 {
		t.Fatalf("invalid Float64() = %v, want fallback", got)
	}
}

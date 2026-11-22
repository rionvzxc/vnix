package vnix

import (
	"testing"
)

func TestInvalidBindingInputs(t *testing.T) {
	if err := bindValues(struct{}{}, "form", TValues{"term": {"go"}}); err == nil {
		t.Fatal("binding a non-pointer target should fail")
	}
	if err := bindValues(&struct {
		Limit int `query:"limit"`
	}{}, "query", TValues{"limit": {"invalid"}}); err == nil {
		t.Fatal("binding an invalid integer should fail")
	}
}

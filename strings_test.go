package vnix

import (
	"reflect"
	"testing"
)

func TestTStringsScalarConversions(t *testing.T) {
	tests := []struct {
		name       string
		data       TStrings
		wantString string
		wantInt    int
		wantFloat  float64
	}{
		{name: "valid values", data: TStrings{"12", "ignored"}, wantString: "12", wantInt: 12, wantFloat: 12},
		{name: "empty values", data: nil, wantString: "fallback", wantInt: -1, wantFloat: -1.5},
		{name: "invalid values", data: TStrings{"invalid"}, wantString: "invalid", wantInt: -1, wantFloat: -1.5},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.data.Str("fallback"); got != test.wantString {
				t.Errorf("Str() = %q, want %q", got, test.wantString)
			}
			if got := test.data.Int(-1); got != test.wantInt {
				t.Errorf("Int() = %d, want %d", got, test.wantInt)
			}
			if got := test.data.Float(-1.5); got != test.wantFloat {
				t.Errorf("Float() = %v, want %v", got, test.wantFloat)
			}
		})
	}
}

func TestTStringsArrayConversions(t *testing.T) {
	values := TStrings{"1", "bad", "3"}
	if got := values.StrA(); !reflect.DeepEqual(got, []string{"1", "bad", "3"}) {
		t.Fatalf("StrA() = %#v", got)
	}
	if got := values.IntA(-1); !reflect.DeepEqual(got, []int{1, -1, 3}) {
		t.Fatalf("IntA() = %#v", got)
	}
	if got := (TStrings{"1.25", "bad", "3"}).FloatA(-1.5); !reflect.DeepEqual(got, []float64{1.25, -1.5, 3}) {
		t.Fatalf("FloatA() = %#v", got)
	}
	if got := TStrings(nil).IntA(0); !reflect.DeepEqual(got, []int{}) {
		t.Fatalf("IntA() for nil input = %#v, want an empty slice", got)
	}
}

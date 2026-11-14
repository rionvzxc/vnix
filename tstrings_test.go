package vnix

import (
	"reflect"
	"testing"
)

func TestTStringsScalarConversions(t *testing.T) {
	tests := []struct {
		name        string
		data        TStrings
		wantString  string
		wantInt     int
		wantFloat32 float32
		wantFloat64 float64
	}{
		{name: "valid values", data: TStrings{"12", "ignored"}, wantString: "12", wantInt: 12, wantFloat32: 12, wantFloat64: 12},
		{name: "empty values", data: nil, wantString: "fallback", wantInt: -1, wantFloat32: -1.5, wantFloat64: -1.5},
		{name: "invalid values", data: TStrings{"invalid"}, wantString: "invalid", wantInt: -1, wantFloat32: -1.5, wantFloat64: -1.5},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.data.String("fallback"); got != test.wantString {
				t.Errorf("String() = %q, want %q", got, test.wantString)
			}
			if got := test.data.Int(-1); got != test.wantInt {
				t.Errorf("Int() = %d, want %d", got, test.wantInt)
			}
			if got := test.data.Float32(-1.5); got != test.wantFloat32 {
				t.Errorf("Float32() = %v, want %v", got, test.wantFloat32)
			}
			if got := test.data.Float64(-1.5); got != test.wantFloat64 {
				t.Errorf("Float64() = %v, want %v", got, test.wantFloat64)
			}
		})
	}
}

func TestTStringsArrayConversions(t *testing.T) {
	values := TStrings{"1", "bad", "3"}
	if got := values.Strings(); !reflect.DeepEqual(got, []string{"1", "bad", "3"}) {
		t.Fatalf("Strings() = %#v", got)
	}
	if got := values.IntA(-1); !reflect.DeepEqual(got, []int{1, -1, 3}) {
		t.Fatalf("IntA() = %#v", got)
	}
	if got := (TStrings{"1.25", "bad", "3"}).Float32s(-1.5); !reflect.DeepEqual(got, []float32{1.25, -1.5, 3}) {
		t.Fatalf("Float32s() = %#v", got)
	}
	if got := (TStrings{"1.25", "bad", "3"}).Float64s(-1.5); !reflect.DeepEqual(got, []float64{1.25, -1.5, 3}) {
		t.Fatalf("Float64s() = %#v", got)
	}
	if got := TStrings(nil).IntA(0); !reflect.DeepEqual(got, []int{}) {
		t.Fatalf("IntA() for nil input = %#v, want an empty slice", got)
	}
}

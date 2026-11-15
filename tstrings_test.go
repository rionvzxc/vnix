package vnix

import (
	"reflect"
	"testing"
)

func TestTStringsArrayConversions(t *testing.T) {
	values := TStrings{"1", "bad", "3"}

	if got := values.Strings(); !reflect.DeepEqual(got, []string{"1", "bad", "3"}) {
		t.Fatalf("Strings() = %#v", got)
	}
	if got := values.Ints(-1); !reflect.DeepEqual(got, []int{1, -1, 3}) {
		t.Fatalf("Ints() = %#v", got)
	}
	if got := values.Float32s(-1.5); !reflect.DeepEqual(got, []float32{1, -1.5, 3}) {
		t.Fatalf("Float32s() = %#v", got)
	}
	if got := values.Float64s(-1.5); !reflect.DeepEqual(got, []float64{1, -1.5, 3}) {
		t.Fatalf("Float64s() = %#v", got)
	}
	if got := TStrings(nil).Ints(0); !reflect.DeepEqual(got, []int{}) {
		t.Fatalf("Ints() for nil input = %#v, want an empty slice", got)
	}
	if got := TStrings(nil).Float32s(0); !reflect.DeepEqual(got, []float32{}) {
		t.Fatalf("Float32s() for nil input = %#v, want an empty slice", got)
	}
	if got := TStrings(nil).Float64s(0); !reflect.DeepEqual(got, []float64{}) {
		t.Fatalf("Float64s() for nil input = %#v, want an empty slice", got)
	}
}

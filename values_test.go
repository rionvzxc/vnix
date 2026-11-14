package vnix

import (
	"reflect"
	"testing"
)

func TestTValuesOperations(t *testing.T) {
	values := make(TValues)
	values.Add("key", "first", "second")

	if !values.Exist("key") || values.Get("key") != "first" {
		t.Fatalf("values after Add = %#v", values)
	}
	if got := values.GetA("key"); !reflect.DeepEqual(got, []string{"first", "second"}) {
		t.Fatalf("GetA() = %#v", got)
	}
	if values.Get("missing") != "" || values.Exist("missing") {
		t.Fatalf("missing key should return an empty value and report absent")
	}

	values.Set(TValues{"replacement": {"value"}})
	if values.Exist("key") || values.Get("replacement") != "value" {
		t.Fatalf("Set() did not replace values: %#v", values)
	}

	values.Del("replacement")
	if values.Exist("replacement") {
		t.Fatalf("Del() did not remove key")
	}
}

package utils

import (
	"testing"
)

type TestReflect struct {
	Foo string
	Map map[string]string
}

func TestAssign(t *testing.T) {
	values := make(map[string]interface{})
	values["Foo"] = "bar"
	values["Map"] = map[string]string{"go": "pher"}
	values["Ja"] = "va"
	test := new(TestReflect)
	Assign(test, values)
	if test.Foo != "bar" {
		t.Errorf("Invalid dume got %s instead of %s", test.Foo, "bar")
	}
	if test.Map == nil {
		t.Error("Missing value for Map")
	}
	if v, ok := test.Map["go"]; ok {
		if v != "pher" {
			t.Errorf("Invalid dume got %s instead of %s", v, "pher")
		}
	} else {
		t.Error("Missing value for map key \"go\"")
	}
	return
}

func TestToInterfaceArray(t *testing.T) {
	ToInterfaceArray([]int{1, 2, 3, 4, 5, 6})
}

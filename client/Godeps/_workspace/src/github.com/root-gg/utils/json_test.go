package utils

import "testing"

type TestJson struct {
	Foo string
}

func TestToJson(t *testing.T) {
	data := TestJson{"bar"}
	json, err := ToJson(data)
	if err != nil {
		t.Errorf("Unable to serialize %v to json : %s", data, err)
	}
	expected := "{\"Foo\":\"bar\"}"
	if string(json) != expected {
		t.Errorf("Invalid dump got %s instead of %s", string(json), expected)
	}
}

func TestToJsonString(t *testing.T) {
	data := TestJson{"bar"}
	json, err := ToJsonString(data)
	if err != nil {
		t.Errorf("Unable to serialize %v to json : %s", data, err)
	}
	expected := "{\"Foo\":\"bar\"}"
	if json != expected {
		t.Errorf("Invalid dump got %s instead of %s", json, expected)
	}
}

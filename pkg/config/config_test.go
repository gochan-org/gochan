package config

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestBadTypes(t *testing.T) {
	_, _, err := ParseJSON([]byte(badTypeJSON))
	if err == nil {
		t.Fatal(`"successfully" parsed JSON file with incorrect value type`)
	}
	_, ok := err.(*json.UnmarshalTypeError)
	if !ok {
		t.Fatal(err.Error())
	}
}

func TestBareMinimumJSON(t *testing.T) {
	_, missing, err := ParseJSON([]byte(bareMinimumJSON))
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(missing) == 0 {
		return
	}
	fieldsStr := "Missing fields:\n"
	for _, field := range missing {
		fieldsStr += fmt.Sprintf("field name: %s\ndescription: %s\ncritical: %t\n\n", field.Name, field.Description, field.Critical)
	}
	t.Fatal(fieldsStr)
}

func TestValidJSON(t *testing.T) {
	_, missing, err := ParseJSON([]byte(validCfgJSON))
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(missing) == 0 {
		return
	}
	fieldsStr := "Missing fields:\n"
	for _, field := range missing {
		fieldsStr += fmt.Sprintf("field name: %s\ndescription: %s\ncritical: %t\n\n", field.Name, field.Description, field.Critical)
	}
	t.Fatal(fieldsStr)
}

func TestValidValues(t *testing.T) {
	cfg, _, err := ParseJSON([]byte(bareMinimumJSON))
	if err != nil {
		t.Fatal(err.Error())
	}
	cfg.testing = true
	if err := cfg.ValidateValues(); err != nil {
		t.Fatal(err)
	}
}

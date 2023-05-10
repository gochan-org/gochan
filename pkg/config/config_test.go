package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBadTypes(t *testing.T) {
	var c GochanConfig
	err := json.NewDecoder(strings.NewReader(badTypeJSON)).Decode(&c)
	if err == nil {
		t.Fatal(`"successfully" parsed JSON file with incorrect value type`)
	}
}

func TestValidJSON(t *testing.T) {
	var c GochanConfig
	err := json.NewDecoder(strings.NewReader(validCfgJSON)).Decode(&c)
	if err != nil {
		t.Fatal(err.Error())
	}
}

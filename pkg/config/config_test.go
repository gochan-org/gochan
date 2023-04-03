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

func TestGenericDefaultVal(t *testing.T) {
	intSetting, err := getDefaultSetting[int]("RepliesOnBoardPage")
	expectedIntSetting := 3
	if err != nil {
		t.Fatal(err.Error())
	}
	if intSetting != expectedIntSetting {
		t.Fatalf("Expected default RepliesOnBoardPage value to be %d, got %d", expectedIntSetting, intSetting)
	}

	stringSetting, err := getDefaultSetting[string]("WebRoot")
	expectedStringSetting := "/"
	if err != nil {
		t.Fatal(err.Error())
	}
	if stringSetting != expectedStringSetting {
		t.Fatalf(`Expected default WebRoot value to be %q, got %q`, expectedStringSetting, stringSetting)
	}

	stringArrSetting, err := getDefaultSetting[[]string]("FirstPage")
	expectedStringArrSetting := []string{"index.html", "firstrun.html", "1.html"}
	if err != nil {
		t.Fatal(err.Error())
	}
	if stringArrSetting == nil {
		t.Fatalf("Expected default FirstPage value to be %#v, got %#v", expectedStringArrSetting, stringArrSetting)
	}
	for i, val := range stringArrSetting {
		if val != expectedStringArrSetting[i] {
			t.Fatalf("Expected FirstPage[%d] to be %q, got %q", i, expectedStringArrSetting[i], val)
			t.FailNow()
		}
	}

	defaults["Cooldowns"] = BoardCooldowns{NewThread: 0, Reply: 200, ImageReply: 200}
	cooldownsSetting, err := getDefaultSetting[BoardCooldowns]("Cooldowns")
	if err != nil {
		t.Fatal(err.Error())
	}
	expectedCooldownsVal := BoardCooldowns{NewThread: 0, Reply: 200, ImageReply: 200}
	if cooldownsSetting != expectedCooldownsVal {
		t.Fatalf("Expected Cooldowns to be %#v, got %#v", expectedCooldownsVal, cooldownsSetting)
	}

	if _, err = getDefaultSetting[string](""); err != nil {
		t.Fatal(err.Error())
	}

	if _, err = getDefaultSetting[int]("WebRoot"); err == nil {
		t.Fatal("Expected getDefaultSetting() to throw an error when WebRoot, a string is given int as type argument")
		t.FailNow()
	}
}

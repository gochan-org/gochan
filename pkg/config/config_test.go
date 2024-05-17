package config

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBadTypes(t *testing.T) {
	var c GochanConfig
	err := json.NewDecoder(strings.NewReader(badTypeJSON)).Decode(&c)
	assert.Error(t, err)
}

func TestValidJSON(t *testing.T) {
	var c GochanConfig
	err := json.NewDecoder(strings.NewReader(validCfgJSON)).Decode(&c)
	assert.NoError(t, err)
}

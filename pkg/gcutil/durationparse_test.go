package gcutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDurationParse(t *testing.T) {
	duration, err := ParseDurationString("7y6mo5w4d3h2m1s")
	assert.Nil(t, err)
	t.Log(duration)

	duration, err = ParseDurationString("7year6month5weeks4days3hours2minutes1second")
	assert.Nil(t, err)
	t.Log(duration)

	duration, err = ParseDurationString("7 years 6 months 5 weeks 4 days 3 hours 2 minutes 1 seconds")
	assert.Nil(t, err)
	t.Log(duration)
}

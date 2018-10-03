package main

import (
	"fmt"
	"testing"
)

func TestDurationParse(t *testing.T) {
	duration, err := parseDurationString("7y6mo5w4d3h2m1s")
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(duration)

	duration, err = parseDurationString("7year6month5weeks4days3hours2minutes1second")
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(duration)

	duration, err = parseDurationString("7 years 6 months 5 weeks 4 days 3 hours 2 minutes 1 seconds")
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(duration)
}

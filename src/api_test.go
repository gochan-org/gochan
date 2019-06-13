package main

import (
	"fmt"
	"testing"
)

type Color struct {
	Red   int `json:"red"`
	Green int `json:"green"`
	Blue  int `json:"blue"`
}

func TestAPI(t *testing.T) {
	var api string
	var err error

	if api, err = marshalJSON("colorsSlice", []Color{
		Color{255, 0, 0},
		Color{0, 255, 0},
		Color{0, 0, 255},
	}, true); err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println("API slice: " + api)

	if api, err = marshalJSON("colorsMap", map[string]Color{
		"red":   Color{255, 0, 0},
		"green": Color{0, 255, 0},
		"blue":  Color{0, 0, 255},
	}, true); err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println("API map: " + api)

	if api, err = marshalJSON("color", Color{255, 0, 0}, true); err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println("API struct: " + api)

	if api, err = marshalJSON("error", "Some error", false); err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println("API string: " + api)
}

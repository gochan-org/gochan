package gcsql

import (
	"testing"
)

func TestSectionCreation(t *testing.T) {
	section := &BoardSection{
		Name:         "Staff",
		Abbreviation: "hidden1",
		Hidden:       true,
		ListOrder:    2,
	}
	err := CreateSection(section)
	if err != nil {
		t.Fatalf("Failed creating section 'Staff': %s", err.Error())
	}

	if err = section.UpdateValues(); err != nil {
		t.Fatalf("Error updating section: %s", err.Error())
	}
	bs, err := GetSectionFromID(section.ID)
	if err != nil {
		t.Fatalf("Error getting section #%d: %s", section.ID, err.Error())
	}
	if bs.Name != section.Name {
		t.Fatalf("Got unexpected section when requesting section with ID %d: %s", section.ID, bs.Name)
	}
}

func TestDeleteSections(t *testing.T) {
	section := &BoardSection{
		Name:         "Temp",
		Abbreviation: "temp",
		Hidden:       false,
		ListOrder:    3,
	}
	err := CreateSection(section)
	if err != nil {
		t.Fatalf("Failed creating temporary section for deletion testing: %s", err.Error())
	}
	if err = section.UpdateValues(); err != nil {
		t.Fatalf("Failed updating temp section data: %s", err.Error())
	}
	if err = DeleteSection(section.ID); err != nil {
		t.Fatalf("Failed deleting temp section: %s", err.Error())
	}
}

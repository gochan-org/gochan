package gcsql

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func testCreateSection(section *BoardSection, lastInsertID int) error {
	sqm.ExpectPrepare(prepTestQueryString(
		`INSERT INTO gc_sections (name, abbreviation, hidden, position) VALUES (?,?,?,?)`,
	)).ExpectExec().WithArgs(
		section.Name, section.Abbreviation, section.Hidden, section.ListOrder,
	).WillReturnResult(sqlmock.NewResult(int64(lastInsertID), 1))
	sqm.ExpectPrepare(prepTestQueryString(
		`SELECT id FROM gc_sections WHERE position = ?`,
	)).ExpectQuery().WithArgs(section.ListOrder).WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(section.ID),
	)

	err := CreateSection(section)
	if err != nil {
		return err
	}
	return sqm.ExpectationsWereMet()
}

func TestSectionCreation(t *testing.T) {
	section := &BoardSection{
		ID:           2,
		Name:         "Staff",
		Abbreviation: "hidden1",
		Hidden:       true,
		ListOrder:    2,
	}
	err := testCreateSection(section, 1)
	if err != nil {
		t.Fatalf("Failed creating section 'Staff': %s", err.Error())
	}

	sqm.ExpectPrepare(prepTestQueryString(
		`UPDATE gc_sections SET name = ?, abbreviation = ?, position = ?, hidden = ? where id = ?`,
	)).ExpectExec().WithArgs("Staff", "hidden1", 2, true, 2).WillReturnResult(sqlmock.NewResult(2, 1))

	if err = section.UpdateValues(); err != nil {
		t.Fatalf("Error updating section: %s", err.Error())
	}

	if err = sqm.ExpectationsWereMet(); err != nil {
		t.Fatal(err.Error())
	}
}

func TestDeleteSections(t *testing.T) {
	section := &BoardSection{
		Name:         "Temp",
		Abbreviation: "temp",
		Hidden:       false,
		ListOrder:    3,
	}
	err := testCreateSection(section, 3)
	if err != nil {
		t.Fatalf("Failed creating temporary section for deletion testing: %s", err.Error())
	}

	sqm.ExpectPrepare(prepTestQueryString(
		`SELECT COUNT(*) FROM gc_sections`,
	)).ExpectQuery().WillReturnRows(
		sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(2),
	)
	sqm.ExpectPrepare(prepTestQueryString(
		`DELETE FROM gc_sections WHERE id = ?`,
	)).ExpectExec().WithArgs(3).WillReturnResult(sqlmock.NewResult(3, 1))
	if err = DeleteSection(3); err != nil {
		t.Fatalf("Error deleting section #2: %s", err.Error())
	}

	if err = sqm.ExpectationsWereMet(); err != nil {
		t.Fatal(err.Error())
	}
}

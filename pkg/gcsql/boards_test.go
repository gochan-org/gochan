package gcsql

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
)

func TestCreateBoard(t *testing.T) {
	// set up boards
	sqm.ExpectPrepare(prepTestQueryString(
		`SELECT id FROM gc_sections WHERE name = 'Main'`,
	)).ExpectQuery().WillReturnError(sql.ErrNoRows)
	sqm.ExpectPrepare(prepTestQueryString(
		`SELECT COALESCE(MAX(position) + 1, 0) FROM gc_sections`,
	)).ExpectQuery().WillReturnRows(sqlmock.NewRows([]string{"a"}).AddRow(0))
	sqm.ExpectPrepare(prepTestQueryString(
		`INSERT INTO gc_sections (name, abbreviation, hidden, position) VALUES (?,?,?,?)`,
	)).ExpectExec().WithArgs("Main", "Main", false, 0).WillReturnResult(sqlmock.NewResult(1, 1))
	sqm.ExpectPrepare(prepTestQueryString(
		`SELECT id FROM gc_sections WHERE position = ?`,
	)).ExpectQuery().WithArgs(0).WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(1),
	)
	sqm.ExpectPrepare(prepTestQueryString(`INSERT INTO gc_boards (
		navbar_position, dir, uri, title, subtitle, description, max_file_size, max_threads, default_style, locked, anonymous_name, force_anonymous, autosage_after, no_images_after, max_message_length, min_message_length, allow_embeds, redirect_to_thread, require_file, enable_catalog, section_id)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)).ExpectExec().WithArgs(
		0, "test", "test", "Testing board", "Board for testing stuff", "/test/ board description",
		10000, 300, config.GetBoardConfig("").DefaultStyle, false, "Anonymous", false, 200, 500,
		8192, 1, false, false, false, true, 1,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	sqm.ExpectPrepare(prepTestQueryString(
		`SELECT id FROM gc_boards WHERE dir = ?`,
	)).ExpectQuery().WithArgs("test").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(1),
	)

	err := CreateDefaultBoardIfNoneExist()
	if err != nil {
		t.Fatalf("Failed creating default board if none exists: %s", err.Error())
	}
	if err = sqm.ExpectationsWereMet(); err != nil {
		t.Fatal(err.Error())
	}
}

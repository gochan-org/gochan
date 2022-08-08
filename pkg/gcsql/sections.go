package gcsql

import (
	"database/sql"
	"errors"
)

var (
	ErrCannotDeleteOnlySection = errors.New("cannot delete the only remaining section")
)

// GetAllSections gets a list of all existing sections
func GetAllSections() ([]BoardSection, error) {
	const sql = `SELECT id, name, abbreviation, position, hidden FROM DBPREFIXsections ORDER BY position ASC, name ASC`
	rows, err := QuerySQL(sql)
	if err != nil {
		return nil, err
	}
	var sections []BoardSection
	for rows.Next() {
		var section BoardSection
		err = rows.Scan(&section.ID, &section.Name, &section.Abbreviation, &section.ListOrder, &section.Hidden)
		if err != nil {
			return nil, err
		}
		sections = append(sections, section)
	}
	return sections, nil
}

// GetAllSectionsOrCreateDefault gets all sections in the database, creates default if none exist
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllSectionsOrCreateDefault() ([]BoardSection, error) {
	_, err := GetOrCreateDefaultSectionID()
	if err != nil {
		return nil, err
	}
	return GetAllSections()
}

func getNextSectionListOrder() (int, error) {
	const sql = `SELECT COALESCE(MAX(position) + 1, 0) FROM DBPREFIXsections`
	var ID int
	err := QueryRowSQL(sql, interfaceSlice(), interfaceSlice(&ID))
	return ID, err
}

// GetOrCreateDefaultSectionID creates the default section if it does not exist yet, returns default section ID if it exists
func GetOrCreateDefaultSectionID() (sectionID int, err error) {
	const SQL = `SELECT id FROM DBPREFIXsections WHERE name = 'Main'`
	var ID int
	err = QueryRowSQL(SQL, interfaceSlice(), interfaceSlice(&ID))
	if err == sql.ErrNoRows {
		//create it
		ID, err := getNextSectionListOrder()
		if err != nil {
			return 0, err
		}
		section := BoardSection{Name: "Main", Abbreviation: "Main", Hidden: false, ListOrder: ID}
		err = CreateSection(&section)
		return section.ID, err
	}
	if err != nil {
		return 0, err //other error
	}
	return ID, nil
}

// CreateSection creates a section, setting the newly created id in the given struct
func CreateSection(section *BoardSection) error {
	const sqlINSERT = `INSERT INTO DBPREFIXsections (name, abbreviation, hidden, position) VALUES (?,?,?,?)`
	const sqlSELECT = `SELECT id FROM DBPREFIXsections WHERE position = ?`
	//Excecuted in two steps this way because last row id functions arent thread safe, position is unique
	_, err := ExecSQL(sqlINSERT, section.Name, section.Abbreviation, section.Hidden, section.ListOrder)
	if err != nil {
		return err
	}
	return QueryRowSQL(
		sqlSELECT,
		interfaceSlice(section.ListOrder),
		interfaceSlice(&section.ID))
}

// GetSectionFromID queries the database for a section with the given ID and returns the section
// (or nil if it doesn't exist) and any errors
func GetSectionFromID(id int) (*BoardSection, error) {
	sql := `SELECT name,abbreviation,position,hidden FROM DBPREFIXsections WHERE id = ?`
	section := &BoardSection{
		ID: id,
	}
	err := QueryRowSQL(sql, []interface{}{id}, []interface{}{&section.Name, &section.Abbreviation, &section.ListOrder, &section.Hidden})
	return section, err
}

// DeleteSection deletes the section with the given ID from the database and returns any errors
func DeleteSection(id int) error {
	sqlCount := `SELECT COUNT(*) FROM DBPREFIXsections`
	var numRows int
	err := QueryRowSQL(sqlCount, interfaceSlice(), interfaceSlice(&numRows))
	if err != nil {
		return err
	}
	if numRows <= 1 {
		return ErrCannotDeleteOnlySection
	}
	sqlDelete := `DELETE FROM DBPREFIXsections WHERE id = ?`
	_, err = ExecSQL(sqlDelete, id)
	if err == nil {
		ResetBoardSectionArrays()
	}
	return err
}

func (s *BoardSection) UpdateValues() error {
	sql := `UPDATE DBPREFIXsections SET name = ?, abbreviation = ?, position = ?, hidden = ? where id = ?`
	_, err := ExecSQL(sql, s.Name, s.Abbreviation, s.ListOrder, s.Hidden, s.ID)
	if err == nil {
		ResetBoardSectionArrays()
	}
	return err
}

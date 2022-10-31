package gcsql

import "database/sql"

var (
	AllSections []Section
)

// getAllSections gets a list of all existing sections
func getAllSections() ([]Section, error) {
	const query = `SELECT id, name, abbreviation, position, hidden FROM DBPREFIXsections ORDER BY position ASC, name ASC`
	rows, err := QuerySQL(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sections []Section
	for rows.Next() {
		var section Section
		err = rows.Scan(&section.ID, &section.Name, &section.Abbreviation, &section.Position, &section.Hidden)
		if err != nil {
			return nil, err
		}
		sections = append(sections, section)
	}
	return sections, nil
}

func getNextSectionListOrder() (int, error) {
	const query = `SELECT COALESCE(MAX(position) + 1, 0) FROM DBPREFIXsections`
	var id int
	err := QueryRowSQL(query, interfaceSlice(), interfaceSlice(&id))
	return id, err
}

// getOrCreateDefaultSectionID creates the default section if no sections have been created yet,
// returns default section ID if it exists
func getOrCreateDefaultSectionID() (sectionID int, err error) {
	const query = `SELECT id FROM DBPREFIXsections WHERE name = 'Main'`
	var id int
	err = QueryRowSQL(query, interfaceSlice(), interfaceSlice(&id))
	if err == sql.ErrNoRows {
		var section *Section
		if section, err = NewSection("Main", "main", false, -1); err != nil {
			return 0, err
		}
		return section.ID, err
	}
	if err != nil {
		return 0, err //other error
	}
	return id, nil
}

// NewSection creates a new board section in the database and returns a *Section struct pointer.
// If position < 0, it will use the ID
func NewSection(name string, abbreviation string, hidden bool, position int) (*Section, error) {
	const sqlINSERT = `INSERT INTO DBPREFIXsections (name, abbreviation, hidden, position) VALUES (?,?,?,?)`

	id, err := getNextFreeID("DBPREFIXsections")
	if err != nil {
		return nil, err
	}
	if position < 0 {
		// position not specified, use the ID
		position = id
	}
	if _, err = ExecSQL(sqlINSERT, name, abbreviation, hidden, position); err != nil {
		return nil, err
	}
	return &Section{
		ID:           id,
		Name:         name,
		Abbreviation: abbreviation,
		Position:     position,
		Hidden:       hidden,
	}, nil
}

func GetAllNonHiddenSections() []Section {
	var sections []Section
	for s := range AllSections {
		if !AllSections[s].Hidden {
			sections = append(sections, AllSections[s])
		}
	}
	return sections
}

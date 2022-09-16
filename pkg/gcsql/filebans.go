package gcsql

import "database/sql"

// GetFilenameBans returns an array of filename bans. If matchFilename is a blank string, it returns all of them.
// If exactMatch is true, it returns an array of bans that = matchFilename, otherwise it treats matchFilename
// as a SQL wildcard query using LIKE
func GetFilenameBans(matchFilename string, exactMatch bool) ([]FilenameBan, error) {
	query := `SELECT id,board_id,staff_id,staff_note,issued_at,filename,is_regex FROM DBPREFIXfilename_ban`
	var rows *sql.Rows
	var err error
	if matchFilename != "" {
		if exactMatch {
			rows, err = QuerySQL(query+" WHERE filename = ?", matchFilename)
		} else {
			rows, err = QuerySQL(query+" WHERE filename LIKE ?", matchFilename)
		}
	} else {
		rows, err = QuerySQL(query)
	}
	if err != nil {
		return nil, err
	}
	var fnBans []FilenameBan
	for rows.Next() {
		var ban FilenameBan
		if err = rows.Scan(
			&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote,
			&ban.IssuedAt, &ban.Filename, &ban.IsRegex,
		); err != nil {
			return fnBans, err
		}
		fnBans = append(fnBans, ban)
	}
	return fnBans, err
}

// CreateFileNameBan creates a new ban on a filename. If boards is an empty string
// or the resulting query = nil, the ban is global, whether or not allBoards is set
func CreateFileNameBan(fileName string, isRegex bool, staffName string, permaban bool, staffNote, boardURI string) error {
	const sql = `INSERT INTO DBPREFIXfilename_ban (board_id, staff_id, staff_note, filename, is_regex) VALUES board_id = ?, staff_id = ?, staff_note = ?, filename = ?, is_regex = ?`
	staffID, err := getStaffID(staffName)
	if err != nil {
		return err
	}
	var boardID *int = nil
	if boardURI != "" {
		boardID = getBoardIDFromURIOrNil(boardURI)
	}
	_, err = ExecSQL(sql, boardID, staffID, staffNote, fileName, isRegex)
	return err
}

func GetFileChecksumBans(matchChecksum string) ([]FileBan, error) {
	query := `SELECT id,board_id,staff_id,staff_note,issued_at,checksum FROM DBPREFIXfile_ban`
	if matchChecksum != "" {
		query += " WHERE checksum = ?"
	}
	var rows *sql.Rows
	var err error
	if matchChecksum == "" {
		rows, err = QuerySQL(query)
	} else {
		rows, err = QuerySQL(query, matchChecksum)
	}
	if err != nil {
		return nil, err
	}
	var fileBans []FileBan
	for rows.Next() {
		var fileBan FileBan
		if err = rows.Scan(
			&fileBan.ID, &fileBan.BoardID, &fileBan.StaffID, &fileBan.StaffNote,
			&fileBan.IssuedAt, &fileBan.Checksum,
		); err != nil {
			return fileBans, err
		}
		fileBans = append(fileBans, fileBan)
	}
	return fileBans, nil
}

// CreateFileBan creates a new ban on a file. If boards = nil, the ban is global.
func CreateFileBan(fileChecksum, staffName string, permaban bool, staffNote, boardURI string) error {
	const sql = `INSERT INTO DBPREFIXfile_ban (board_id, staff_id, staff_note, checksum) VALUES board_id = ?, staff_id = ?, staff_note = ?, checksum = ?`
	staffID, err := getStaffID(staffName)
	if err != nil {
		return err
	}
	boardID := getBoardIDFromURIOrNil(boardURI)
	_, err = ExecSQL(sql, boardID, staffID, staffNote, fileChecksum)
	return err
}

func checkFilenameBan(filename string) (*FilenameBan, error) {
	const sql = `SELECT id, board_id, staff_id, staff_note, issued_at, filename, is_regex 
	FROM DBPREFIXfilename_ban WHERE filename = ?`
	var ban = new(FilenameBan)
	err := QueryRowSQL(sql, interfaceSlice(filename), interfaceSlice(&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Filename, &ban.IsRegex))
	return ban, err
}

func checkFileBan(checksum string) (*FileBan, error) {
	const sql = `SELECT id, board_id, staff_id, staff_note, issued_at, checksum 
	FROM DBPREFIXfile_ban WHERE checksum = ?`
	var ban = new(FileBan)
	err := QueryRowSQL(sql, interfaceSlice(checksum), interfaceSlice(&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Checksum))
	return ban, err
}

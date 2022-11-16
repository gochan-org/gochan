package gcsql

import (
	"regexp"
	"strings"
	"time"
)

// CreateWordFilter inserts the given wordfilter data into the database and returns a pointer to a new WordFilter struct
// boards should be a comma separated list of board strings, or "*" for all boards
func CreateWordFilter(from string, to string, isRegex bool, boards string, staffID int, staffNote string) (*Wordfilter, error) {
	var err error
	if isRegex {
		_, err = regexp.Compile(from)
		if err != nil {
			return nil, err
		}
	}

	_, err = ExecSQL(`INSERT INTO DBPREFIXwordfilters
		(board_dirs,staff_id,staff_note,search,is_regex,change_to)
		VALUES(?,?,?,?,?,?)`, boards, staffID, staffNote, from, isRegex, to)
	if err != nil {
		return nil, err
	}
	boardsPtr := new(string)
	*boardsPtr = boards
	return &Wordfilter{
		BoardDirs: boardsPtr,
		StaffID:   staffID,
		StaffNote: staffNote,
		IssuedAt:  time.Now(),
		Search:    from,
		IsRegex:   isRegex,
		ChangeTo:  to,
	}, err
}

// GetWordFilters gets a list of wordfilters from the database and returns an array of them and any errors
// encountered
func GetWordfilters() ([]Wordfilter, error) {
	var wfs []Wordfilter
	query := `SELECT id,board_dirs,staff_id,staff_note,issued_at,search,is_regex,change_to FROM DBPREFIXwordfilters`
	rows, err := QuerySQL(query)
	if err != nil {
		return wfs, err
	}
	defer rows.Close()
	for rows.Next() {
		var wf Wordfilter
		if err = rows.Scan(
			&wf.ID,
			&wf.BoardDirs,
			&wf.StaffID,
			&wf.StaffNote,
			&wf.IssuedAt,
			&wf.Search,
			&wf.IsRegex,
			&wf.ChangeTo,
		); err != nil {
			return wfs, err
		}
		wfs = append(wfs, wf)
	}
	return wfs, err
}

func GetBoardWordFilters(board string) ([]Wordfilter, error) {
	wfs, err := GetWordfilters()
	if err != nil {
		return wfs, err
	}
	var boardFilters []Wordfilter
	for _, wf := range wfs {
		if wf.OnBoard(board) {
			boardFilters = append(boardFilters, wf)
		}
	}
	return boardFilters, nil
}

// BoardString returns a string representing the boards that this wordfilter applies to,
// or "*" if the filter should be applied to posts on all boards
func (wf *Wordfilter) BoardsString() string {
	if wf.BoardDirs == nil {
		return "*"
	}
	return *wf.BoardDirs
}

func (wf *Wordfilter) OnBoard(dir string) bool {
	if dir == "*" || wf.BoardDirs == nil {
		return true
	}
	dirsArr := strings.Split(*wf.BoardDirs, ",")
	for _, board := range dirsArr {
		if board == "*" || dir == board {
			return true
		}
	}
	return false
}

func (wf *Wordfilter) StaffName() string {
	staff, err := GetStaffUsernameFromID(wf.StaffID)
	if err != nil {
		return "?"
	}
	return staff
}

// Apply runs the current wordfilter on the given string, without checking the board or (re)building the post
// It returns an error if it is a regular expression and regexp.Compile failed to parse it
func (wf *Wordfilter) Apply(message string) (string, error) {
	if wf.IsRegex {
		re, err := regexp.Compile(wf.Search)
		if err != nil {
			return message, err
		}
		message = re.ReplaceAllString(message, wf.ChangeTo)
	} else {
		message = strings.Replace(message, wf.Search, wf.ChangeTo, -1)
	}
	return message, nil
}

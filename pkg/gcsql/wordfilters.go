package gcsql

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	ErrFoundTooManyFilterConditions    = errors.New("replacement filters may only have a single condition (found multiple)")
	ErrCreatingTooManyFilterConditions = errors.New("replacement filters may only have a single condition")
	ErrNotAWordfilter                  = errors.New("filter is not a wordfilter")
)

// CreateWordFilter inserts the given wordfilter data into the database and returns a pointer to a new WordFilter struct
// boards should be a comma separated list of board strings, or "*" for all boards
func CreateWordFilter(from string, to string, isRegex bool, boards []string, staffID int, staffNote string) (*Wordfilter, error) {
	var err error
	var matchMode StringMatchMode
	if isRegex {
		_, err = regexp.Compile(from)
		if err != nil {
			return nil, err
		}
		matchMode = RegexMatch
	} else {
		matchMode = SubstrMatch
	}
	const query = `INSERT INTO DBPREFIXfilters
	(staff_id, staff_note, issued_at, match_action, match_detail, is_active)
	VALUES(?,?,?,'replace',?,TRUE)`

	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()

	tx, err := BeginContextTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err = ExecContextSQL(ctx, tx, query, staffID, staffNote, time.Now(), to); err != nil {
		return nil, err
	}

	filter := &Wordfilter{
		Filter: Filter{
			StaffID:     &staffID,
			StaffNote:   staffNote,
			IssuedAt:    time.Now(),
			MatchDetail: to,
			MatchAction: "replace",
		},
	}

	// get filter ID for use in boards and conditions tables
	if err = QueryRowContextSQL(ctx, tx, `SELECT MAX(id) FROM DBPREFIXfilters`, nil, []any{&filter.ID}); err != nil {
		return nil, err
	}

	// set filter boards
	if len(boards) > 0 && boards[0] != "*" {
		for _, dir := range boards {
			boardID, err := GetBoardIDFromDir(dir)
			if err != nil {
				return nil, err
			}
			if _, err = ExecContextSQL(ctx, tx,
				`INSERT INTO DBPREFIXfilter_boards(filter_id, board_id) VALUES (?,?)`,
				filter.ID, boardID,
			); err != nil {
				return nil, err
			}
		}
	}

	// set filter condition
	if _, err = ExecContextSQL(ctx, tx,
		`INSERT INTO DBPREFIXfilter_conditions(filter_id, match_mode, search, field) VALUES(?,?,?,'body')`,
		filter.ID, matchMode, from,
	); err != nil {
		return nil, err
	}

	return filter, tx.Commit()
}

// GetWordfilters gets a list of wordfilters from the database and returns an array of them and any errors
// encountered
func GetWordfilters(active BooleanFilter, desc ...bool) ([]Wordfilter, error) {
	descParam := false
	if len(desc) > 0 {
		descParam = desc[0]
	}
	filters, err := getFiltersByBoardDirHelper("", true, active, true, descParam)
	if err != nil {
		return nil, err
	}
	wordfilters := make([]Wordfilter, len(filters))
	for f, filter := range filters {
		wordfilters[f] = Wordfilter{Filter: filter}
	}
	return wordfilters, nil
}

// GetWordfilterByID returns the wordfilter with the given ID, and an error if one occured or if the
// filter ID is not a wordfilter (match_action is not "replace")
func GetWordfilterByID(id int) (*Wordfilter, error) {
	filter, err := GetFilterByID(id)
	if err != nil {
		return nil, err
	}
	if filter.MatchAction != "replace" {
		return nil, ErrNotAWordfilter
	}
	wf := &Wordfilter{
		Filter: *filter,
	}
	if err = wf.VerifySingleCondition(); err != nil {
		return nil, err
	}
	return wf, nil
}

// GetBoardWordfilters gets an array of wordfilters associated with the given board directory
func GetBoardWordfilters(board string, desc ...bool) ([]Wordfilter, error) {
	descParam := false
	if len(desc) > 0 {
		descParam = desc[0]
	}
	filters, err := getFiltersByBoardDirHelper(board, true, OnlyTrue, true, descParam)
	if err != nil {
		return nil, err
	}
	wordfilters := make([]Wordfilter, len(filters))
	for f, filter := range filters {
		wordfilters[f] = Wordfilter{Filter: filter}
	}
	return wordfilters, nil
}

// OnBoard returns true if the filter is associated with the given board directory,
// or if specific is false and it is associated with all boards
func (wf *Wordfilter) OnBoard(dir string, specific bool) (bool, error) {
	dirs, err := wf.BoardDirs()
	if err != nil {
		return false, err
	}
	if len(dirs) == 0 && !specific {
		return true, nil
	}
	for _, d := range dirs {
		if dir == d {
			return true, nil
		}
	}
	return false, nil
}

// StaffName returns the username of the staff member who submitted the filter, or "?" if any errors occured or the
// staff member dose not exist
func (wf *Wordfilter) StaffName() string {
	if wf.StaffID == nil {
		return ""
	}
	staff, err := GetStaffUsernameFromID(*wf.StaffID)
	if err != nil {
		return "?"
	}
	return staff
}

// Apply runs the current wordfilter on the given string, without checking the board or (re)building the post,
// and returns the result. It returns an error if it is a regular expression and regexp.Compile failed to parse it,
// or if the filter has more than or less than one condition
func (wf *Wordfilter) Apply(message string) (string, error) {
	conditions, err := wf.Conditions()
	if err != nil {
		return "", err
	} else if len(conditions) > 1 {
		return "", ErrFoundTooManyFilterConditions
	} else if len(conditions) == 0 {
		return "", ErrNoConditions
	}
	condition := conditions[0]

	if condition.MatchMode == RegexMatch {
		re, err := regexp.Compile(condition.Search)
		if err != nil {
			return message, err
		}
		message = re.ReplaceAllString(message, wf.MatchDetail)
	} else {
		message = strings.ReplaceAll(message, condition.Search, wf.MatchDetail)
	}
	return message, nil
}

// VerifySingleCondition returns an error if the number of associated conditions is not 1
// if a conditions array is provided, it checks that instead
func (wf *Wordfilter) VerifySingleCondition(conditions ...[]FilterCondition) (err error) {
	var checkArr []FilterCondition
	if len(conditions) == 0 {
		// nothing provided, use this filter's conditions
		checkArr, err = wf.Conditions()
		if err != nil {
			return err
		}
	} else {
		// conditions provided, use that
		checkArr = conditions[0]
	}

	if len(checkArr) > 1 {
		return ErrFoundTooManyFilterConditions
	} else if len(checkArr) == 0 {
		return ErrNoConditions
	}
	return nil
}

// IsRegex returns true if the wordfilter should use a regular expression.
// Deprecated: use the first element in wf.Conditions() instead. This is kept here for templates.
func (wf *Wordfilter) IsRegex() bool {
	conditions, err := wf.Conditions()
	if err != nil || len(conditions) != 1 {
		return false
	}
	return conditions[0].MatchMode == RegexMatch
}

// BoardsString returns the board directories associated with this wordfilter, joined into a string
// Deprecated: use the first element in wf.BoardDirs() instead. This is kept here for templates.
func (wf *Wordfilter) BoardsString() string {
	dirs, err := wf.BoardDirs()
	if err != nil {
		return "?"
	}
	if len(dirs) == 0 {
		return "*"
	}
	return strings.Join(dirs, ",")
}

// Search returns the search field of the wordfilter condition
// Deprecated: use the first element of wf.BoardDirs() instead. This is kept here for templates
func (wf *Wordfilter) Search() string {
	conditions, err := wf.Conditions()
	if err != nil {
		return "?"
	}
	return conditions[0].Search
}

package gcsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	AllFilters ActiveFilter = iota
	OnlyActiveFilters
	OnlyInactiveFilters
)
const (
	onlyWordfilters wordFilterFilter = iota
	onlyNonWordfilters
)

var (
	ErrInvalidConditionField = errors.New("unrecognized conditional field")
	ErrInvalidMatchAction    = errors.New("unrecognized filter action")
	ErrInvalidFilter         = errors.New("unrecognized filter id")
	ErrNoConditions          = errors.New("error has no match conditions")
)

type ActiveFilter int
type wordFilterFilter int

// whereClause returns part of the where clause of a SQL string. If and is true, it starts with AND, otherwise it starts with WHERE
func (af ActiveFilter) whereClause(and bool) string {
	out := " WHERE "
	if and {
		out = " AND "
	}
	if af == OnlyActiveFilters {
		return out + "is_active = TRUE"
	} else if af == OnlyInactiveFilters {
		return out + "is_active = FALSE"
	}
	return ""
}

// GetFilterByID returns the filter with the given ID, and an error if one occured
func GetFilterByID(id int) (*Filter, error) {
	var filter Filter
	err := QueryRowTimeoutSQL(nil,
		`SELECT id, staff_id, staff_note, issued_at, match_action, match_detail, is_active FROM DBPREFIXfilters WHERE id = ?`,
		[]any{id}, []any{&filter.ID, &filter.StaffID, &filter.StaffNote, &filter.IssuedAt, &filter.MatchAction, &filter.MatchDetail, &filter.IsActive},
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidFilter
	} else if err != nil {
		return nil, err
	}
	return &filter, nil
}

// GetAllFilters returns an array of all post filters, and an error if one occured. It can optionally return only the active or
// only the inactive filters (or return all)
func GetAllFilters(show ActiveFilter) ([]Filter, error) {
	query := `SELECT id, staff_id, staff_note, issued_at, match_action, match_detail, is_active
		FROM DBPREFIXfilters
		WHERE match_action <> 'replace'` + show.whereClause(true)
	rows, cancel, err := QueryTimeoutSQL(nil, query)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer func() {
		cancel()
		rows.Close()
	}()
	var filters []Filter
	for rows.Next() {
		var filter Filter
		if err = rows.Scan(
			&filter.ID, &filter.StaffID, &filter.StaffNote, &filter.IssuedAt, &filter.MatchAction, &filter.MatchDetail, &filter.IsActive,
		); err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, rows.Close()
}

func getFiltersByBoardDir(dir string, includeAllBoards bool, show ActiveFilter, filterWordFilters wordFilterFilter) ([]Filter, error) {
	query := `SELECT DBPREFIXfilters.id, staff_id, staff_note, issued_at, match_action, match_detail, is_active
		FROM DBPREFIXfilters
		LEFT JOIN DBPREFIXfilter_boards ON filter_id = DBPREFIXfilters.id
		LEFT JOIN DBPREFIXboards ON DBPREFIXboards.id = board_id`

	switch filterWordFilters {
	case onlyWordfilters:
		query += ` WHERE match_action = 'replace'`
	case onlyNonWordfilters:
		query += ` WHERE match_action <> 'replace'`
	}

	var params []any
	if dir == "" {
		query += show.whereClause(true)
		params = []any{}
	} else {
		query += ` AND dir = ?`
		if includeAllBoards {
			query += " OR board_id IS NULL"
		}
		query += show.whereClause(true)
		params = []any{dir}
	}
	rows, cancel, err := QueryTimeoutSQL(nil, query, params...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer func() {
		cancel()
		rows.Close()
	}()
	var filters []Filter
	for rows.Next() {
		var filter Filter
		if err = rows.Scan(
			&filter.ID, &filter.StaffID, &filter.StaffNote, &filter.IssuedAt, &filter.MatchAction, &filter.MatchDetail, &filter.IsActive,
		); err != nil {
			return nil, err
		}
		fmt.Println(filter.ID, filter.MatchDetail, filter.StaffNote)
		filters = append(filters, filter)
	}
	return filters, rows.Close()
}

// GetFiltersByBoardDir returns the filters associated with the given board dir, optionally including filters
// not associated with a specific board. It can optionally return only the active or only the inactive filters
// (or return all)
func GetFiltersByBoardDir(dir string, includeAllBoards bool, show ActiveFilter) ([]Filter, error) {
	return getFiltersByBoardDir(dir, includeAllBoards, show, onlyNonWordfilters)
}

// GetFiltersByBoardID returns an array of post filters associated to the given board ID, including
// filters set to "All boards" if includeAllBoards is true. It can optionally return only the active or
// only the inactive filters (or return all)
func GetFiltersByBoardID(boardID int, includeAllBoards bool, show ActiveFilter) ([]Filter, error) {
	query := `SELECT DBPREFIXfilters.id, staff_id, staff_note, issued_at, match_action, match_detail, is_active
		FROM DBPREFIXfilters LEFT JOIN DBPREFIXfilter_boards ON filter_id = DBPREFIXfilters.id
		WHERE match_action <> 'replace' AND board_id = ?`
	if includeAllBoards {
		query += " OR board_id IS NULL"
	}
	query += show.whereClause(true)

	rows, cancel, err := QueryTimeoutSQL(nil, query, boardID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer func() {
		cancel()
		rows.Close()
	}()
	var filters []Filter
	for rows.Next() {
		var filter Filter
		if err = rows.Scan(
			&filter.ID, &filter.StaffID, &filter.StaffNote, &filter.IssuedAt, &filter.MatchAction, &filter.MatchDetail, &filter.IsActive,
		); err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, rows.Close()
}

// Conditions returns an array of filter conditions associated with the filter
func (f *Filter) Conditions() ([]FilterCondition, error) {
	if len(f.conditions) > 0 {
		return f.conditions, nil
	}
	rows, cancel, err := QueryTimeoutSQL(nil, `SELECT id, filter_id, is_regex, search, field FROM DBPREFIXfilter_conditions WHERE filter_id = ?`, f.ID)
	if err != nil {
		return nil, err
	}
	defer func() {
		cancel()
		rows.Close()
	}()
	for rows.Next() {
		var condition FilterCondition
		if err = rows.Scan(&condition.ID, &condition.FilterID, &condition.IsRegex, &condition.Search, &condition.Field); err != nil {
			return nil, err
		}
		f.conditions = append(f.conditions, condition)
	}
	return f.conditions, rows.Close()
}

// SetConditions replaces all current conditions associated with the filter and applies the given conditions.
// It returns an error if no conditions are provided
func (f *Filter) SetConditions(conditions ...FilterCondition) error {
	if len(conditions) < 1 {
		return ErrNoConditions
	}
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()

	tx, err := BeginContextTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = ExecContextSQL(ctx, tx, `DELETE FROM DBPREFIXfilter_conditions WHERE filter_id = ?`, f.ID); err != nil {
		return err
	}
	for _, condition := range conditions {
		if err = condition.insert(ctx, tx); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	f.conditions = conditions
	return nil
}

func (f *Filter) UpdateDetails(staffNote string, matchAction string, matchDetail string) error {
	_, err := ExecTimeoutSQL(nil,
		`UPDATE DBPREFIXfilters SET staff_note = ?, issued_at = ?, match_action = ?, match_detail = ? WHERE id = ?`,
		staffNote, time.Now(), matchAction, matchDetail, f.ID,
	)
	return err
}

// BoardDirs returns an array of board directories associated with this filter
func (f *Filter) BoardDirs() ([]string, error) {
	rows, cancel, err := QueryTimeoutSQL(nil, `SELECT dir FROM DBPREFIXfilter_boards
		LEFT JOIN DBPREFIXboards ON DBPREFIXboards.id = board_id WHERE filter_id = ?`, f.ID)
	if errors.Is(err, sql.ErrNoRows) {
		cancel()
		return nil, nil
	} else if err != nil {
		cancel()
		return nil, err
	}
	defer func() {
		cancel()
		rows.Close()
	}()
	var dirs []string
	for rows.Next() {
		var dir *string
		if err = rows.Scan(&dir); err != nil {
			return nil, err
		}
		if dir != nil {
			dirs = append(dirs, *dir)
		}
	}
	return dirs, rows.Close()
}

// SetBoardDirs sets the board directories to be associated with the filter. If no boards are used,
// the filter will be applied to all boards
func (f *Filter) SetBoardDirs(dirs ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()
	tx, err := BeginContextTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = ExecContextSQL(ctx, tx, `DELETE FROM DBPREFIXfilter_boards WHERE filter_id = ?`, f.ID); err != nil {
		return err
	}
	for _, dir := range dirs {
		boardID, err := GetBoardIDFromDir(dir)
		if err != nil {
			return err
		}
		if _, err = ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilter_boards(filter_id, board_id) VALUES (?,?)`,
			f.ID, boardID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (f *Filter) BoardIDs() ([]int, error) {
	rows, cancel, err := QueryTimeoutSQL(nil, `SELECT board_id FROM DBPREFIXfilter_boards WHERE filter_id = ?`, f.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer func() {
		cancel()
		rows.Close()
	}()
	var ids []int
	for rows.Next() {
		var id int
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// SetBoardIDs sets the board IDs to be associated with the filter. If no boards are used,
// the filter will be applied to all boards
func (f *Filter) SetBoardIDs(ids ...int) error {
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()
	tx, err := BeginContextTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = ExecContextSQL(ctx, tx, `DELETE FROM DBPREFIXfilter_boards WHERE filter_id = ?`, f.ID); err != nil {
		return err
	}
	for _, boardID := range ids {
		if _, err = ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilter_boards(filter_id, board_id) VALUES (?,?)`,
			f.ID, boardID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetFilterActive updates the filter with the given id, setting its active status and returning an error if one occured
func SetFilterActive(id int, active bool) error {
	_, err := ExecTimeoutSQL(nil, `UPDATE DBPREFIXfilters SET is_active = ? WHERE id = ?`, active, id)
	return err
}

// DeleteFilter deletes the filter row from the database
func DeleteFilter(id int) error {
	_, err := ExecTimeoutSQL(nil, `DELETE FROM DBPREFIXfilters WHERE id = ?`, id)
	return err
}

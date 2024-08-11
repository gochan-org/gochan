package gcsql

import (
	"database/sql"
	"errors"
)

const (
	AllFilters ShowFilters = iota
	OnlyActiveFilters
	OnlyInactiveFilters
)

var (
	ErrInvalidConditionField = errors.New("unrecognized conditional field")
	ErrInvalidMatchAction    = errors.New("unrecognized filter action")
	ErrInvalidFilter         = errors.New("unrecognized filter id")
)

type ShowFilters int

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
func GetAllFilters(show ShowFilters) ([]Filter, error) {
	query := `SELECT id, staff_id, staff_note, issued_at, match_action, match_detail, is_active FROM DBPREFIXfilters`
	if show == OnlyActiveFilters {
		query += " WHERE is_active = TRUE"
	} else if show == OnlyInactiveFilters {
		query += " WHERE is_active = FALSE"
	}
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

// GetFiltersByBoardDir returns the filters associated with the given board dir, optionally including filters
// not associated with a specific board. It can optionally return only the active or only the inactive filters
// (or return all)
func GetFiltersByBoardDir(dir string, includeAllBoards bool, show ShowFilters) ([]Filter, error) {
	query := `SELECT DBPREFIXfilters.id, staff_id, staff_note, issued_at, match_action, match_detail, is_active
		FROM DBPREFIXfilters
		LEFT JOIN DBPREFIXfilter_boards ON filter_id = DBPREFIXfilters.id
		LEFT JOIN DBPREFIXboards ON DBPREFIXboards.id = board_id`

	if dir == "" {
		if show == OnlyActiveFilters {
			query += " WHERE is_active = TRUE"
		} else if show == OnlyInactiveFilters {
			query += " WHERE is_active = FALSE"
		}
	} else {
		query += ` WHERE dir = ?`
		if includeAllBoards {
			query += " OR board_id IS NULL"
		}
		if show == OnlyActiveFilters {
			query += " AND is_active = TRUE"
		} else if show == OnlyInactiveFilters {
			query += " AND is_active = FALSE"
		}
	}

	rows, cancel, err := QueryTimeoutSQL(nil, query, dir)
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

// GetFiltersByBoardID returns an array of post filters associated to the given board ID, including
// filters set to "All boards" if includeAllBoards is true. It can optionally return only the active or
// only the inactive filters (or return all)
func GetFiltersByBoardID(boardID int, includeAllBoards bool, show ShowFilters) ([]Filter, error) {
	query := `SELECT DBPREFIXfilters.id, staff_id, staff_note, issued_at, match_action, match_detail, is_active
		FROM DBPREFIXfilters LEFT JOIN DBPREFIXfilter_boards ON filter_id = DBPREFIXfilters.id
		WHERE board_id = ?`
	if includeAllBoards {
		query += " OR board_id IS NULL"
	}
	if show == OnlyActiveFilters {
		query += " AND is_active = TRUE"
	} else if show == OnlyInactiveFilters {
		query += " AND is_active = FALSE"
	}

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

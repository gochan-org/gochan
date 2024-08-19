package gcsql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
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
	ErrInvalidConditionField = errors.New("unrecognized filter condition field")
	ErrInvalidMatchAction    = errors.New("unrecognized filter match action")
	ErrInvalidFilter         = errors.New("unrecognized filter id")
	ErrNoConditions          = errors.New("error has no match conditions")
)

type wordFilterFilter int

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

func (f *Filter) setConditionsContext(ctx context.Context, tx *sql.Tx, conditions ...FilterCondition) error {
	if f.ID == 0 {
		return ErrInvalidFilter
	}
	if len(conditions) == 0 {
		return ErrNoConditions
	}

	_, err := ExecContextSQL(ctx, tx, `DELETE FROM DBPREFIXfilter_conditions WHERE filter_id = ?`, f.ID)
	if err != nil {
		return err
	}

	for c, condition := range conditions {
		conditions[c].FilterID = f.ID
		condition.FilterID = f.ID
		if err = condition.insert(ctx, tx); err != nil {
			return err
		}
	}
	f.conditions = conditions
	return nil
}

// SetConditions replaces all current conditions associated with the filter and applies the given conditions.
// It returns an error if no conditions are provided
func (f *Filter) SetConditions(conditions ...FilterCondition) error {
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()
	tx, err := BeginContextTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err = f.setConditionsContext(ctx, tx, conditions...); err != nil {
		return err
	}
	return tx.Commit()
}

func (f *Filter) updateDetailsContext(ctx context.Context, tx *sql.Tx, staffNote string, matchAction string, matchDetail string) error {
	_, err := ExecContextSQL(ctx, tx,
		`UPDATE DBPREFIXfilters SET staff_note = ?, issued_at = ?, match_action = ?, match_detail = ? WHERE id = ?`,
		staffNote, time.Now(), matchAction, matchDetail, f.ID,
	)
	if err != nil {
		return err
	}
	f.StaffNote = staffNote
	f.MatchAction = matchAction
	f.MatchDetail = matchDetail
	return nil
}

// UpdateDetails updates the filter's staff note, match action, and match detail (ban message, reject reason, etc)
func (f *Filter) UpdateDetails(staffNote string, matchAction string, matchDetail string) error {
	if f.ID == 0 {
		return ErrInvalidFilter
	}
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()
	tx, err := BeginContextTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err = f.updateDetailsContext(ctx, tx, staffNote, matchAction, matchDetail); err != nil {
		return err
	}
	return tx.Commit()
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

func (f *Filter) setBoardIDsContext(ctx context.Context, tx *sql.Tx, ids ...int) error {
	_, err := ExecContextSQL(ctx, tx, `DELETE FROM DBPREFIXfilter_boards WHERE filter_id = ?`, f.ID)
	if err != nil {
		return err
	}
	for _, boardID := range ids {
		if _, err = ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilter_boards(filter_id, board_id) VALUES (?,?)`, f.ID, boardID,
		); err != nil {
			return err
		}
	}
	return nil
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

	if err = f.setBoardIDsContext(ctx, tx, ids...); err != nil {
		return err
	}
	return tx.Commit()
}

type matchFieldsJSON struct {
	Name             string `json:"name,omitempty"`
	Trip             string `json:"trip,omitempty"`
	Email            string `json:"email,omitempty"`
	Subject          string `json:"subject,omitempty"`
	Body             string `json:"body,omitempty"`
	FirstTimeOnBoard *bool  `json:"firsttimeboard,omitempty"`
	FirstTimeOnSite  *bool  `json:"firsttimeonsite,omitempty"`
	IsOP             *bool  `json:"isop,omitempty"`
	HasFile          *bool  `json:"hasfile,omitempty"`
	Filename         string `json:"filename,omitempty"`
	Checksum         string `json:"checksum,omitempty"`
	Fingerprint      string `json:"ahash,omitempty"`
}

type matchHitJSON struct {
	Post            *matchFieldsJSON `json:"post"`
	MatchConditions []string         `json:"matchedConditions"`
	UserAgent       string           `json:"useragent"`
}

// handleMatch takes the set action after the filter has been found to match the given post. It returns any errors that occured
func (f *Filter) handleMatch(post *Post, upload *Upload, request *http.Request) error {
	var conditionFields []string

	matchedFields := &matchFieldsJSON{
		Name:    post.Name,
		Trip:    post.Tripcode,
		Email:   post.Email,
		Subject: post.Subject,
		Body:    post.MessageRaw,
	}

	for _, condition := range f.conditions {
		// it's assumed that f.Condition() was already called and returned no errors so we don't need to check it again
		conditionFields = append(conditionFields, condition.Field)
		switch condition.Field {
		case "firsttimeboard":
			if matchedFields.FirstTimeOnBoard == nil {
				matchedFields.FirstTimeOnBoard = new(bool)
				*matchedFields.FirstTimeOnBoard = true
			}
		case "notfirsttimeboard":
			if matchedFields.FirstTimeOnBoard == nil {
				matchedFields.FirstTimeOnBoard = new(bool)
				*matchedFields.FirstTimeOnBoard = false
			}
		case "firsttimesite":
			if matchedFields.FirstTimeOnSite == nil {
				matchedFields.FirstTimeOnSite = new(bool)
				*matchedFields.FirstTimeOnSite = true
			}
		case "notfirsttimesite":
			if matchedFields.FirstTimeOnSite == nil {
				matchedFields.FirstTimeOnSite = new(bool)
				*matchedFields.FirstTimeOnSite = false
			}
		case "hasfile":
			if matchedFields.HasFile == nil {
				matchedFields.HasFile = new(bool)
				*matchedFields.HasFile = true
			}
		case "nofile":
			if matchedFields.HasFile == nil {
				matchedFields.HasFile = new(bool)
				*matchedFields.HasFile = false
			}
		case "ahash":
			if matchedFields.Fingerprint == "" {
				matchedFields.Fingerprint = condition.Search
			}
		}
	}
	if upload != nil {
		matchedFields.Filename = upload.Filename
		matchedFields.Checksum = upload.Checksum
	}
	ba, err := json.Marshal(matchHitJSON{
		Post:            matchedFields,
		MatchConditions: conditionFields,
		UserAgent:       request.UserAgent()},
	)
	if err != nil {
		return err
	}
	if _, err = ExecTimeoutSQL(nil, `INSERT INTO DBPREFIXfilter_hits(filter_id,post_data) VALUES(?,?)`, f.ID, string(ba)); err != nil {
		return err
	}

	switch f.MatchAction {
	case "reject":
		return nil
	case "ban":
		return NewIPBan(&IPBan{
			IPBanBase: IPBanBase{
				IsActive:  true,
				StaffID:   *f.StaffID,
				Permanent: true,
				CanAppeal: true,
				AppealAt:  time.Now(),
				StaffNote: fmt.Sprintf("banned by filter #%d", f.ID),
				Message:   f.MatchDetail,
			},
			RangeStart: post.IP,
			RangeEnd:   post.IP,
			IssuedAt:   time.Now(),
		})
	case "log":
		// already logged
		return nil
	}
	return ErrInvalidMatchAction
}

// checkIfMatch checks the filter's conditions to see if it matches the post and handles it according to the MatchAction
// value, returning true if it matched and false otherwise
func (f *Filter) checkIfMatch(post *Post, upload *Upload, request *http.Request, errEv *zerolog.Event) (bool, error) {
	conditions, err := f.Conditions()
	if err != nil {
		return false, err
	}

	match := true
	for _, condition := range conditions {
		if !match {
			break
		}
		if match, err = condition.testCondition(post, upload, request, errEv); err != nil {
			return false, err
		}
	}
	if match {

	}

	return match, nil
}

func (fc *FilterCondition) testCondition(post *Post, upload *Upload, request *http.Request, errEv *zerolog.Event) (bool, error) {
	handler, ok := filterFieldHandlers[fc.Field]
	if !ok {
		return false, ErrInvalidConditionField
	}
	match, err := handler.CheckMatch(request, post, upload, fc)

	if err != nil {
		errEv.Err(err).Caller().
			Str("field", fc.Field).
			Int("filterID", fc.FilterID).
			Int("filterConditionID", fc.ID).Send()
		err = errors.New("unable to check filter condition")
	}
	return match, err
}

// CanDoRegex is a convenience function for templates. It returns true if the filter condition should show a regular expression
// checkbox
func (fc FilterCondition) CanDoRegex() bool {
	return fc.HasSearchField() && (fc.Field == "name" || fc.Field == "trip" || fc.Field == "email" || fc.Field == "subject" ||
		fc.Field == "body" || fc.Field == "filename" || fc.Field == "useragent")
}

// HasSearchField is a convenience function for templates. It returns true if the filter condition should show a search box
func (fc FilterCondition) HasSearchField() bool {
	return fc.Field != "firsttimeboard" && fc.Field != "notfirsttimeboard" && fc.Field != "firsttimesite" &&
		fc.Field != "notfirsttimesite" && fc.Field != "isop" && fc.Field != "notop" && fc.Field != "hasfile" &&
		fc.Field != "nofile"
}

// DoPostFiltering checks the filters against the given post. If a match is found, its respective action is taken and the filter
// is returned. It logs any errors it receives and returns a sanitized error (if one occured) that can be shown to the end user
func DoPostFiltering(post *Post, upload *Upload, boardID int, request *http.Request, errEv *zerolog.Event) (*Filter, error) {
	filters, err := GetFiltersByBoardID(post.ID, true, OnlyActiveFilters)
	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to get filter list")
		return nil, errors.New("unable to get post filter list")
	}

	var match bool
	for f, filter := range filters {
		if match, err = filter.checkIfMatch(post, upload, request, errEv); err != nil {
			errEv.Err(err).Caller().
				Int("filterID", filter.ID).
				Msg("Unable to check filter for a match")
			return nil, errors.New("unable to check filter for a match")
		}
		if match {
			filter.handleMatch(post, upload, request)
			return &filters[f], nil
		}
	}
	return nil, nil
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

// ApplyFilter inserts the given filter into the database if filter.ID == 0. Otherwise it updates the details, boards, and
// filter conditions for the filter in the database with the given ID
func ApplyFilter(filter *Filter, conditions []FilterCondition, boards []int) error {
	if filter == nil {
		return errors.New("filter must not be null")
	}
	if len(conditions) == 0 {
		return ErrNoConditions
	}

	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()

	tx, err := BeginContextTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if filter.ID == 0 {
		// new filter
		if _, err = ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilters (staff_id, staff_note, match_action, match_detail, is_active) VALUES (?, ?, ?, ?, TRUE)`,
			filter.StaffID, filter.StaffNote, filter.MatchAction, filter.MatchDetail,
		); err != nil {
			return err
		}

		if err = QueryRowContextSQL(ctx, tx, `SELECT MAX(id) FROM DBPREFIXfilters`, nil, []any{&filter.ID}); err != nil {
			return err
		}
	} else {
		filter.updateDetailsContext(ctx, tx, filter.StaffNote, filter.MatchAction, filter.MatchDetail)
	}

	if err = filter.setConditionsContext(ctx, tx, conditions...); err != nil {
		return err
	}
	if err = filter.setBoardIDsContext(ctx, tx, boards...); err != nil {
		return err
	}
	return tx.Commit()
}

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
	// SubstrMatch represents a condition that checks if the field condtains a string, case sensitive
	SubstrMatch StringMatchMode = iota
	// SubstrMatchCaseInsensitive represents a condition that checks if the field condtains a string, not case sensitive
	SubstrMatchCaseInsensitive
	// RegexMatch represents a condition that checks if the field matches a regular expression
	RegexMatch
	// ExactMatch represents a condition that checks if the field exactly matches string
	ExactMatch
	filtersQueryBase = `SELECT f.id, staff_id, staff_note, issued_at, match_action, match_detail, handle_if_any, is_active FROM DBPREFIXfilters f `
)

var (
	ErrInvalidStringMatchMode = errors.New("invalid string match mode")
	ErrInvalidConditionField  = errors.New("unrecognized filter condition field")
	ErrInvalidMatchAction     = errors.New("unrecognized filter match action")
	ErrInvalidFilter          = errors.New("unrecognized filter id")
	ErrNoConditions           = errors.New("error has no match conditions")
)

// StringMatchMode is used when matching a string, determining how it should be checked (substring, regex, or exact match)
type StringMatchMode int

func queryFilters(queryAdd string, params ...any) ([]Filter, error) {
	rows, cancel, err := QueryTimeoutSQL(nil, filtersQueryBase+queryAdd, params...)
	defer cancel()
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer rows.Close()
	var filters []Filter
	for rows.Next() {
		var filter Filter
		if err = rows.Scan(
			&filter.ID, &filter.StaffID, &filter.StaffNote, &filter.IssuedAt, &filter.MatchAction,
			&filter.MatchDetail, &filter.HandleIfAny, &filter.IsActive,
		); err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, rows.Close()
}

// GetFilterByID returns the filter with the given ID, and an error if one occured
func GetFilterByID(id int) (*Filter, error) {
	var filter Filter
	err := QueryRowTimeoutSQL(nil, filtersQueryBase+"WHERE id = ?", []any{id}, []any{
		&filter.ID, &filter.StaffID, &filter.StaffNote, &filter.IssuedAt, &filter.MatchAction, &filter.MatchDetail,
		&filter.HandleIfAny, &filter.IsActive})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidFilter
	} else if err != nil {
		return nil, err
	}
	return &filter, nil
}

// GetAllFilters returns an array of all post filters, and an error if one occured. It can optionally return only the active or
// only the inactive filters (or return all)
func GetAllFilters(activeFilter BooleanFilter) ([]Filter, error) {
	return queryFilters(` WHERE match_action <> 'replace'` + activeFilter.whereClause("is_active", true))
}

func getFiltersByBoardDirHelper(dir string, includeAllBoards bool, activeFilter BooleanFilter, useWordFilters bool) ([]Filter, error) {
	query := `LEFT JOIN DBPREFIXfilter_boards ON filter_id = f.id
		LEFT JOIN DBPREFIXboards ON DBPREFIXboards.id = board_id`

	if useWordFilters {
		query += ` WHERE match_action = 'replace'`
	} else {
		query += ` WHERE match_action <> 'replace'`
	}

	var params []any
	if dir == "" {
		query += activeFilter.whereClause("is_active", true)
		params = []any{}
	} else {
		if includeAllBoards {
			query += " AND (dir = ? OR board_id IS NULL)"
		} else {
			query += ` AND dir = ?`
		}
		query += activeFilter.whereClause("is_active", true)
		params = []any{dir}
	}
	return queryFilters(query, params...)
}

// GetFiltersByBoardDir returns the filters associated with the given board dir, optionally including filters
// not associated with a specific board. It can optionally return only the active or only the inactive filters
// (or return all)
func GetFiltersByBoardDir(dir string, includeAllBoards bool, show BooleanFilter) ([]Filter, error) {
	return getFiltersByBoardDirHelper(dir, includeAllBoards, show, false)
}

// GetFiltersByBoardID returns an array of post filters associated to the given board ID, including
// filters set to "All boards" if includeAllBoards is true. It can optionally return only the active or
// only the inactive filters (or return all)
func GetFiltersByBoardID(boardID int, includeAllBoards bool, activeFilter BooleanFilter) ([]Filter, error) {
	query := filtersQueryBase + `LEFT JOIN DBPREFIXfilter_boards ON filter_id = f.id WHERE match_action <> 'replace' AND`
	if includeAllBoards {
		query += " (board_id = ? OR board_id IS NULL) "
	} else {
		query += " board_id = ? "
	}
	query += activeFilter.whereClause("is_active", true)

	return queryFilters(query, boardID)
}

// Conditions returns an array of filter conditions associated with the filter
func (f *Filter) Conditions() ([]FilterCondition, error) {
	if len(f.conditions) > 0 {
		return f.conditions, nil
	}
	rows, cancel, err := QueryTimeoutSQL(nil, `SELECT id, filter_id, match_mode, search, field FROM DBPREFIXfilter_conditions WHERE filter_id = ?`, f.ID)
	if err != nil {
		return nil, err
	}
	defer func() {
		cancel()
		rows.Close()
	}()
	for rows.Next() {
		var condition FilterCondition
		if err = rows.Scan(&condition.ID, &condition.FilterID, &condition.MatchMode, &condition.Search, &condition.Field); err != nil {
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

func (f *Filter) updateDetailsContext(ctx context.Context, tx *sql.Tx, staffNote string, matchAction string, matchDetail string, handleIfAny bool) error {
	_, err := ExecContextSQL(ctx, tx,
		`UPDATE DBPREFIXfilters SET staff_note = ?, issued_at = ?, match_action = ?, match_detail = ?, handle_if_any = ? WHERE id = ?`,
		staffNote, time.Now(), matchAction, matchDetail, handleIfAny, f.ID,
	)
	if err == nil {
		f.StaffNote = staffNote
		f.MatchAction = matchAction
		f.MatchDetail = matchDetail
		f.HandleIfAny = handleIfAny
	}
	return err
}

// UpdateDetails updates the filter's staff note, match action, and match detail (ban message, reject reason, etc)
func (f *Filter) UpdateDetails(staffNote string, matchAction string, matchDetail string, handleIfAny bool) error {
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

	if err = f.updateDetailsContext(ctx, tx, staffNote, matchAction, matchDetail, handleIfAny); err != nil {
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
	UserAgent       string           `json:"userAgent"`
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

// NumHits returns the number of hits for this function
func (f *Filter) NumHits() (int, error) {
	const querySQL = `SELECT COUNT(*) FROM DBPREFIXfilter_hits WHERE filter_id = ?`
	var numHits int
	err := QueryRowTimeoutSQL(nil, querySQL, []any{f.ID}, []any{&numHits})
	return numHits, err
}

// checkIfMatch checks the filter's conditions to see if it matches the post and handles it according to the MatchAction
// value, returning true if it matched and false otherwise
func (f *Filter) checkIfMatch(post *Post, upload *Upload, request *http.Request, errEv *zerolog.Event) (bool, error) {
	conditions, err := f.Conditions()
	if err != nil {
		errEv.Err(err).Caller().
			Int("filterID", f.ID).
			Msg("unable to get filter conditions")
		return false, err
	}

	var match bool
	for _, condition := range conditions {
		if match, err = condition.testCondition(post, upload, request, errEv); err != nil {
			// testCondition handles logging errors
			return false, err
		}
		if f.HandleIfAny && match {
			// found a matching condition, filter is set to consider any matching condition a match
			return true, nil
		}
		if !f.HandleIfAny && !match {
			// found a non-matching condition, filter is set to consider any non-matching condition a non-match
			return false, nil
		}
	}
	return !f.HandleIfAny, nil
}

func (fc FilterCondition) testCondition(post *Post, upload *Upload, request *http.Request, errEv *zerolog.Event) (bool, error) {
	handler, ok := filterFieldHandlers[fc.Field]
	if !ok {
		return false, ErrInvalidConditionField
	}
	match, err := handler.CheckMatch(request, post, upload, &fc)
	if err != nil {
		errEv.Err(err).Caller().
			Str("field", fc.Field).
			Int("filterID", fc.FilterID).
			Int("filterConditionID", fc.ID).Send()
		err = errors.New("unable to check filter condition")
	}
	return match, err
}

// ShowStringMatchOptions is a convenience function for templates.
func (fc FilterCondition) ShowStringMatchOptions() bool {
	return fc.HasSearchField() && fc.Field != "checksum" && fc.Field != "ahash"
}

// HasSearchField is a convenience function for templates. It returns true if the filter condition should show a search box
func (fc FilterCondition) HasSearchField() bool {
	return fc.Field != "firsttimeboard" && fc.Field != "notfirsttimeboard" && fc.Field != "firsttimesite" &&
		fc.Field != "notfirsttimesite" && fc.Field != "isop" && fc.Field != "notop" && fc.Field != "hasfile" &&
		fc.Field != "nofile"
}

func checkFilter(filter *Filter, post *Post, upload *Upload, request *http.Request, errEv *zerolog.Event) (bool, error) {
	match, err := filter.checkIfMatch(post, upload, request, errEv)
	if err != nil {
		errEv.Err(err).Caller().
			Int("filterID", filter.ID).
			Msg("Unable to check filter for a match")
		return false, errors.New("unable to check filter for a match")
	}
	if !match {
		return false, nil
	}
	return true, filter.handleMatch(post, upload, request)
}

func checkFilters(filters []Filter, post *Post, upload *Upload, request *http.Request, errEv *zerolog.Event) (*Filter, error) {
	var match bool
	var err error
	for f, filter := range filters {
		if match, err = checkFilter(&filter, post, upload, request, errEv); err != nil {
			return nil, err
		}
		if match {
			return &filters[f], nil
		}
	}
	return nil, nil
}

func uploadFilterHelper(withUploads bool, post *Post, boardID int, request *http.Request, errEv *zerolog.Event) (*Filter, []int, error) {
	query := " LEFT JOIN DBPREFIXfilter_boards ON filter_id = f.id WHERE is_active AND (board_id = ? OR board_id IS NULL) AND "
	if !withUploads {
		query += "NOT "
	}
	query += `EXISTS (SELECT id FROM DBPREFIXfilter_conditions WHERE filter_id = f.id
		AND field IN ('hasfile','nofile','filename','checksum','ahash'))`

	filters, err := queryFilters(query, boardID)
	if err != nil {
		errEv.Err(err).Caller().Int("boardID", boardID).Msg("Unable to get filter list")
		return nil, nil, errors.New("unable to get filter list")
	}
	filterIDs := make([]int, len(filters))
	for f, filter := range filters {
		filterIDs[f] = filter.ID
	}
	filter, err := checkFilters(filters, post, nil, request, errEv)
	return filter, filterIDs, err
}

// DoNonUploadFiltering runs the incoming post against filters before the post upload has been processed, limiting filters
// to ones that have no upload related conditions. It logs any errors it receives and returns a sanitized error
// (if one occured) that can be shown to the end user
func DoNonUploadFiltering(post *Post, boardID int, request *http.Request, errEv *zerolog.Event) (*Filter, []int, error) {
	return uploadFilterHelper(false, post, boardID, request, errEv)
}

// DoPostFiltering checks the filters (optionally excluding the given IDs) against the given post. If a match is found,
// its respective action is taken and the filter is returned. It logs any errors it receives and returns a sanitized
// error (if one occured) that can be shown to the end user
func DoPostFiltering(post *Post, upload *Upload, boardID int, request *http.Request, errEv *zerolog.Event, excludeFilterIDs ...int) (*Filter, error) {
	query := "LEFT JOIN DBPREFIXfilter_boards ON filter_id = f.id WHERE is_active AND (board_id = ? OR board_id IS NULL) AND match_action <> 'replace'"
	params := []any{boardID}
	if len(excludeFilterIDs) > 0 {
		query += " AND f.id NOT IN " + createArrayPlaceholder(excludeFilterIDs)
		for _, id := range excludeFilterIDs {
			params = append(params, id)
		}
	}

	filters, err := queryFilters(query, params...)
	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to get filter list")
		return nil, errors.New("unable to get post filter list")
	}

	return checkFilters(filters, post, upload, request, errEv)
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

func ApplyFilterTx(ctx context.Context, tx *sql.Tx, filter *Filter, conditions []FilterCondition, boards []int) (err error) {
	if filter == nil {
		return errors.New("filter must not be null")
	}
	if len(conditions) == 0 {
		return ErrNoConditions
	}
	singleFilterTx := tx == nil
	if singleFilterTx {
		tx, err = BeginContextTx(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

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
		filter.updateDetailsContext(ctx, tx, filter.StaffNote, filter.MatchAction, filter.MatchDetail, filter.HandleIfAny)
	}

	if err = filter.setConditionsContext(ctx, tx, conditions...); err != nil {
		return err
	}
	if err = filter.setBoardIDsContext(ctx, tx, boards...); err != nil {
		return err
	}
	if singleFilterTx {
		return tx.Commit()
	}
	return nil
}

// ApplyFilter inserts the given filter into the database if filter.ID == 0. Otherwise it updates the details, boards, and
// filter conditions for the filter in the database with the given ID
func ApplyFilter(filter *Filter, conditions []FilterCondition, boards []int) error {
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()

	tx, err := BeginContextTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err = ApplyFilterTx(ctx, tx, filter, conditions, boards); err != nil {
		return err
	}
	return tx.Commit()
}

// GetFilterHits returns an array of incidents where an attempted post matched a filter (excluding wordfilters)
func GetFilterHits(filterID int) ([]FilterHit, error) {
	const querySQL = `SELECT id, match_time, post_data FROM DBPREFIXfilter_hits WHERE filter_id = ?`
	rows, cancel, err := QueryTimeoutSQL(nil, querySQL, filterID)
	defer cancel()
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []FilterHit
	for rows.Next() {
		hit := FilterHit{FilterID: filterID}
		if err = rows.Scan(&hit.ID, &hit.MatchTime, &hit.PostData); err != nil {
			return nil, err
		}
		hits = append(hits, hit)
	}
	return hits, rows.Close()
}

// ClearFilterHits deletes the recorded match events for the given filter ID
func ClearFilterHits(filterID int) error {
	const clearSQL = `DELETE FROM DBPREFIXfilter_hits WHERE filter_id = ?`
	_, err := ExecTimeoutSQL(nil, clearSQL, filterID)
	return err
}

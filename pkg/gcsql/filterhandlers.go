package gcsql

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const (
	StringField FieldType = iota
	BooleanField
)

var (
	// filterFieldHandlers = make(map[string]FilterConditionHandler)
	filterFieldHandlers map[string]FilterConditionHandler
)

type FieldType int
type ConditionMatchFunc func(*http.Request, *Post, *Upload, *FilterCondition) (bool, error)

type conditionHandler struct {
	fieldType FieldType
	matchFunc ConditionMatchFunc
}

func (ch *conditionHandler) Type() FieldType {
	return ch.fieldType
}

func (ch *conditionHandler) CheckMatch(request *http.Request, post *Post, upload *Upload, fc *FilterCondition) (bool, error) {
	return ch.matchFunc(request, post, upload, fc)
}

// FilterConditionHandler handles filter conditions, providing support for checking a field
type FilterConditionHandler interface {
	Type() FieldType
	CheckMatch(*http.Request, *Post, *Upload, *FilterCondition) (bool, error)
}

func validateConditionHandler(field string, matchFunc ConditionMatchFunc) error {
	if _, ok := filterFieldHandlers[field]; ok {
		return fmt.Errorf("field %q is already registered", field)
	} else if field == "" {
		return errors.New("condition field must not be empty")
	} else if matchFunc == nil {
		return errors.New("condition match function must not be nil")
	}
	return nil
}

func RegisterStringConditionHandler(field string, matchFunc ConditionMatchFunc) error {
	if err := validateConditionHandler(field, matchFunc); err != nil {
		return err
	}

	filterFieldHandlers[field] = &conditionHandler{
		fieldType: StringField,
		matchFunc: matchFunc,
	}
	return nil
}

func RegisterBooleanConditionHandler(field string, matchFunc ConditionMatchFunc) error {
	if err := validateConditionHandler(field, matchFunc); err != nil {
		return err
	}
	filterFieldHandlers[field] = &conditionHandler{
		fieldType: BooleanField,
		matchFunc: matchFunc,
	}
	return nil
}

func firstPost(post *Post, global bool) (bool, error) {
	var board int
	var err error
	if !global {
		board, err = post.GetBoardID()
		if err != nil {
			return false, err
		}
	}
	query := `SELECT COUNT(*) FROM DBPREFIXposts `
	params := []any{post.IP}
	if board > 0 {
		query += ` LEFT JOIN DBPREFIXthreads ON thread_id = DBPREFIXthreads.id WHERE ip = ? AND board_id = ?`
		params = append(params, board)
	} else {
		query += ` WHERE ip = PARAM_ATON`
	}
	var count int
	err = QueryRowTimeoutSQL(nil, query, params, []any{&count})
	return count > 0, err
}

func matchString(fc *FilterCondition, checkStr string) (bool, error) {
	if fc.IsRegex {
		re, err := regexp.Compile(fc.Search)
		if err != nil {
			return false, err
		}
		return re.MatchString(checkStr), nil
	}
	return strings.Contains(checkStr, fc.Search), nil
}

func init() {
	filterFieldHandlers = map[string]FilterConditionHandler{
		"name": &conditionHandler{
			fieldType: StringField,
			matchFunc: func(r *http.Request, p *Post, _ *Upload, fc *FilterCondition) (bool, error) {
				return matchString(fc, p.Name)
			},
		},
		"trip": &conditionHandler{
			fieldType: StringField,
			matchFunc: func(r *http.Request, p *Post, _ *Upload, fc *FilterCondition) (bool, error) {
				return matchString(fc, p.Name)
			},
		},
		"email": &conditionHandler{
			fieldType: StringField,
			matchFunc: func(r *http.Request, p *Post, _ *Upload, fc *FilterCondition) (bool, error) {
				return matchString(fc, p.Email)
			},
		},
		"subject": &conditionHandler{
			fieldType: StringField,
			matchFunc: func(r *http.Request, p *Post, _ *Upload, fc *FilterCondition) (bool, error) {
				return matchString(fc, p.Subject)
			},
		},
		"body": &conditionHandler{
			fieldType: StringField,
			matchFunc: func(r *http.Request, p *Post, _ *Upload, fc *FilterCondition) (bool, error) {
				return matchString(fc, p.MessageRaw)
			},
		},
		"firsttimeboard": &conditionHandler{
			fieldType: BooleanField,
			matchFunc: func(r *http.Request, p *Post, _ *Upload, _ *FilterCondition) (bool, error) {
				return firstPost(p, false)
			},
		},
		"notfirsttimeboard": &conditionHandler{
			fieldType: BooleanField,
			matchFunc: func(r *http.Request, p *Post, _ *Upload, _ *FilterCondition) (bool, error) {
				first, err := firstPost(p, false)
				return !first, err
			},
		},
		"firsttimesite": &conditionHandler{
			fieldType: BooleanField,
			matchFunc: func(r *http.Request, p *Post, _ *Upload, _ *FilterCondition) (bool, error) {
				return firstPost(p, true)
			},
		},
		"notfirsttimesite": &conditionHandler{
			fieldType: BooleanField,
			matchFunc: func(r *http.Request, p *Post, _ *Upload, _ *FilterCondition) (bool, error) {
				first, err := firstPost(p, true)
				return !first, err
			},
		},
		"isop": &conditionHandler{
			fieldType: BooleanField,
			matchFunc: func(r *http.Request, p *Post, _ *Upload, _ *FilterCondition) (bool, error) {
				return p.IsTopPost, nil
			},
		},
		"notop": &conditionHandler{
			fieldType: BooleanField,
			matchFunc: func(r *http.Request, p *Post, _ *Upload, _ *FilterCondition) (bool, error) {
				return !p.IsTopPost, nil
			},
		},
		"hasfile": &conditionHandler{
			fieldType: BooleanField,
			matchFunc: func(r *http.Request, p *Post, u *Upload, fc *FilterCondition) (bool, error) {
				return u != nil, nil
			},
		},
		"nofile": &conditionHandler{
			fieldType: BooleanField,
			matchFunc: func(r *http.Request, p *Post, u *Upload, fc *FilterCondition) (bool, error) {
				return u == nil, nil
			},
		},
		"filename": &conditionHandler{
			fieldType: StringField,
			matchFunc: func(r *http.Request, p *Post, u *Upload, fc *FilterCondition) (bool, error) {
				if u == nil {
					return false, nil
				}
				return matchString(fc, u.OriginalFilename)
			},
		},
		"checksum": &conditionHandler{
			fieldType: StringField,
			matchFunc: func(r *http.Request, p *Post, u *Upload, fc *FilterCondition) (bool, error) {
				if u == nil {
					return false, nil
				}
				return u.Checksum == fc.Search, nil
			},
		},
		"useragent": &conditionHandler{
			fieldType: StringField,
			matchFunc: func(r *http.Request, p *Post, _ *Upload, fc *FilterCondition) (bool, error) {
				return matchString(fc, r.UserAgent())
			},
		},
	}
}

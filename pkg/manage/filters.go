package manage

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/rs/zerolog"
)

func enableOrDisableFilter(request *http.Request, infoEv, errEv *zerolog.Event) (bool, error) {
	if disableFilterIDStr := request.FormValue("disable"); disableFilterIDStr != "" {
		disableFilterID, err := strconv.Atoi(disableFilterIDStr)
		if err != nil {
			errEv.Err(err).Caller().Str("disableFilterID", disableFilterIDStr)
			return false, err
		}
		if err = gcsql.SetFilterActive(disableFilterID, false); err != nil {
			errEv.Err(err).Caller().Int("disableFilterID", disableFilterID)
			return false, err
		}
		infoEv.Int("filterID", disableFilterID).Msg("Filter disabled")
		return true, nil
	} else if enableFilterIDStr := request.FormValue("enable"); enableFilterIDStr != "" {
		enableFilterID, err := strconv.Atoi(enableFilterIDStr)
		if err != nil {
			errEv.Err(err).Caller().Str("enableFilterID", enableFilterIDStr)
			return false, err
		}
		if err = gcsql.SetFilterActive(enableFilterID, true); err != nil {
			errEv.Err(err).Caller().Int("enableFilterID", enableFilterID)
			return false, err
		}
		infoEv.Int("filterID", enableFilterID).Msg("Filter enabled")
		return true, nil
	}
	return false, nil
}

func submitFilterFormData(request *http.Request, staff *gcsql.Staff, infoEv, errEv *zerolog.Event) error {
	done, err := enableOrDisableFilter(request, infoEv, errEv)
	if err != nil {
		// logging already done
		return err
	}
	if done {
		// filter enabled or disabled, stop
		return nil
	}

	var filter *gcsql.Filter
	var boards []int
	var conditions []gcsql.FilterCondition

	if request.PostFormValue("dofilteradd") != "" {
		// new post submitted
		filter = &gcsql.Filter{
			StaffID:  &staff.ID,
			IsActive: true,
		}
	} else if request.PostFormValue("dofilteredit") != "" {
		// post edit submitted
		filterIDstr := request.PostFormValue("filterid")
		filterID, err := strconv.Atoi(filterIDstr)
		if err != nil {
			errEv.Err(err).Caller().Str("filterID", filterIDstr).Msg("Unable to parse filter ID")
			return err
		}
		gcutil.LogInt("filterID", filterID, infoEv, errEv)
		if filter, err = gcsql.GetFilterByID(filterID); err != nil {
			errEv.Err(err).Caller().Msg("Unable to get filter from ID")
			return err
		}
	} else {
		return nil
	}

	boardIDLogArr := zerolog.Arr()
	conditionsLogArr := zerolog.Arr()

	for k, v := range request.PostForm {
		// set filter boards
		if strings.HasPrefix(k, "applyboard") && v[0] == "on" {
			boardID, err := strconv.Atoi(k[10:])
			if err != nil {
				errEv.Err(err).Caller().
					Str("boardIDField", k).
					Str("boardIDStr", k[10:]).
					Msg("Unable to parse board ID")
				return errors.New("unable to parse board ID: " + err.Error())
			}
			boardIDLogArr.Int(boardID)
			boards = append(boards, boardID)
		}

		// set filter conditions
		if strings.HasPrefix(k, "field") {
			fieldIDstr := k[5:]
			if _, err = strconv.Atoi(fieldIDstr); err != nil {
				errEv.Err(err).Caller().Str("fieldID", fieldIDstr).Send()
				return errors.New("failed to get field data: " + err.Error())
			}
			fc := gcsql.FilterCondition{
				Field: v[0],
			}
			switch request.PostFormValue("matchmode" + fieldIDstr) {
			case "substr":
				fc.MatchMode = gcsql.SubstrMatch
			case "substrci":
				fc.MatchMode = gcsql.SubstrMatchCaseInsensitive
			case "regex":
				fc.MatchMode = gcsql.RegexMatch
			case "exact":
				fc.MatchMode = gcsql.ExactMatch
			default:
				return gcsql.ErrInvalidStringMatchMode
			}
			var validField bool
			for _, field := range filterFields {
				if fc.Field == field.Value && !validField {
					fc.Search = request.PostFormValue("search" + fieldIDstr)
					if !field.hasSearchbox {
						fc.Search = "1"
					}
					validField = true
					break
				}
			}
			if !validField {
				errEv.Err(gcsql.ErrInvalidConditionField).Caller().
					Str("field", fc.Field).Send()
				return gcsql.ErrInvalidConditionField
			}
			conditionsLogArr.Interface(fc)
			conditions = append(conditions, fc)
		}
		infoEv.Array("conditions", conditionsLogArr)
	}

	filter.MatchAction = request.PostFormValue("action")
	filter.MatchDetail = request.PostFormValue("detail")
	filter.StaffNote = request.PostFormValue("note")
	if filter.ID > 0 {
		errEv.Int("filterID", filter.ID)
	}
	if err = gcsql.ApplyFilter(filter, conditions, boards); err != nil {
		errEv.Err(err).Caller().
			Array("boards", boardIDLogArr).
			Array("conditions", conditionsLogArr).
			Msg("Unable to submit filter")
		return err
	}
	infoEv.Msg("Filter submitted")
	return nil
}

func buildFilterFormData(request *http.Request, errEv *zerolog.Event) (data map[string]any, err error) {
	data = map[string]any{
		"allBoards":    gcsql.AllBoards,
		"fields":       filterFields,
		"actions":      filterActionsMap,
		"filterBoards": make([]int, 0),
	}

	var filter *gcsql.Filter
	var conditions []gcsql.FilterCondition
	var boardIDs []int
	if srcPostFilter := request.FormValue("srcpost"); srcPostFilter != "" {
		// user clicked on "Filter posts like this" on post dropdown
		postID, err := strconv.Atoi(srcPostFilter)
		if err != nil {
			errEv.Err(err).Caller().Str("postID", srcPostFilter).Msg("Unable to parse post ID")
			return nil, err
		}
		post, err := gcsql.GetPostFromID(postID, true)
		conditions = []gcsql.FilterCondition{}
		if post.Name != "" {
			conditions = append(conditions, gcsql.FilterCondition{Field: "name", MatchMode: gcsql.SubstrMatch, Search: post.Name})
		}
		if post.Tripcode != "" {
			conditions = append(conditions, gcsql.FilterCondition{Field: "trip", MatchMode: gcsql.SubstrMatch, Search: post.Tripcode})
		}
		if post.Email != "" {
			conditions = append(conditions, gcsql.FilterCondition{Field: "email", MatchMode: gcsql.SubstrMatch, Search: post.Email})
		}
		if post.Subject != "" {
			conditions = append(conditions, gcsql.FilterCondition{Field: "subject", MatchMode: gcsql.SubstrMatch, Search: post.Subject})
		}
		if post.IsTopPost {
			conditions = append(conditions, gcsql.FilterCondition{Field: "isop", MatchMode: gcsql.SubstrMatch})
		} else {
			conditions = append(conditions, gcsql.FilterCondition{Field: "notop", MatchMode: gcsql.SubstrMatch})
		}
		if post.MessageRaw != "" {
			conditions = append(conditions, gcsql.FilterCondition{Field: "body", MatchMode: gcsql.SubstrMatch, Search: post.MessageRaw})
		}
		upload, err := post.GetUpload()
		if err != nil {
			errEv.Err(err).Caller().Send()
			return nil, errors.New("unable to check post for uploaded file")
		}
		if upload == nil {
			conditions = append(conditions, gcsql.FilterCondition{Field: "nofile", MatchMode: gcsql.SubstrMatch})
		} else {
			fingerprint, _ := uploads.GetPostImageFingerprint(postID)
			conditions = append(conditions,
				gcsql.FilterCondition{Field: "hasfile", MatchMode: gcsql.SubstrMatch},
				gcsql.FilterCondition{Field: "filename", MatchMode: gcsql.SubstrMatch, Search: upload.OriginalFilename},
				gcsql.FilterCondition{Field: "checksum", MatchMode: gcsql.ExactMatch, Search: upload.Checksum},
				gcsql.FilterCondition{Field: "ahash", MatchMode: gcsql.ExactMatch, Search: fingerprint},
			)
		}

		opID, opBoard, err := gcsql.GetTopPostAndBoardDirFromPostID(postID)
		if err != nil {
			errEv.Err(err).Caller().Int("postID", postID).Msg("unable to get top post and board")
			return nil, errors.New("unable to get top post and board")
		}
		if opID == 0 || opBoard == "" {
			err = errors.New("post or board does not exist")
			errEv.Err(err).Caller().Int("postID", postID).Send()
			return nil, err
		}
		data["cancelURL"] = config.WebPath(opBoard, "res", strconv.Itoa(opID)+".html#"+strconv.Itoa(postID))
		data["sourcePostID"] = postID
		data["sourcePostBoard"] = opBoard
		data["sourcePostThread"] = opID
		filter = &gcsql.Filter{
			MatchAction: "reject",
		}
	} else if editFilter := request.FormValue("edit"); editFilter != "" {
		// user clicked on Edit link in filter row
		filterID, err := strconv.Atoi(editFilter)
		if err != nil {
			errEv.Err(err).Caller().Str("filterID", editFilter).Send()
			return nil, err
		}
		if filter, err = gcsql.GetFilterByID(filterID); err != nil {
			errEv.Err(err).Caller().Int("filterID", filterID).Send()
			return nil, errors.New("unable to get filter")
		}
		if conditions, err = filter.Conditions(); err != nil {
			errEv.Err(err).Caller().Int("filterID", filterID).Msg("Unable to get filter conditions")
			return nil, errors.New("unable to get filter conditions")
		}
		if boardIDs, err = filter.BoardIDs(); err != nil {
			errEv.Err(err).Caller().Msg("Unable to get filter board IDs")
			return nil, errors.New("unable to get filter board IDs")
		}
		data["cancelURL"] = config.WebPath("/manage/filters")
	} else {
		// user loaded /manage/filters, populate single "default" condition
		filter = &gcsql.Filter{
			MatchAction: "reject",
		}
		conditions = []gcsql.FilterCondition{
			{Field: "name"},
		}
	}
	data["filter"] = filter
	data["filterConditions"] = conditions
	data["filterBoards"] = boardIDs
	return data, nil
}

package manage

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

// manage actions that require moderator-level permission go here

var (
	filterFields = []filterField{
		{Value: "name", Text: "Name", hasRegex: true, hasSearchbox: true},
		{Value: "trip", Text: `Tripcode`, hasRegex: true, hasSearchbox: true},
		{Value: "email", Text: "Email", hasRegex: true, hasSearchbox: true},
		{Value: "subject", Text: "Subject", hasRegex: true, hasSearchbox: true},
		{Value: "body", Text: "Message body", hasRegex: true, hasSearchbox: true},
		{Value: "firsttimeboard", Text: "First time poster (board)"},
		{Value: "notfirsttimeboard", Text: "Not a first time poster (board)"},
		{Value: "firsttimesite", Text: "First time poster (site-wide)"},
		{Value: "notfirsttimesite", Text: "Not a first time poster (site-wide)"},
		{Value: "isop", Text: "Is OP"},
		{Value: "notop", Text: "Is reply"},
		{Value: "hasfile", Text: "Has file"},
		{Value: "nofile", Text: "No file"},
		{Value: "filename", Text: "Filename", hasRegex: true, hasSearchbox: true},
		{Value: "checksum", Text: "File checksum", hasSearchbox: true},
		{Value: "ahash", Text: "Image fingerprint", hasSearchbox: true},
		{Value: "useragent", Text: "User agent", hasRegex: true, hasSearchbox: true},
	}
	filterActionsMap = map[string]string{
		"reject": "Reject post",
		"ban":    "Ban IP",
		"log":    "Log match",
	}
)

func bansCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
	var outputStr string
	var ban gcsql.IPBan
	ban.StaffID = staff.ID
	deleteIDStr := request.FormValue("delete")
	postIDstr := request.FormValue("postid")
	if deleteIDStr != "" {
		// deleting a ban
		ban.ID, err = strconv.Atoi(deleteIDStr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("deleteBan", deleteIDStr).
				Send()
			return "", err
		}
		if err = ban.Deactivate(staff.ID); err != nil {
			errEv.Err(err).Caller().
				Int("deleteBan", ban.ID).
				Send()
			return "", err
		}

	} else if request.FormValue("do") == "add" {
		ip := request.PostFormValue("ip")
		ban.RangeStart, ban.RangeEnd, err = gcutil.ParseIPRange(ip)
		if err != nil {
			errEv.Err(err).Caller().
				Str("ip", ip)
			return "", err
		}
		gcutil.LogStr("rangeStart", ban.RangeStart, infoEv, errEv)
		gcutil.LogStr("rangeEnd", ban.RangeEnd, infoEv, errEv)
		gcutil.LogStr("reason", ban.Message, infoEv, errEv)
		gcutil.LogBool("appealable", ban.CanAppeal, infoEv, errEv)
		err := ipBanFromRequest(&ban, request, infoEv, errEv)
		if err != nil {
			errEv.Err(err).Caller().
				Msg("unable to submit ban")
			return "", err
		}
		infoEv.Msg("Added IP ban")
	} else if postIDstr != "" {
		postID, err := strconv.Atoi(postIDstr)
		if err != nil {
			errEv.Err(err).Caller().Send()
			return "", err
		}
		if ban.RangeStart, err = gcsql.GetPostIP(postID); err != nil {
			errEv.Err(err).Caller().
				Int("postID", postID).Send()
			return "", err
		}
		ban.RangeEnd = ban.RangeStart
	}

	filterBoardIDstr := request.FormValue("filterboardid")
	var filterBoardID int
	if filterBoardIDstr != "" {
		if filterBoardID, err = strconv.Atoi(filterBoardIDstr); err != nil {
			errEv.Err(err).Caller().
				Str("filterboardid", filterBoardIDstr).Send()
			return "", err
		}
	}
	limitStr := request.FormValue("limit")
	limit := 200
	if limitStr != "" {
		if limit, err = strconv.Atoi(limitStr); err != nil {
			errEv.Err(err).Caller().
				Str("limit", limitStr).Send()
			return "", err
		}
	}
	banlist, err := gcsql.GetIPBans(filterBoardID, limit, true)
	if err != nil {
		errEv.Err(err).Caller().Msg("Error getting ban list")
		err = errors.New("Error getting ban list: " + err.Error())
		return "", err
	}
	manageBansBuffer := bytes.NewBufferString("")

	if err = serverutil.MinifyTemplate(gctemplates.ManageBans, map[string]interface{}{
		"banlist":       banlist,
		"allBoards":     gcsql.AllBoards,
		"ban":           ban,
		"filterboardid": filterBoardID,
	}, manageBansBuffer, "text/html"); err != nil {
		errEv.Err(err).Str("template", "manage_bans.html").Caller().Send()
		return "", errors.New("Error executing ban management page template: " + err.Error())
	}
	outputStr += manageBansBuffer.String()
	return outputStr, nil
}

func appealsCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv, errEv *zerolog.Event) (output interface{}, err error) {
	banIDstr := request.FormValue("banid")
	var banID int
	if banIDstr != "" {
		if banID, err = strconv.Atoi(banIDstr); err != nil {
			errEv.Err(err).Caller().Send()
			return "", err
		}
	}
	infoEv.Int("banID", banID)

	limitStr := request.FormValue("limit")
	limit := 20
	if limitStr != "" {
		if limit, err = strconv.Atoi(limitStr); err != nil {
			errEv.Err(err).Caller().Send()
			return "", err
		}
	}
	approveStr := request.FormValue("approve")
	if approveStr != "" {
		// approving an appeal
		approveID, err := strconv.Atoi(approveStr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("approveStr", approveStr).Send()
		}
		if err = gcsql.ApproveAppeal(approveID, staff.ID); err != nil {
			errEv.Err(err).Caller().
				Int("approveAppeal", approveID).Send()
			return "", err
		}
	}

	appeals, err := gcsql.GetAppeals(banID, limit)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return "", errors.New("Unable to get appeals: " + err.Error())
	}

	if wantsJSON {
		return appeals, nil
	}
	manageAppealsBuffer := bytes.NewBufferString("")
	pageData := map[string]interface{}{}
	if len(appeals) > 0 {
		pageData["appeals"] = appeals
	}
	if err = serverutil.MinifyTemplate(gctemplates.ManageAppeals, pageData, manageAppealsBuffer, "text/html"); err != nil {
		errEv.Err(err).Str("template", "manage_appeals.html").Caller().Send()
		return "", errors.New("Error executing appeal management page template: " + err.Error())
	}
	return manageAppealsBuffer.String(), err
}

type filterField struct {
	Value        string
	Text         string
	hasRegex     bool
	hasSearchbox bool
}

func filtersCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv, errEv *zerolog.Event) (output any, err error) {
	var boards []int
	data := map[string]any{
		"allBoards": gcsql.AllBoards,
		"fields":    filterFields,
		"actions":   filterActionsMap,
	}
	boardIDLogArr := zerolog.Arr()
	conditionsLogArr := zerolog.Arr()

	doFilterAdd := request.PostFormValue("dofilteradd") != ""
	doFilterEdit := request.PostFormValue("dofilteredit") != ""

	if disableFilterIDStr := request.FormValue("disable"); disableFilterIDStr != "" {
		disableFilterID, err := strconv.Atoi(disableFilterIDStr)
		if err != nil {
			return nil, err
		}
		gcsql.SetFilterActive(disableFilterID, false)
	} else if enableFilterIDStr := request.FormValue("enable"); enableFilterIDStr != "" {
		enableFilterID, err := strconv.Atoi(enableFilterIDStr)
		if err != nil {
			return nil, err
		}
		gcsql.SetFilterActive(enableFilterID, true)
	} else if doFilterAdd || doFilterEdit {
		var conditions []gcsql.FilterCondition
		var filter *gcsql.Filter
		if doFilterAdd {
			// new post submitted
			filter = &gcsql.Filter{
				StaffID:  &staff.ID,
				IsActive: true,
			}
		} else if doFilterEdit {
			// post edit submitted
			filterIDstr := request.PostFormValue("filterid")
			filterID, err := strconv.Atoi(filterIDstr)
			if err != nil {
				errEv.Err(err).Caller().Str("filterID", filterIDstr).Msg("Unable to parse filter ID")
				return nil, err
			}
			if filter, err = gcsql.GetFilterByID(filterID); err != nil {
				errEv.Err(err).Caller().Int("filterID", filterID).Msg("Unable to get filter from ID")
				return nil, err
			}
			if conditions, err = filter.Conditions(); err != nil {
				errEv.Err(err).Caller().Int("filterID", filterID).Msg("Unable to get filter conditions list")
			}
		}

		for k, v := range request.PostForm {
			// set filter boards
			if strings.HasPrefix(k, "applyboard") && v[0] == "on" {
				boardID, err := strconv.Atoi(k[10:])
				if err != nil {
					errEv.Err(err).Caller().
						Str("boardIDField", k).
						Str("boardIDStr", k[10:]).
						Msg("Unable to parse board ID")
					return nil, errors.New("unable to parse board ID: " + err.Error())
				}
				boardIDLogArr.Int(boardID)
				boards = append(boards, boardID)
			}
			infoEv.Array("boardIDs", boardIDLogArr)

			// set filter conditions
			if strings.HasPrefix(k, "field") {
				fieldIDstr := k[5:]
				if _, err = strconv.Atoi(fieldIDstr); err != nil {
					errEv.Err(err).Caller().Str("fieldID", fieldIDstr).Send()
					return nil, errors.New("failed to get field data: " + err.Error())
				}
				fc := gcsql.FilterCondition{
					Field:   v[0],
					IsRegex: request.PostFormValue("isregex"+fieldIDstr) == "on",
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
					return nil, gcsql.ErrInvalidConditionField
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
			return nil, err
		}
	}

	if editFilter := request.FormValue("edit"); editFilter != "" {
		// user clicked on Edit link in filter row
		filterID, err := strconv.Atoi(editFilter)
		if err != nil {
			errEv.Err(err).Caller().Str("filterID", editFilter).Send()
			return nil, err
		}
		filter, err := gcsql.GetFilterByID(filterID)
		if err != nil {
			errEv.Err(err).Caller().Int("filterID", filterID).Send()
			return nil, errors.New("unable to get filter")
		}
		data["filter"] = filter
		if data["filterConditions"], err = filter.Conditions(); err != nil {
			errEv.Err(err).Caller().Int("filterID", filterID).Msg("Unable to get filter conditions")
			return nil, errors.New("unable to get filter conditions")
		}
	} else {
		// user loaded /manage/filters, populate single "default" condition
		data["filter"] = &gcsql.Filter{
			MatchAction: "reject",
		}
		data["filterConditions"] = []gcsql.FilterCondition{
			{Field: "name"},
		}
	}

	showStr := request.FormValue("show")
	var show gcsql.ActiveFilter
	switch showStr {
	case "inactive":
		show = gcsql.OnlyInactiveFilters
	case "all":
		show = gcsql.AllFilters
	default:
		show = gcsql.OnlyActiveFilters
	}
	var filters []gcsql.Filter
	boardSearch := request.FormValue("boardsearch")
	if boardSearch == "" {
		filters, err = gcsql.GetAllFilters(show)
	} else {
		filters, err = gcsql.GetFiltersByBoardDir(boardSearch, false, show)
	}

	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to get filter list")
		return nil, err
	}
	fieldsMap := make(map[string]string)
	for _, ff := range filterFields {
		fieldsMap[ff.Value] = ff.Text
	}
	var staffUsernames []string

	var conditionsText []string
	var boardsText []string
	for _, filter := range filters {
		if _, ok := filterActionsMap[filter.MatchAction]; !ok {
			return nil, gcsql.ErrInvalidMatchAction
		}
		conditions, err := filter.Conditions()
		if err != nil {
			errEv.Err(err).Caller().Int("filterID", filter.ID).Msg("Unable to get filter conditions")
			return nil, err
		}

		var filterConditionsText string
		for _, condition := range conditions {
			text, ok := fieldsMap[condition.Field]
			if !ok {
				return nil, gcsql.ErrInvalidConditionField
			}
			filterConditionsText += text + ","
		}
		filterConditionsText = strings.TrimRight(filterConditionsText, ",")
		conditionsText = append(conditionsText, filterConditionsText)

		boards, err := filter.BoardDirs()
		if err != nil {
			return nil, err
		}
		boardsText = append(boardsText, strings.Join(boards, ","))

		username, err := gcsql.GetStaffUsernameFromID(*filter.StaffID)
		if err != nil {
			return nil, err
		}
		staffUsernames = append(staffUsernames, username)
	}

	data["filters"] = filters
	data["conditions"] = conditionsText
	data["filterTableBoards"] = boardsText
	data["staff"] = staffUsernames
	data["show"] = showStr
	data["boardSearch"] = boardSearch

	var buf bytes.Buffer
	if err = serverutil.MinifyTemplate(gctemplates.ManageFilters, data, &buf, "text/html"); err != nil {
		errEv.Err(err).Caller().Str("template", gctemplates.ManageFilters).Send()
		return "", errors.New("Unable to execute filter management template: " + err.Error())
	}
	return buf.String(), nil
}

func ipSearchCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, _ *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
	ipQuery := request.FormValue("ip")
	limitStr := request.FormValue("limit")
	data := map[string]interface{}{
		"ipQuery": ipQuery,
		"limit":   20,
	}

	if ipQuery != "" && limitStr != "" {
		var limit int
		if limit, err = strconv.Atoi(limitStr); err == nil && limit > 0 {
			data["limit"] = limit
		}
		var names []string
		if names, err = net.LookupAddr(ipQuery); err == nil {
			data["reverseAddrs"] = names
		} else {
			data["reverseAddrs"] = []string{err.Error()}
		}

		data["posts"], err = building.GetBuildablePostsByIP(ipQuery, limit)
		if err != nil {
			errEv.Err(err).Caller().
				Str("ipQuery", ipQuery).
				Int("limit", limit).
				Bool("onlyNotDeleted", true).
				Send()
			return "", fmt.Errorf("Error getting list of posts from %q by staff %s: %s", ipQuery, staff.Username, err.Error())
		}
	}

	manageIpBuffer := bytes.NewBufferString("")
	if err = serverutil.MinifyTemplate(gctemplates.ManageIPSearch, data, manageIpBuffer, "text/html"); err != nil {
		errEv.Err(err).Caller().
			Str("template", "manage_ipsearch.html").Send()
		return "", errors.New("Error executing IP search page template:" + err.Error())
	}
	return manageIpBuffer.String(), nil
}

func reportsCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
	dismissIDstr := request.FormValue("dismiss")
	if dismissIDstr != "" {
		// staff is dismissing a report
		dismissID := gcutil.HackyStringToInt(dismissIDstr)
		block := request.FormValue("block")
		if block != "" && staff.Rank != 3 {
			errEv.Caller().
				Int("postID", dismissID).
				Str("rejected", "not an admin").Send()
			return "", errors.New("only the administrator can block reports")
		}
		found, err := gcsql.ClearReport(dismissID, staff.ID, block != "" && staff.Rank == 3)
		if err != nil {
			errEv.Err(err).Caller().
				Int("postID", dismissID).Send()
			return nil, err
		}
		if !found {
			return nil, errors.New("no matching reports")
		}
		infoEv.
			Int("reportID", dismissID).
			Bool("blocked", block != "").
			Msg("Report cleared")
	}
	rows, err := gcsql.QuerySQL(`SELECT id,
		handled_by_staff_id as staff_id,
		(SELECT username FROM DBPREFIXstaff WHERE id = DBPREFIXreports.handled_by_staff_id) as staff_user,
		post_id, IP_NTOA, reason, is_cleared from DBPREFIXreports WHERE is_cleared = FALSE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reports := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int
		var staff_id interface{}
		var staff_user []byte
		var post_id int
		var ip string
		var reason string
		var is_cleared int
		err = rows.Scan(&id, &staff_id, &staff_user, &post_id, &ip, &reason, &is_cleared)
		if err != nil {
			return nil, err
		}

		post, err := gcsql.GetPostFromID(post_id, true)
		if err != nil {
			return nil, err
		}

		staff_id_int, _ := staff_id.(int64)
		reports = append(reports, map[string]interface{}{
			"id":         id,
			"staff_id":   int(staff_id_int),
			"staff_user": string(staff_user),
			"post_link":  post.WebPath(),
			"ip":         ip,
			"reason":     reason,
			"is_cleared": is_cleared,
		})
	}
	if wantsJSON {
		return reports, err
	}
	reportsBuffer := bytes.NewBufferString("")
	err = serverutil.MinifyTemplate(gctemplates.ManageReports,
		map[string]interface{}{
			"reports": reports,
			"staff":   staff,
		}, reportsBuffer, "text/html")
	if err != nil {
		errEv.Err(err).Caller().Send()
		return "", err
	}
	output = reportsBuffer.String()
	return
}

func threadAttrsCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, wantsJSON bool, infoEv, errEv *zerolog.Event) (output interface{}, err error) {
	boardDir := request.FormValue("board")
	attrBuffer := bytes.NewBufferString("")
	data := map[string]interface{}{
		"boards": gcsql.AllBoards,
	}
	if boardDir == "" {
		if wantsJSON {
			return nil, errors.New(`missing required field "board"`)
		}
		if err = serverutil.MinifyTemplate(gctemplates.ManageThreadAttrs, data, attrBuffer, "text/html"); err != nil {
			errEv.Err(err).Caller().Send()
			return "", err
		}
		return attrBuffer.String(), nil
	}
	gcutil.LogStr("board", boardDir, errEv, infoEv)
	board, err := gcsql.GetBoardFromDir(boardDir)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return "", err
	}
	data["board"] = board
	topPostStr := request.FormValue("thread")
	if topPostStr != "" {
		var topPostID int
		if topPostID, err = strconv.Atoi(topPostStr); err != nil {
			errEv.Err(err).Str("topPostStr", topPostStr).Caller().Send()
			return "", err
		}
		gcutil.LogInt("topPostID", topPostID, errEv, infoEv)
		data["topPostID"] = topPostID
		var attr string
		var newVal bool
		var doChange bool // if false, don't bother executing any SQL since nothing will change
		thread, err := gcsql.GetPostThread(topPostID)
		if err != nil {
			errEv.Err(err).Caller().Send()
			return "", err
		}
		if request.FormValue("unlock") != "" {
			attr = "locked"
			newVal = false
			doChange = thread.Locked != newVal
		} else if request.FormValue("lock") != "" {
			attr = "locked"
			newVal = true
			doChange = thread.Locked != newVal
		} else if request.FormValue("unsticky") != "" {
			attr = "stickied"
			newVal = false
			doChange = thread.Stickied != newVal
		} else if request.FormValue("sticky") != "" {
			attr = "stickied"
			newVal = true
			doChange = thread.Stickied != newVal
		} else if request.FormValue("unanchor") != "" {
			attr = "anchored"
			newVal = false
			doChange = thread.Anchored != newVal
		} else if request.FormValue("anchor") != "" {
			attr = "anchored"
			newVal = true
			doChange = thread.Anchored != newVal
		} else if request.FormValue("uncyclical") != "" {
			attr = "cyclical"
			newVal = false
			doChange = thread.Cyclical != newVal
		} else if request.FormValue("cyclical") != "" {
			attr = "cyclical"
			newVal = true
			doChange = thread.Cyclical != newVal
		}

		if attr != "" && doChange {
			gcutil.LogStr("attribute", attr, errEv, infoEv)
			gcutil.LogBool("attrVal", newVal, errEv, infoEv)
			if err = thread.UpdateAttribute(attr, newVal); err != nil {
				errEv.Err(err).Caller().Send()
				return "", err
			}
			if err = building.BuildBoardPages(board); err != nil {
				return "", err
			}
			post, err := gcsql.GetPostFromID(topPostID, true)
			if err != nil {
				errEv.Err(err).Caller().Send()
				return "", err
			}
			if err = building.BuildThreadPages(post); err != nil {
				return "", err
			}
			fmt.Println("Done rebuilding", board.Dir)
		}
		data["thread"] = thread
	}

	threads, err := gcsql.GetThreadsWithBoardID(board.ID, true)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return "", err
	}
	data["threads"] = threads
	var threadIDs []interface{}
	for i := len(threads) - 1; i >= 0; i-- {
		threadIDs = append(threadIDs, threads[i].ID)
	}
	if wantsJSON {
		return threads, nil
	}

	opMap, err := gcsql.GetTopPostIDsInThreadIDs(threadIDs...)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return "", err
	}
	data["opMap"] = opMap
	var formURL url.URL
	formURL.Path = config.WebPath("/manage/threadattrs")
	vals := formURL.Query()
	vals.Set("board", boardDir)
	if topPostStr != "" {
		vals.Set("thread", topPostStr)
	}
	formURL.RawQuery = vals.Encode()
	data["formURL"] = formURL.String()
	if err = serverutil.MinifyTemplate(gctemplates.ManageThreadAttrs, data, attrBuffer, "text/html"); err != nil {
		errEv.Err(err).Caller().Send()
		return "", err
	}
	return attrBuffer.String(), nil
}

type postInfoJSON struct {
	Post *gcsql.Post `json:"post"`
	FQDN []string    `json:"ipFQDN"`

	OriginalFilename string `json:"originalFilename,omitempty"`
	Checksum         string `json:"checksum,omitempty"`
	Fingerprint      string `json:"fingerprint,omitempty"`
}

func postInfoCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, _ *zerolog.Event, _ *zerolog.Event) (output interface{}, err error) {
	postIDstr := request.FormValue("postid")
	if postIDstr == "" {
		return "", errors.New("invalid request (missing postid)")
	}
	var postID int
	if postID, err = strconv.Atoi(postIDstr); err != nil {
		return "", err
	}
	post, err := gcsql.GetPostFromID(postID, true)
	if err != nil {
		return "", err
	}

	postInfo := postInfoJSON{
		Post: post,
	}
	names, err := net.LookupAddr(post.IP)
	if err == nil {
		postInfo.FQDN = names
	} else {
		postInfo.FQDN = []string{err.Error()}
	}
	upload, err := post.GetUpload()
	if err != nil {
		return "", err
	}
	if upload != nil {
		postInfo.OriginalFilename = upload.OriginalFilename
		postInfo.Checksum = upload.Checksum
		postInfo.Fingerprint, err = uploads.GetPostImageFingerprint(postID)
		if err != nil {
			return "", err
		}
	}
	return postInfo, nil
}

type fingerprintJSON struct {
	Fingerprint string `json:"fingerprint"`
}

func fingerprintCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, _ *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
	postIDstr := request.Form.Get("post")
	if postIDstr == "" {
		return "", errors.New("missing 'post' field")
	}
	postID, err := strconv.Atoi(postIDstr)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return "", err
	}
	fingerprint, err := uploads.GetPostImageFingerprint(postID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("post has no files or post doesn't exist")
	} else if err != nil {
		errEv.Err(err).Caller().Send()
		return "", err
	}
	return fingerprintJSON{
		Fingerprint: fingerprint,
	}, nil
}

func wordfiltersCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
	do := request.PostFormValue("dowordfilter")
	editIDstr := request.FormValue("edit")
	disableIDstr := request.FormValue("disable")
	enableIDstr := request.FormValue("enable")

	if disableIDstr != "" {
		disableID, err := strconv.Atoi(disableIDstr)
		if err != nil {
			errEv.Err(err).Caller().Str("disableID", disableIDstr).Send()
			return nil, err
		}
		if err = gcsql.SetFilterActive(disableID, false); err != nil {
			errEv.Err(err).Caller().Int("disableID", disableID).Msg("Unable to disable filter")
			return nil, errors.New("unable to disable wordfilter")
		}
		infoEv.Int("disableID", disableID)
	} else if enableIDstr != "" {
		enableID, err := strconv.Atoi(enableIDstr)
		if err != nil {
			errEv.Err(err).Caller().Str("enableID", enableIDstr).Send()
			return nil, err
		}
		if err = gcsql.SetFilterActive(enableID, true); err != nil {
			errEv.Err(err).Caller().Int("enableID", enableID).Msg("Unable to enable filter")
			return nil, errors.New("unable to enable wordfilter")
		}
		infoEv.Int("enableID", enableID)
	}

	var filter *gcsql.Wordfilter
	if editIDstr != "" {
		editID, err := strconv.Atoi(editIDstr)
		if err != nil {
			errEv.Err(err).Str("editID", editIDstr).Send()
			return nil, err
		}
		gcutil.LogInt("editID", editID, infoEv, errEv)

		filter, err = gcsql.GetWordfilterByID(editID)
		if err != nil {
			errEv.Err(err).Caller().Msg("Unable to get wordfilter")
			return nil, fmt.Errorf("Unable to get wordfilter with id #%d", editID)
		}
	}
	searchFor := request.PostFormValue("searchfor")
	replaceWith := request.PostFormValue("replace")
	isRegex := request.PostFormValue("isregex") == "on"
	staffNote := request.PostFormValue("staffnote")

	var boards []string
	boardsLog := zerolog.Arr()
	for k, v := range request.PostForm {
		if strings.HasPrefix(k, "board-") && v[0] == "on" {
			boards = append(boards, k[6:])
			boardsLog.Str(k[6:])
		}
	}
	if do != "" {
		infoEv.Array("boards", boardsLog)
		errEv.Array("boards", boardsLog)
		gcutil.LogStr("searchFor", searchFor, infoEv, errEv)
		gcutil.LogStr("replaceWith", replaceWith, infoEv, errEv)
		gcutil.LogStr("staffnote", staffNote, infoEv, errEv)
		gcutil.LogBool("isRegex", isRegex, infoEv, errEv)
	}

	switch do {
	case "Edit wordfilter":
		if err = filter.UpdateDetails(staffNote, "replace", replaceWith); err != nil {
			errEv.Err(err).Caller().Msg("Unable to update wordfilter details")
			return nil, errors.New("unable to update wordfilter details")
		}
		if err = filter.SetConditions(gcsql.FilterCondition{
			FilterID: filter.ID,
			IsRegex:  isRegex,
			Search:   searchFor,
			Field:    "body",
		}); err != nil {
			errEv.Err(err).Caller().Msg("Unable to set filter condition")
			return nil, errors.New("unable to set filter conditions")
		}
		if err = filter.SetBoardDirs(boards...); err != nil {
			errEv.Err(err).Caller().Msg("Unable to set board directories")
			return nil, errors.New("unable to set board directories")
		}
		infoEv.Str("do", "update")
	case "Create wordfilter":
		if _, err = gcsql.CreateWordFilter(searchFor, replaceWith, isRegex, boards, staff.ID, staffNote); err != nil {
			errEv.Err(err).Caller().Msg("Unable to create wordfilter")
			return nil, errors.New("unable to create wordfilter")
		}
		infoEv.Str("do", "create")
	}

	wordfilters, err := gcsql.GetWordfilters(gcsql.AllFilters)
	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to get wordfilters")
		return nil, err
	}
	var searchFields []string
	for _, wordfilter := range wordfilters {
		conditions, err := wordfilter.Conditions()
		if err != nil {
			return nil, err
		}
		if err = wordfilter.VerifySingleCondition(conditions); err != nil {
			return nil, err
		}
		searchFields = append(searchFields, conditions[0].Search)
	}

	var buf bytes.Buffer
	if err = serverutil.MinifyTemplate(gctemplates.ManageWordfilters, map[string]any{
		"wordfilters":  wordfilters,
		"filter":       filter,
		"searchFields": searchFields,
		"allBoards":    gcsql.AllBoards,
	}, &buf, "text/html"); err != nil {
		errEv.Err(err).Str("template", "manage_wordfilters.html").Caller().Send()
		return nil, err
	}
	if do != "" || enableIDstr != "" || disableIDstr != "" {
		infoEv.Send()
	}
	return buf.String(), nil
}

func registerModeratorPages() {
	actions = append(actions,
		Action{
			ID:          "bans",
			Title:       "Bans",
			Permissions: ModPerms,
			Callback:    bansCallback,
		},
		Action{
			ID:          "appeals",
			Title:       "Ban appeals",
			Permissions: ModPerms,
			JSONoutput:  OptionalJSON,
			Callback:    appealsCallback,
		},
		Action{
			ID:          "filters",
			Title:       "Post filters",
			Permissions: ModPerms,
			JSONoutput:  NoJSON,
			Callback:    filtersCallback,
		},
		Action{
			ID:          "ipsearch",
			Title:       "IP Search",
			Permissions: ModPerms,
			JSONoutput:  NoJSON,
			Callback:    ipSearchCallback,
		},
		Action{
			ID:          "reports",
			Title:       "Reports",
			Permissions: ModPerms,
			JSONoutput:  OptionalJSON,
			Callback:    reportsCallback,
		},
		Action{
			ID:          "threadattrs",
			Title:       "View/Update Thread Attributes",
			Permissions: ModPerms,
			JSONoutput:  OptionalJSON,
			Callback:    threadAttrsCallback,
		},
		Action{
			ID:          "postinfo",
			Title:       "Post info",
			Permissions: ModPerms,
			JSONoutput:  AlwaysJSON,
			Callback:    postInfoCallback,
		},
		Action{
			ID:          "fingerprint",
			Title:       "Get image/thumbnail fingerprint",
			Permissions: ModPerms,
			JSONoutput:  AlwaysJSON,
			Callback:    fingerprintCallback,
		},
		Action{
			ID:          "wordfilters",
			Title:       "Wordfilters",
			Permissions: ModPerms,
			Callback:    wordfiltersCallback,
		},
	)
}

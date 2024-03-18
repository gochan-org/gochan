package manage

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

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

func fileBansCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv, errEv *zerolog.Event) (output interface{}, err error) {
	delFilenameBanIDStr := request.FormValue("delfnb") // filename ban deletion
	delChecksumBanIDStr := request.FormValue("delcsb") // checksum ban deletion

	boardidStr := request.FormValue("boardid")
	boardid := 0
	if boardidStr != "" {
		boardid, err = strconv.Atoi(boardidStr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("boardid", boardidStr).Send()
			return "", err
		}
	}
	gcutil.LogInt("boardid", boardid, infoEv, errEv)
	staffnote := request.FormValue("staffnote")

	if request.FormValue("dofilenameban") != "" {
		// creating a new filename ban
		filename := request.FormValue("filename")
		isRegex := request.FormValue("isregex") == "on"
		if isRegex {
			_, err = regexp.Compile(filename)
			if err != nil {
				// invalid regular expression
				errEv.Err(err).Caller().
					Str("regex", filename).Send()
				return "", err
			}
		}
		if _, err = gcsql.NewFilenameBan(filename, isRegex, boardid, staff.ID, staffnote); err != nil {
			errEv.Err(err).Caller().
				Str("filename", filename).
				Bool("isregex", isRegex).Send()
			return "", err
		}
		infoEv.
			Str("filename", filename).
			Bool("isregex", isRegex).
			Msg("Created new filename ban")
		if wantsJSON {
			return "success", nil
		}
	} else if delFilenameBanIDStr != "" {
		delFilenameBanID, err := strconv.Atoi(delFilenameBanIDStr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("delfnb", delFilenameBanIDStr).
				Send()
			return "", err
		}
		var fnb gcsql.FilenameBan
		fnb.ID = delFilenameBanID
		if err = fnb.Deactivate(staff.ID); err != nil {
			errEv.Err(err).Caller().
				Int("deleteFilenameBanID", delFilenameBanID).Send()
			return "", err
		}
		infoEv.
			Int("deleteFilenameBanID", delFilenameBanID).
			Msg("Filename ban deleted")
		if wantsJSON {
			return "success", nil
		}
	} else if request.PostFormValue("dochecksumban") != "" {
		// creating a new file checksum ban
		checksum := request.PostFormValue("checksum")
		ipBan := request.PostFormValue("ban") == "on"
		var reason string
		if ipBan {
			reason = request.PostFormValue("banmsg")
			if reason == "" {
				return "", errors.New("ban reason required if IP ban is set")
			}
		}
		gcutil.LogBool("ipBan", ipBan, infoEv, errEv)
		fingerprinter := request.PostFormValue("fingerprinter")
		if fingerprinter == "checksum" {
			fingerprinter = ""
		}
		gcutil.LogStr("fingerprinter", fingerprinter, infoEv, errEv)
		if _, err = gcsql.NewFileChecksumBan(
			checksum, fingerprinter, boardid, staff.ID, staffnote, ipBan, reason,
		); err != nil {
			errEv.Err(err).Caller().
				Str("checksum", checksum).Send()
			return "", err
		}
		infoEv.
			Str("checksum", checksum).
			Msg("Created new file checksum ban")
		if wantsJSON {
			return "success", nil
		}
	} else if delChecksumBanIDStr != "" {
		// user requested a checksum ban ID to delete
		delChecksumBanID, err := strconv.Atoi(delChecksumBanIDStr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("deleteChecksumBanIDStr", delChecksumBanIDStr).Send()
			return "", err
		}
		if err = (gcsql.FileBan{ID: delChecksumBanID}).Deactivate(staff.ID); err != nil {
			errEv.Err(err).Caller().
				Int("deleteChecksumBanID", delChecksumBanID).Send()
			return "", err
		}
		infoEv.Int("deleteChecksumBanID", delChecksumBanID).Msg("File checksum ban deleted")
		if wantsJSON {
			return "success", nil
		}
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
	checksumBans, err := gcsql.GetFileBans(filterBoardID, limit)
	if err != nil {
		return "", err
	}
	filenameBans, err := gcsql.GetFilenameBans(filterBoardID, limit)
	if err != nil {
		return "", err
	}
	manageBansBuffer := bytes.NewBufferString("")

	if err = serverutil.MinifyTemplate(gctemplates.ManageFileBans, map[string]interface{}{
		"allBoards":     gcsql.AllBoards,
		"checksumBans":  checksumBans,
		"filenameBans":  filenameBans,
		"filterboardid": filterBoardID,
		"currentStaff":  staff.Username,
	}, manageBansBuffer, "text/html"); err != nil {
		errEv.Err(err).Str("template", "manage_filebans.html").Caller().Send()
		return "", errors.New("Error executing ban management page template: " + err.Error())
	}
	outputStr := manageBansBuffer.String()
	return outputStr, nil
}

func nameBansCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, _, errEv *zerolog.Event) (output interface{}, err error) {
	doNameBan := request.FormValue("donameban")
	deleteIDstr := request.FormValue("del")
	if deleteIDstr != "" {
		deleteID, err := strconv.Atoi(deleteIDstr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("delStr", deleteIDstr).Send()
			return "", err
		}
		if err = gcsql.DeleteNameBan(deleteID); err != nil {
			errEv.Err(err).Caller().
				Int("deleteID", deleteID).
				Msg("Unable to delete name ban")
			return "", errors.New("Unable to delete name ban: " + err.Error())
		}
	}
	data := map[string]interface{}{
		"currentStaff": staff.Username,
		"allBoards":    gcsql.AllBoards,
	}
	if doNameBan == "Create" {
		var name string
		if name, err = getStringField("name", staff.Username, request); err != nil {
			return "", err
		}
		if name == "" {
			return "", errors.New("name field must not be empty in name ban submission")
		}
		var boardID int
		if boardID, err = getIntField("boardid", staff.Username, request); err != nil {
			return "", err
		}
		isRegex := request.FormValue("isregex") == "on"
		if _, err = gcsql.NewNameBan(name, isRegex, boardID, staff.ID, request.FormValue("staffnote")); err != nil {
			errEv.Err(err).Caller().
				Str("name", name).
				Int("boardID", boardID).Send()
			return "", err
		}
	}
	if data["nameBans"], err = gcsql.GetNameBans(0, 0); err != nil {
		return "", err
	}
	buf := bytes.NewBufferString("")
	if err = serverutil.MinifyTemplate(gctemplates.ManageNameBans, data, buf, "text/html"); err != nil {
		errEv.Err(err).Str("template", "manage_namebans.html").Caller().Send()
		return "", errors.New("Error executing name ban management page template: " + err.Error())
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
			errEv.
				Int("postID", dismissID).
				Str("rejected", "not an admin").
				Caller().Send()
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
			ID:          "filebans",
			Title:       "Filename and checksum bans",
			Permissions: ModPerms,
			JSONoutput:  OptionalJSON,
			Callback:    fileBansCallback,
		},
		Action{
			ID:          "namebans",
			Title:       "Name bans",
			Permissions: ModPerms,
			Callback:    nameBansCallback,
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
	)
}

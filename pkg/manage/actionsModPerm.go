package manage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Eggbertx/go-forms"
	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"
)

// manage actions that require moderator-level permission go here

func bansCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	var outputStr string
	var ban gcsql.IPBan
	ban.StaffID = staff.ID

	var banForm banPageFields
	if err = forms.FillStructFromForm(request, &banForm); err != nil {
		logger.Err(err).Caller().
			Msg("Unable to fill struct from form")
		return "", server.NewServerError("received invalid form data", http.StatusBadRequest)
	}

	if banForm.PostID > 0 {
		ban.BannedForPostID = new(int)
		*ban.BannedForPostID = banForm.PostID
		logger = logger.With().Int("postID", banForm.PostID).Logger()
	}

	if banForm.DeleteID > 0 {
		// deleting a ban
		ban.ID = banForm.DeleteID
		if err = ban.Deactivate(staff.ID); err != nil {
			logger.Err(err).Caller().
				Int("deleteBan", ban.ID).
				Send()
			return "", err
		}
	} else if banForm.Do == "add" {
		err := banForm.fillBanFields(&ban, logger.Info(), logger.Error())
		if err != nil {
			return "", err
		}
		if err = gcsql.NewIPBan(&ban); err != nil {
			logger.Err(err).Caller().
				Msg("Unable to create new IP ban")
			return "", server.NewServerError("failed to create new IP ban", http.StatusInternalServerError)
		}
		logger = logger.With().Int("banID", ban.ID).Logger()

		if banForm.UseBannedMessage && banForm.BannedMessage != "" {
			if err = gcsql.SetPostBannedMessage(banForm.PostID, banForm.BannedMessage, staff.Username); err != nil {
				logger.Err(err).Caller().
					Str("bannedMessage", banForm.BannedMessage).
					Msg("Unable to set banned message")
				return "", server.NewServerError("failed to set banned message", http.StatusInternalServerError)
			}

			board, err := ban.BannedPostBoard()
			if err != nil {
				logger.Err(err).Caller().
					Int("postID", *ban.BannedForPostID).
					Msg("Unable to get board from banned post")
				return "", server.NewServerError("failed to get board from banned post", http.StatusInternalServerError)
			}
			if board == nil {
				logger.Warn().Caller().
					Int("postID", *ban.BannedForPostID).
					Msg("Unable to get board from banned post (ban.BannedPostBoard() returned nil board)")
				return "", server.NewServerError("failed to get board from banned post", http.StatusInternalServerError)
			}
			logger = logger.With().Str("board", board.Dir).Logger()
			if err = building.BuildBoards(true, board.ID); err != nil {
				logger.Err(err).Caller().
					Int("postID", *ban.BannedForPostID).
					Msg("Unable to rebuild board")
				return "", server.NewServerError("failed to rebuild board", http.StatusInternalServerError)
			}
		}
		logger.Info().Msg("Added IP ban")
	} else if banForm.PostID > 0 {
		if ban.RangeStart, err = gcsql.GetPostIP(banForm.PostID); err != nil {
			logger.Err(err).Caller().
				Int("postID", banForm.PostID).Send()
			return "", err
		}
		ban.RangeEnd = ban.RangeStart
	}

	banlist, err := gcsql.GetIPBans(banForm.BoardID, banForm.Limit, true)
	if err != nil {
		logger.Err(err).Caller().Msg("Error getting ban list")
		err = fmt.Errorf("failed getting ban list: %w", err)
		return "", err
	}
	manageBansBuffer := bytes.NewBufferString("")
	data := map[string]any{
		"banlist":       banlist,
		"allBoards":     gcsql.AllBoards,
		"ban":           ban,
		"filterboardid": banForm.FilterBoardID,
		"boardConfig":   config.GetBoardConfig(""),
	}
	if ban.BannedForPostID != nil {
		data["postID"] = banForm.PostID
	}

	if err = serverutil.MinifyTemplate(gctemplates.ManageBans, data, manageBansBuffer, "text/html"); err != nil {
		logger.Err(err).Str("template", gctemplates.ManageBans).Caller().Send()
		return "", fmt.Errorf("failed executing ban management page template: %w", err)
	}
	outputStr += manageBansBuffer.String()
	return outputStr, nil
}

func appealsCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, logger zerolog.Logger) (output any, err error) {
	banIDstr := request.FormValue("banid")
	var banID int
	if banIDstr != "" {
		if banID, err = strconv.Atoi(banIDstr); err != nil {
			logger.Err(err).Caller().Send()
			return "", err
		}
	}
	logger = logger.With().Int("banID", banID).Logger()

	limitStr := request.FormValue("limit")
	limit := 20
	if limitStr != "" {
		if limit, err = strconv.Atoi(limitStr); err != nil {
			logger.Err(err).Caller().Send()
			return "", err
		}
	}
	approveStr := request.FormValue("approve")
	if approveStr != "" {
		// approving an appeal
		approveID, err := strconv.Atoi(approveStr)
		if err != nil {
			logger.Err(err).Caller().
				Str("approveStr", approveStr).Send()
		}
		if err = gcsql.ApproveAppeal(approveID, staff.ID); err != nil {
			logger.Err(err).Caller().
				Int("approveAppeal", approveID).Send()
			return "", err
		}
	}

	appeals, err := gcsql.GetAppeals(banID, limit)
	if err != nil {
		logger.Err(err).Caller().Send()
		return "", fmt.Errorf("failed to get appeals list: %w", err)
	}

	if wantsJSON {
		return appeals, nil
	}
	manageAppealsBuffer := bytes.NewBufferString("")
	pageData := map[string]any{}
	if len(appeals) > 0 {
		pageData["appeals"] = appeals
	}
	if err = serverutil.MinifyTemplate(gctemplates.ManageAppeals, pageData, manageAppealsBuffer, "text/html"); err != nil {
		logger.Err(err).Str("template", gctemplates.ManageAppeals).Caller().Send()
		return "", fmt.Errorf("failed executing appeal management page template: %w", err)
	}
	return manageAppealsBuffer.String(), err
}

func filterHitsCallback(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	params, _ := request.Context().Value(requestContextKey{}).(bunrouter.Params)
	filterIDStr := params.ByName("filterID")
	filterID, err := strconv.Atoi(filterIDStr)
	if err != nil {
		logger.Err(err).Caller().Str("filterID", filterIDStr).Msg("Filter ID is not a valid integer")
		return nil, err
	}
	logger = logger.With().Int("filterID", filterID).Logger()
	if request.Method == http.MethodPost && request.PostFormValue("clearhits") == "Clear hits" {
		if staff.Rank < 3 {
			writer.WriteHeader(http.StatusForbidden)
			return nil, ErrInsufficientPermission
		}
		if err = gcsql.ClearFilterHits(filterID); err != nil {
			logger.Err(err).Caller().Send()
			return nil, errors.New("unable to clear filter hits")
		}
		logger = logger.With().Int("filterID", filterID).Logger()
	}

	hits, err := gcsql.GetFilterHits(filterID)
	if err != nil {
		logger.Err(err).Caller().Msg("Unable to get filter hits")
		return nil, errors.New("unable to get list of filter hits")
	}
	m := make(map[string]any)
	var jsonBuf bytes.Buffer
	encoder := json.NewEncoder(&jsonBuf)
	encoder.SetEscapeHTML(true)
	encoder.SetIndent("", "&ensp;&ensp;&ensp;")
	var hitsJSON []template.HTML
	for _, hit := range hits {
		jsonBuf.Reset()
		// un-minify the JSON data to make it more readable
		if err = json.Unmarshal([]byte(hit.PostData), &m); err != nil {
			logger.Err(err).Caller().Msg("Unable to unmarshal post data for filter hit")
			return nil, err
		}
		if err = encoder.Encode(m); err != nil {
			logger.Err(err).Caller().RawJSON("postData", []byte(hit.PostData)).Msg("Unable to marshal un-minified post data")
			return nil, err
		}
		hitsJSON = append(hitsJSON, template.HTML(strings.ReplaceAll(jsonBuf.String(), "\n", "<br>"))) // skipcq: GSC-G203
	}
	var buf bytes.Buffer
	if err = serverutil.MinifyTemplate(gctemplates.ManageFilterHits, map[string]any{
		"staff":    staff,
		"filterID": filterID,
		"hits":     hits,
		"hitsJSON": hitsJSON,
	}, &buf, "text/html"); err != nil {
		logger.Err(err).Caller().Str("template", gctemplates.ManageFilterHits).Msg("Unable to render template")
		return nil, errors.New("unable to render filter hits page")
	}

	return buf.String(), nil
}

type filterField struct {
	Value        string
	Text         string
	hasRegex     bool
	hasSearchbox bool
}

func filtersCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	if err = submitFilterFormData(request, staff, logger.Info(), logger.Error()); err != nil {
		// submitFilterFormData logs any errors
		return nil, err
	}

	data, err := buildFilterFormData(request, logger.Error())
	if err != nil {
		// buildFilterPageData logs any errors
		return nil, err
	}

	showStr := request.FormValue("show")
	var show gcsql.BooleanFilter
	switch showStr {
	case "inactive":
		show = gcsql.OnlyFalse
	case "all":
		show = gcsql.TrueOrFalse
	default:
		show = gcsql.OnlyTrue
	}
	var filters []gcsql.Filter
	boardSearch := request.FormValue("boardsearch")
	if boardSearch == "" {
		filters, err = gcsql.GetAllFilters(show, true)
	} else {
		filters, err = gcsql.GetFiltersByBoardDir(boardSearch, false, show, true)
	}

	if err != nil {
		logger.Err(err).Caller().
			Str("boardSearch", boardSearch).
			Msg("Unable to get filter list")
		return nil, err
	}
	fieldsMap := make(map[string]string)
	for _, ff := range filterFields {
		fieldsMap[ff.Value] = ff.Text
	}
	staffUsernames := make([]string, len(filters))
	conditionsText := make([]string, len(filters))
	boardsText := make([]string, len(filters))
	filterHits := make([]int, len(filters))

	for f, filter := range filters {
		if _, ok := filterActionsMap[filter.MatchAction]; !ok {
			logger.Err(gcsql.ErrInvalidMatchAction).Caller().Str("filterAction", filter.MatchAction).Send()
			return nil, gcsql.ErrInvalidMatchAction
		}
		conditions, err := filter.Conditions()
		if err != nil {
			logger.Err(err).Caller().Int("filterID", filter.ID).Msg("Unable to get filter conditions")
			return nil, err
		}
		conditionsMap := make(map[string]int) // used to prevent printing duplicate conditions to make the filter list more readable
		for _, condition := range conditions {
			conditionsMap[condition.Field]++
		}
		var combined []string
		for conditionField, count := range conditionsMap {
			if count > 1 {
				combined = append(combined, fmt.Sprintf("%s (%d)", fieldsMap[conditionField], count))
			} else {
				combined = append(combined, fieldsMap[conditionField])
			}
		}

		conditionsText[f] = strings.Join(combined, ", ")

		boards, err := filter.BoardDirs()
		if err != nil {
			logger.Err(err).Caller().Int("filterID", filter.ID)
			return nil, err
		}
		boardsText = append(boardsText, strings.Join(boards, ","))
		if filter.StaffID == nil {
			staffUsernames[f] = "?"
		} else {
			username, err := gcsql.GetStaffUsernameFromID(*filter.StaffID)
			if err != nil {
				logger.Err(err).Caller().Int("filterID", filter.ID).Msg("Unable to get staff from filter")
				return nil, err
			}
			staffUsernames[f] = username
		}
		hits, err := filter.NumHits()
		if err != nil {
			logger.Err(err).Caller().Int("filterID", filter.ID).Send()
			return nil, fmt.Errorf("unable to get list of hits for filter %d", filter.ID)
		}
		filterHits[f] = hits
	}

	data["filters"] = filters
	data["filterHits"] = filterHits
	data["conditions"] = conditionsText
	data["filterTableBoards"] = boardsText
	data["staff"] = staffUsernames
	data["show"] = showStr
	data["boardSearch"] = boardSearch

	var buf bytes.Buffer
	if err = serverutil.MinifyTemplate(gctemplates.ManageFilters, data, &buf, "text/html"); err != nil {
		logger.Err(err).Caller().Str("template", gctemplates.ManageFilters).Send()
		return "", fmt.Errorf("failed to execute filter management template: %w", err)
	}
	return buf.String(), nil
}

func ipSearchCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	ipQuery := request.FormValue("ip")
	limitStr := request.FormValue("limit")
	data := map[string]any{
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
			logger.Err(err).Caller().
				Str("ipQuery", ipQuery).
				Int("limit", limit).
				Bool("onlyNotDeleted", true).
				Send()
			return "", fmt.Errorf("Error getting list of posts from %q by staff %s: %w", ipQuery, staff.Username, err)
		}
	}

	manageIpBuffer := bytes.NewBufferString("")
	if err = serverutil.MinifyTemplate(gctemplates.ManageIPSearch, data, manageIpBuffer, "text/html"); err != nil {
		logger.Err(err).Caller().
			Str("template", gctemplates.ManageIPSearch).Send()
		return "", errors.New("unable to render IP search page template")
	}
	return manageIpBuffer.String(), nil
}

func threadAttrsCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, wantsJSON bool, logger zerolog.Logger) (output any, err error) {
	boardDir := request.FormValue("board")
	attrBuffer := bytes.NewBufferString("")
	data := map[string]any{
		"boards": gcsql.AllBoards,
	}
	if boardDir == "" {
		if wantsJSON {
			return nil, errors.New(`missing required field "board"`)
		}
		if err = serverutil.MinifyTemplate(gctemplates.ManageThreadAttrs, data, attrBuffer, "text/html"); err != nil {
			logger.Err(err).Caller().Send()
			return "", err
		}
		return attrBuffer.String(), nil
	}
	logger = logger.With().Str("board", boardDir).Logger()
	board, err := gcsql.GetBoardFromDir(boardDir)
	if err != nil {
		logger.Err(err).Caller().Send()
		return "", err
	}
	data["board"] = board
	topPostStr := request.FormValue("thread")
	if topPostStr != "" {
		var topPostID int
		if topPostID, err = strconv.Atoi(topPostStr); err != nil {
			logger.Err(err).Str("topPostStr", topPostStr).Caller().Send()
			return "", err
		}
		logger = logger.With().Int("topPostID", topPostID).Logger()
		data["topPostID"] = topPostID
		var attr string
		var newVal bool
		var doChange bool // if false, don't bother executing any SQL since nothing will change
		thread, err := gcsql.GetPostThread(topPostID)
		if err != nil {
			logger.Err(err).Caller().Send()
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
		} else if request.FormValue("uncyclic") != "" {
			attr = "cyclic"
			newVal = false
			doChange = thread.Cyclic != newVal
		} else if request.FormValue("cyclic") != "" {
			attr = "cyclic"
			newVal = true
			doChange = thread.Cyclic != newVal
		}

		if attr != "" && doChange {
			logger = logger.With().
				Str("attribute", attr).
				Bool("newVal", newVal).
				Logger()
			if err = thread.UpdateAttribute(attr, newVal); err != nil {
				logger.Err(err).Caller().Send()
				return "", err
			}
			if err = building.BuildBoardPages(board, logger.Error()); err != nil {
				return "", err
			}
			post, err := gcsql.GetPostFromID(topPostID, true)
			if err != nil {
				logger.Err(err).Caller().Send()
				return "", err
			}
			if err = building.BuildThreadPages(post); err != nil {
				return "", err
			}
			gcutil.LogInfo().Msg("Done rebuilding")
		}
		data["thread"] = thread
	}

	threads, err := gcsql.GetThreadsWithBoardID(board.ID, true)
	if err != nil {
		logger.Err(err).Caller().Send()
		return "", err
	}
	data["threads"] = threads
	var threadIDs []any
	for i := len(threads) - 1; i >= 0; i-- {
		threadIDs = append(threadIDs, threads[i].ID)
	}
	if wantsJSON {
		return threads, nil
	}

	opMap, err := gcsql.GetTopPostIDsInThreadIDs(threadIDs...)
	if err != nil {
		logger.Err(err).Caller().Send()
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
		logger.Err(err).Caller().Send()
		return "", err
	}
	return attrBuffer.String(), nil
}

type postJSONWithIP struct {
	// gcsql.Post.IP's struct tag hides the IP field, but we want to see it here
	*gcsql.Post
	IP string
}

type postInfoJSON struct {
	Post *postJSONWithIP `json:"post"`
	FQDN []string        `json:"ipFQDN"`

	OriginalFilename string `json:"originalFilename,omitempty"`
	Checksum         string `json:"checksum,omitempty"`
	Fingerprint      string `json:"fingerprint,omitempty"`
}

func postInfoCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	postIDstr := request.FormValue("postid")
	if postIDstr == "" {
		return "", errors.New("invalid request (missing postid)")
	}
	var postID int
	if postID, err = strconv.Atoi(postIDstr); err != nil {
		logger.Err(err).Caller().
			Str("postID", postIDstr).Send()
		return "", err
	}
	post, err := gcsql.GetPostFromID(postID, true)
	if err != nil {
		logger.Err(err).Caller().
			Int("postID", postID).Send()
		return "", err
	}

	postInfo := postInfoJSON{
		Post: &postJSONWithIP{
			Post: post,
			IP:   post.IP,
		},
	}
	names, err := net.LookupAddr(post.IP)
	if err == nil {
		postInfo.FQDN = names
	} else {
		postInfo.FQDN = []string{err.Error()}
	}
	upload, err := post.GetUpload()
	if err != nil {
		logger.Err(err).Caller().Msg("Unable to get upload")
		return "", err
	}
	if upload != nil {
		postInfo.OriginalFilename = upload.OriginalFilename
		postInfo.Checksum = upload.Checksum
		if postInfo.OriginalFilename != "deleted" {
			postInfo.Fingerprint, err = uploads.GetPostImageFingerprint(postID)
			if err != nil {
				logger.Err(err).Caller().Msg("Unable to get image fingerprint")
				return "", err
			}
		}
	}
	return postInfo, nil
}

type fingerprintJSON struct {
	Fingerprint string `json:"fingerprint"`
}

func fingerprintCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	postIDstr := request.Form.Get("post")
	if postIDstr == "" {
		return "", errors.New("missing 'post' field")
	}
	postID, err := strconv.Atoi(postIDstr)
	if err != nil {
		logger.Err(err).Caller().Send()
		return "", err
	}
	fingerprint, err := uploads.GetPostImageFingerprint(postID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("post has no files or post doesn't exist")
	} else if err != nil {
		logger.Err(err).Caller().Send()
		return "", err
	}
	return fingerprintJSON{
		Fingerprint: fingerprint,
	}, nil
}

func wordfiltersCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	do := request.PostFormValue("dowordfilter")
	editIDstr := request.FormValue("edit")
	disableIDstr := request.FormValue("disable")
	enableIDstr := request.FormValue("enable")

	if disableIDstr != "" {
		disableID, err := strconv.Atoi(disableIDstr)
		if err != nil {
			logger.Err(err).Caller().Str("disableID", disableIDstr).Send()
			return nil, err
		}
		if err = gcsql.SetFilterActive(disableID, false); err != nil {
			logger.Err(err).Caller().Int("disableID", disableID).Msg("Unable to disable filter")
			return nil, errors.New("unable to disable wordfilter")
		}
		logger = logger.With().Int("disableID", disableID).Logger()
	} else if enableIDstr != "" {
		enableID, err := strconv.Atoi(enableIDstr)
		if err != nil {
			logger.Err(err).Caller().Str("enableID", enableIDstr).Send()
			return nil, err
		}
		if err = gcsql.SetFilterActive(enableID, true); err != nil {
			logger.Err(err).Caller().Int("enableID", enableID).Msg("Unable to enable filter")
			return nil, errors.New("unable to enable wordfilter")
		}
		logger = logger.With().Int("enableID", enableID).Logger()
	}

	var filter *gcsql.Wordfilter
	if editIDstr != "" {
		editID, err := strconv.Atoi(editIDstr)
		if err != nil {
			logger.Err(err).Str("editID", editIDstr).Send()
			return nil, err
		}
		logger = logger.With().Int("editID", editID).Logger()

		filter, err = gcsql.GetWordfilterByID(editID)
		if err != nil {
			logger.Err(err).Caller().Msg("Unable to get wordfilter")
			return nil, fmt.Errorf("unable to get wordfilter with id #%d", editID)
		}
	}
	searchFor := request.PostFormValue("searchfor")
	replaceWith := request.PostFormValue("replace")
	isRegex := request.PostFormValue("isregex") == "on"
	matchMode := gcsql.SubstrMatch
	if isRegex {
		matchMode = gcsql.RegexMatch
	}

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
		logger = logger.With().
			Array("boards", boardsLog).
			Str("searchFor", searchFor).
			Str("replaceWith", replaceWith).
			Str("staffNote", staffNote).
			Bool("isRegex", isRegex).
			Logger()
	}

	switch do {
	case "Edit wordfilter":
		if err = filter.UpdateDetails(staffNote, "replace", replaceWith, false); err != nil {
			logger.Err(err).Caller().Msg("Unable to update wordfilter details")
			return nil, errors.New("unable to update wordfilter details")
		}
		if err = filter.SetConditions(gcsql.FilterCondition{
			FilterID:  filter.ID,
			MatchMode: matchMode,
			Search:    searchFor,
			Field:     "body",
		}); err != nil {
			logger.Err(err).Caller().Msg("Unable to set filter condition")
			return nil, errors.New("unable to set filter conditions")
		}
		if err = filter.SetBoardDirs(boards...); err != nil {
			logger.Err(err).Caller().Msg("Unable to set board directories")
			return nil, errors.New("unable to set board directories")
		}
		logger = logger.With().Str("do", "update").Logger()
	case "Create wordfilter":
		if _, err = gcsql.CreateWordFilter(searchFor, replaceWith, isRegex, boards, staff.ID, staffNote); err != nil {
			logger.Err(err).Caller().Msg("Unable to create wordfilter")
			return nil, errors.New("unable to create wordfilter")
		}
		logger = logger.With().Str("do", "create").Logger()
	}

	wordfilters, err := gcsql.GetWordfilters(gcsql.TrueOrFalse)
	if err != nil {
		logger.Err(err).Caller().Msg("Unable to get wordfilters")
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
		logger.Err(err).Str("template", "manage_wordfilters.html").Caller().Send()
		return nil, err
	}
	if do != "" || enableIDstr != "" || disableIDstr != "" {
		logger.Info().Send()
	}
	return buf.String(), nil
}

func registerModeratorPages() {
	RegisterManagePage("bans", "Bans", ModPerms, NoJSON, bansCallback)
	RegisterManagePage("appeals", "Ban Appeals", ModPerms, OptionalJSON, appealsCallback)
	RegisterManagePage("filters", "Post Filters", ModPerms, NoJSON, filtersCallback)
	RegisterManagePageWithMethods("filters/hits/:filterID", "Filter Hits", ModPerms, NoJSON, true, filterHitsCallback, http.MethodGet, http.MethodPost)
	RegisterManagePage("ipsearch", "IP Search", ModPerms, NoJSON, ipSearchCallback)
	RegisterManagePage("reports", "Reports", ModPerms, OptionalJSON, reportsCallback)
	RegisterManagePage("threadattrs", "View/Update Thread Attributes", ModPerms, OptionalJSON, threadAttrsCallback)
	RegisterManagePage("postinfo", "Post Info", ModPerms, AlwaysJSON, postInfoCallback)
	RegisterManagePage("fingerprint", "Get Image/Thumbnail Fingerprint", ModPerms, AlwaysJSON, fingerprintCallback)
	RegisterManagePage("wordfilters", "Wordfilters", ModPerms, NoJSON, wordfiltersCallback)
}

package manage

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

var (
	ErrPasswordConfirm        = errors.New("passwords do not match")
	ErrInsufficientPermission = errors.New("insufficient account permission")
)

type uploadInfo struct {
	PostID      int
	OpID        int
	Filename    string
	Spoilered   bool
	Width       int
	Height      int
	ThumbWidth  int
	ThumbHeight int
}

// manage actions that require admin-level permission go here

func registerAdminPages() {
	actions = append(actions,
		Action{
			ID:          "cleanup",
			Title:       "Cleanup",
			Permissions: AdminPerms,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
				outputStr := ""
				if request.FormValue("run") == "Run Cleanup" {
					outputStr += "Removing deleted posts from the database.<hr />"
					if err = gcsql.PermanentlyRemoveDeletedPosts(); err != nil {
						errEv.Err(err).
							Str("cleanup", "removeDeletedPosts").
							Caller().Send()
						err = errors.New("Error removing deleted posts from database: " + err.Error())
						return outputStr + "<tr><td>" + err.Error() + "</td></tr></table>", err
					}

					outputStr += "Optimizing all tables in database.<hr />"
					err = gcsql.OptimizeDatabase()
					if err != nil {
						errEv.Err(err).
							Str("sql", "optimization").
							Caller().Send()
						err = errors.New("Error optimizing SQL tables: " + err.Error())
						return outputStr + "<tr><td>" + err.Error() + "</td></tr></table>", err
					}
					outputStr += "Cleanup finished"
				} else {
					outputStr += `<form action="` + config.GetSystemCriticalConfig().WebRoot + `manage/cleanup" method="post">` +
						`<input name="run" id="run" type="submit" value="Run Cleanup" />` +
						`</form>`
				}
				return outputStr, nil
			}},
		Action{
			ID:          "staff",
			Title:       "Staff",
			Permissions: JanitorPerms,
			JSONoutput:  OptionalJSON,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
				var outputStr string
				do := request.FormValue("do")
				allStaff, err := getAllStaffNopass(true)
				if wantsJSON {
					if err != nil {
						errEv.Err(err).Caller().Msg("Failed getting staff list")
					}
					return allStaff, err
				}
				if err != nil {
					errEv.Err(err).Caller().Msg("Failed getting staff list")
					err = errors.New("Error getting staff list: " + err.Error())
					return "", err
				}

				updateUsername := request.FormValue("update")
				username := request.FormValue("username")
				password := request.FormValue("password")
				passwordConfirm := request.FormValue("passwordconfirm")
				if (do == "add" || do == "update") && password != passwordConfirm {
					return "", ErrPasswordConfirm
				}

				rankStr := request.FormValue("rank")
				var rank int
				if rankStr != "" {
					if rank, err = strconv.Atoi(rankStr); err != nil {
						errEv.Err(err).Caller().
							Str("rank", rankStr).Send()
						return "", err
					}
				}

				if do == "add" {
					if staff.Rank < 3 {
						writer.WriteHeader(http.StatusUnauthorized)
						errEv.Err(ErrInsufficientPermission).Caller().
							Int("rank", staff.Rank).Send()
						return "", ErrInsufficientPermission
					}
					if _, err = gcsql.NewStaff(username, password, rank); err != nil {
						errEv.Caller().
							Str("newStaff", username).
							Str("newPass", password).
							Int("newRank", rank).
							Msg("Error creating new staff account")
						return "", fmt.Errorf("Error creating new staff account %q by %q: %s",
							username, staff.Username, err.Error())
					}
				} else if do == "del" && username != "" {
					if staff.Rank < 3 {
						writer.WriteHeader(http.StatusUnauthorized)
						errEv.Err(ErrInsufficientPermission).Caller().
							Int("rank", staff.Rank).Send()
						return "", ErrInsufficientPermission
					}
					if err = gcsql.DeactivateStaff(username); err != nil {
						errEv.Err(err).Caller().
							Str("delStaff", username).
							Msg("Error deleting staff account")
						return "", fmt.Errorf("Error deleting staff account %q by %q: %s",
							username, staff.Username, err.Error())
					}
				} else if do == "update" && updateUsername != "" {
					if staff.Username != updateUsername && staff.Rank < 3 {
						writer.WriteHeader(http.StatusUnauthorized)
						errEv.Err(ErrInsufficientPermission).Caller().
							Int("rank", staff.Rank).Send()
						return "", ErrInsufficientPermission
					}
					if err = gcsql.UpdatePassword(updateUsername, password); err != nil {
						errEv.Err(err).Caller().
							Str("updateStaff", username).
							Msg("Error updating password")
						return "", err
					}
				}
				if do == "add" || do == "del" {
					allStaff, err = getAllStaffNopass(true)
					if err != nil {
						errEv.Err(err).Caller().Msg("Error getting updated staff list")
						err = errors.New("Error getting updated staff list: " + err.Error())
						return "", err
					}
				}

				staffBuffer := bytes.NewBufferString("")
				if err = serverutil.MinifyTemplate(gctemplates.ManageStaff, map[string]interface{}{
					"do":             do,
					"updateUsername": updateUsername,
					"allstaff":       allStaff,
					"currentStaff":   staff,
				}, staffBuffer, "text/html"); err != nil {
					errEv.Err(err).Str("template", "manage_staff.html").Send()
					return "", errors.New("Error executing staff management page template: " + err.Error())
				}
				outputStr += staffBuffer.String()
				return outputStr, nil
			}},
		Action{
			ID:          "boards",
			Title:       "Boards",
			Permissions: AdminPerms,
			JSONoutput:  NoJSON,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
				board := &gcsql.Board{
					MaxFilesize:      1000 * 1000 * 15,
					AnonymousName:    "Anonymous",
					EnableCatalog:    true,
					MaxMessageLength: 1500,
					AutosageAfter:    200,
					NoImagesAfter:    0,
				}
				requestType, _, _ := boardsRequestType(request)
				switch requestType {
				case "create":
					// create button clicked, create the board with the request fields
					if err = getBoardDataFromForm(board, request); err != nil {
						errEv.Err(err).Caller().Send()
						return "", err
					}
					if err = gcsql.CreateBoard(board, true); err != nil {
						errEv.Err(err).Caller().Send()
						return "", err
					}
					infoEv.
						Str("createBoard", board.Dir).
						Int("boardID", board.ID).
						Msg("New board created")
				case "delete":
					// delete button clicked, delete the board
					boardID, err := getIntField("board", staff.Username, request, 0)
					if err != nil {
						return "", err
					}
					// use a temporary variable so that the form values aren't filled
					var deleteBoard *gcsql.Board
					if deleteBoard, err = gcsql.GetBoardFromID(boardID); err != nil {
						errEv.Err(err).Int("deleteBoardID", boardID).Caller().Send()
						return "", err
					}
					if err = deleteBoard.Delete(); err != nil {
						errEv.Err(err).Str("deleteBoard", deleteBoard.Dir).Caller().Send()
						return "", err
					}
					infoEv.
						Str("deleteBoard", deleteBoard.Dir).Send()
					if err = os.RemoveAll(deleteBoard.AbsolutePath()); err != nil {
						errEv.Err(err).Caller().Send()
						return "", err
					}
				case "edit":
					// edit button clicked, fill the input fields with board data to be edited
					boardID, err := getIntField("board", staff.Username, request, 0)
					if err != nil {
						return "", err
					}
					if board, err = gcsql.GetBoardFromID(boardID); err != nil {
						errEv.Err(err).
							Int("boardID", boardID).
							Caller().Msg("Unable to get board info")
						return "", err
					}
				case "modify":
					// save changes button clicked, apply changes to the board based on the request fields
					if err = getBoardDataFromForm(board, request); err != nil {
						return "", err
					}
					if err = board.ModifyInDB(); err != nil {
						return "", errors.New("Unable to apply changes: " + err.Error())
					}
				case "cancel":
					// cancel button was clicked
					fallthrough
				case "":
					fallthrough
				default:
					// board.SetDefaults("", "", "")
				}

				if requestType == "create" || requestType == "modify" || requestType == "delete" {
					if err = gcsql.ResetBoardSectionArrays(); err != nil {
						errEv.Err(err).Caller().Send()
						return "", errors.New("unable to reset board list: " + err.Error())
					}
					if err = building.BuildBoardListJSON(); err != nil {
						return "", err
					}
					if err = building.BuildBoards(false); err != nil {
						return "", err
					}
				}
				pageBuffer := bytes.NewBufferString("")
				if err = serverutil.MinifyTemplate(gctemplates.ManageBoards,
					map[string]interface{}{
						"siteConfig":  config.GetSiteConfig(),
						"sections":    gcsql.AllSections,
						"boards":      gcsql.AllBoards,
						"boardConfig": config.GetBoardConfig(""),
						"editing":     requestType == "edit",
						"board":       board,
					}, pageBuffer, "text/html"); err != nil {
					errEv.Err(err).Str("template", "manage_boards.html").Caller().Send()
					return "", err
				}

				return pageBuffer.String(), nil
			}},
		Action{
			ID:          "boardsections",
			Title:       "Board sections",
			Permissions: AdminPerms,
			JSONoutput:  OptionalJSON,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
				section := &gcsql.Section{}
				editID := request.Form.Get("edit")
				updateID := request.Form.Get("updatesection")
				deleteID := request.Form.Get("delete")
				if editID != "" {
					if section, err = gcsql.GetSectionFromID(gcutil.HackyStringToInt(editID)); err != nil {
						errEv.Err(err).Caller().Send()
						return "", &ErrStaffAction{
							ErrorField: "db",
							Action:     "boardsections",
							Message:    err.Error(),
						}
					}
				} else if updateID != "" {
					if section, err = gcsql.GetSectionFromID(gcutil.HackyStringToInt(updateID)); err != nil {
						errEv.Err(err).Caller().Send()
						return "", &ErrStaffAction{
							ErrorField: "db",
							Action:     "boardsections",
							Message:    err.Error(),
						}
					}
				} else if deleteID != "" {
					if err = gcsql.DeleteSection(gcutil.HackyStringToInt(deleteID)); err != nil {
						errEv.Err(err).Caller().Send()
						return "", &ErrStaffAction{
							ErrorField: "db",
							Action:     "boardsections",
							Message:    err.Error(),
						}
					}
				}

				if request.PostForm.Get("save_section") != "" {
					// user is creating a new board section
					if section == nil {
						section = &gcsql.Section{}
					}
					section.Name = request.PostForm.Get("sectionname")
					section.Abbreviation = request.PostForm.Get("sectionabbr")
					section.Hidden = request.PostForm.Get("sectionhidden") == "on"
					section.Position, err = strconv.Atoi(request.PostForm.Get("sectionpos"))
					if section.Name == "" || section.Abbreviation == "" || request.PostForm.Get("sectionpos") == "" {
						return "", &ErrStaffAction{
							ErrorField: "formerror",
							Action:     "boardsections",
							Message:    "Missing section title, abbreviation, or hidden status data",
						}
					} else if err != nil {
						errEv.Err(err).Caller().Send()
						return "", &ErrStaffAction{
							ErrorField: "formerror",
							Action:     "boardsections",
							Message:    err.Error(),
						}
					}
					if updateID != "" {
						// submitting changes to the section
						err = section.UpdateValues()
					} else {
						// creating a new section
						section, err = gcsql.NewSection(section.Name, section.Abbreviation, section.Hidden, section.Position)
					}
					if err != nil {
						errEv.Err(err).Caller().Send()
						return "", &ErrStaffAction{
							ErrorField: "db",
							Action:     "boardsections",
							Message:    err.Error(),
						}
					}
					gcsql.ResetBoardSectionArrays()
				}

				sections, err := gcsql.GetAllSections(false)
				if err != nil {
					errEv.Err(err).Caller().Send()
					return "", err
				}
				pageBuffer := bytes.NewBufferString("")
				pageMap := map[string]interface{}{
					"siteConfig": config.GetSiteConfig(),
					"sections":   sections,
				}
				if section.ID > 0 {
					pageMap["edit_section"] = section
				}
				if err = serverutil.MinifyTemplate(gctemplates.ManageSections, pageMap, pageBuffer, "text/html"); err != nil {
					errEv.Err(err).Caller().Send()
					return "", err
				}
				output = pageBuffer.String()
				return
			}},
		Action{
			ID:          "fixthumbnails",
			Title:       "Regenerate thumbnails",
			Permissions: AdminPerms,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv, errEv *zerolog.Event) (output interface{}, err error) {
				board := request.FormValue("board")
				var uploads []uploadInfo
				if board != "" {
					const query = `SELECT p1.id as id, (SELECT id FROM DBPREFIXposts p2 WHERE p2.is_top_post AND p1.thread_id = p2.thread_id LIMIT 1) AS op,
					filename, is_spoilered, width, height, thumbnail_width, thumbnail_height
					FROM DBPREFIXposts p1
					JOIN DBPREFIXthreads t ON t.id = p1.thread_id
					JOIN DBPREFIXboards b ON b.id = t.board_id
					LEFT JOIN DBPREFIXfiles f ON f.post_id = p1.id
					WHERE dir = ? AND p1.is_deleted = FALSE AND filename IS NOT NULL AND filename != 'deleted'
					ORDER BY created_on DESC`
					rows, err := gcsql.QuerySQL(query, board)
					if err != nil {
						return "", err
					}
					defer rows.Close()
					for rows.Next() {
						var info uploadInfo
						if err = rows.Scan(
							&info.PostID, &info.OpID, &info.Filename, &info.Spoilered, &info.Width, &info.Height,
							&info.ThumbWidth, &info.ThumbHeight,
						); err != nil {
							errEv.Err(err).Caller().Send()
							return "", err
						}
						uploads = append(uploads, info)
					}
				}
				buffer := bytes.NewBufferString("")
				err = serverutil.MinifyTemplate(gctemplates.ManageFixThumbnails, map[string]any{
					"allBoards": gcsql.AllBoards,
					"board":     board,
					"uploads":   uploads,
				}, buffer, "text/html")
				if err != nil {
					errEv.Err(err).Str("template", "manage_fixthumbnails.html").Caller().Send()
					return "", err
				}
				return buffer.String(), nil
			},
		},
		Action{
			ID:          "rebuildfront",
			Title:       "Rebuild front page",
			Permissions: AdminPerms,
			JSONoutput:  OptionalJSON,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
				if err = gctemplates.InitTemplates(); err != nil {
					return "", err
				}
				err = building.BuildFrontPage()
				if wantsJSON {
					return map[string]string{
						"front": "Built front page successfully",
					}, err
				}
				return "Built front page successfully", err
			}},
		Action{
			ID:          "rebuildall",
			Title:       "Rebuild everything",
			Permissions: AdminPerms,
			JSONoutput:  OptionalJSON,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
				gctemplates.InitTemplates()
				if err = gcsql.ResetBoardSectionArrays(); err != nil {
					errEv.Err(err).Caller().Send()
					return "", err
				}
				buildErr := &ErrStaffAction{
					ErrorField: "builderror",
					Action:     "rebuildall",
				}
				buildMap := map[string]string{}
				if err = building.BuildFrontPage(); err != nil {
					buildErr.Message = "Error building front page: " + err.Error()
					if wantsJSON {
						return buildErr, buildErr
					}
					return buildErr.Message, buildErr
				}
				buildMap["front"] = "Built front page successfully"

				if err = building.BuildBoardListJSON(); err != nil {
					buildErr.Message = "Error building board list: " + err.Error()
					if wantsJSON {
						return buildErr, buildErr
					}
					return buildErr.Message, buildErr
				}
				buildMap["boardlist"] = "Built board list successfully"

				if err = building.BuildBoards(false); err != nil {
					buildErr.Message = "Error building boards: " + err.Error()
					if wantsJSON {
						return buildErr, buildErr
					}
					return buildErr.Message, buildErr
				}
				buildMap["boards"] = "Built boards successfully"

				if err = building.BuildJS(); err != nil {
					buildErr.Message = "Error building consts.js: " + err.Error()
					if wantsJSON {
						return buildErr, buildErr
					}
					return buildErr.Message, buildErr
				}
				if wantsJSON {
					return buildMap, nil
				}
				buildStr := ""
				for _, msg := range buildMap {
					buildStr += fmt.Sprintln(msg, "<hr />")
				}
				return buildStr, nil
			}},
		Action{
			ID:          "rebuildboards",
			Title:       "Rebuild boards",
			Permissions: AdminPerms,
			JSONoutput:  OptionalJSON,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
				if err = gctemplates.InitTemplates(); err != nil {
					errEv.Err(err).Caller().Msg("Unable to initialize templates")
					return "", err
				}
				err = building.BuildBoards(false)
				if err != nil {
					errEv.Err(err).Caller().Msg("Unable to build boards")
					return "", err
				}
				if wantsJSON {
					return map[string]interface{}{
						"success": true,
						"message": "Boards built successfully",
					}, nil
				}
				return "Boards built successfully", nil
			}},
		Action{
			ID:          "reparsehtml",
			Title:       "Reparse HTML",
			Permissions: AdminPerms,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
				var outputStr string
				tx, err := gcsql.BeginTx()
				if err != nil {
					errEv.Err(err).Msg("Unable to begin transaction")
					return "", errors.New("unable to begin SQL transaction")
				}
				defer tx.Rollback()
				const query = `SELECT
					id, message_raw, thread_id as threadid,
					(SELECT id FROM DBPREFIXposts WHERE is_top_post = TRUE AND thread_id = threadid LIMIT 1) AS op,
					(SELECT board_id FROM DBPREFIXthreads WHERE id = threadid) AS boardid,
					(SELECT dir FROM DBPREFIXboards WHERE id = boardid) AS dir
					FROM DBPREFIXposts WHERE is_deleted = FALSE`
				const updateQuery = `UPDATE DBPREFIXposts SET message = ? WHERE id = ?`

				stmt, err := gcsql.PrepareSQL(query, tx)
				if err != nil {
					errEv.Err(err).Caller().Msg("Unable to prepare SQL query")
					return "", err
				}
				defer stmt.Close()
				rows, err := stmt.Query()
				if err != nil {
					errEv.Err(err).Msg("Unable to query the database")
					return "", err
				}
				defer rows.Close()
				for rows.Next() {
					var postID, threadID, opID, boardID int
					var messageRaw, boardDir string
					if err = rows.Scan(&postID, &messageRaw, &threadID, &opID, &boardID, &boardDir); err != nil {
						errEv.Err(err).Caller().Msg("Unable to scan SQL row")
						return "", err
					}
					formatted := posting.FormatMessage(messageRaw, boardDir)
					gcsql.ExecSQL(updateQuery, formatted, postID)
				}
				outputStr += "Done reparsing HTML<hr />"

				if err = building.BuildFrontPage(); err != nil {
					return "", err
				}
				outputStr += "Done building front page<hr />"

				if err = building.BuildBoardListJSON(); err != nil {
					return "", err
				}
				outputStr += "Done building board list JSON<hr />"

				if err = building.BuildBoards(false); err != nil {
					return "", err
				}
				outputStr += "Done building boards<hr />"
				return outputStr, nil
			}},
		Action{
			ID:          "wordfilters",
			Title:       "Wordfilters",
			Permissions: AdminPerms,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
				managePageBuffer := bytes.NewBufferString("")
				editIDstr := request.FormValue("edit")
				deleteIDstr := request.FormValue("delete")
				if deleteIDstr != "" {
					var result sql.Result
					if result, err = gcsql.ExecSQL(`DELETE FROM DBPREFIXwordfilters WHERE id = ?`, deleteIDstr); err != nil {
						return err, err
					}
					if numRows, _ := result.RowsAffected(); numRows < 1 {
						err = invalidWordfilterID(deleteIDstr)
						errEv.Err(err).Caller().Send()
						return err, err
					}
					infoEv.Str("deletedWordfilterID", deleteIDstr)
				}

				submitBtn := request.FormValue("dowordfilter")
				switch submitBtn {
				case "Edit wordfilter":
					regexCheckStr := request.FormValue("isregex")
					if regexCheckStr == "on" {
						regexCheckStr = "1"
					} else {
						regexCheckStr = "0"
					}
					_, err = gcsql.ExecSQL(`UPDATE DBPREFIXwordfilters
							SET board_dirs = ?,
							staff_note = ?,
							search = ?,
							is_regex = ?,
							change_to = ?
							WHERE id = ?`,
						request.FormValue("boarddirs"),
						request.FormValue("staffnote"),
						request.FormValue("find"),
						regexCheckStr,
						request.FormValue("replace"),
						editIDstr)
					infoEv.Str("do", "update")
				case "Create new wordfilter":
					_, err = gcsql.CreateWordFilter(
						request.FormValue("find"),
						request.FormValue("replace"),
						request.FormValue("isregex") == "on",
						request.FormValue("boarddirs"),
						staff.ID,
						request.FormValue("staffnote"))
					infoEv.Str("do", "create")
				case "":
					infoEv.Discard()
				}
				if err == nil {
					infoEv.
						Str("find", request.FormValue("find")).
						Str("replace", request.FormValue("replace")).
						Str("staffnote", request.FormValue("staffnote")).
						Str("boarddirs", request.FormValue("boarddirs"))
				} else {
					return err, err
				}

				wordfilters, err := gcsql.GetWordfilters()
				if err != nil {
					errEv.Err(err).Caller().Msg("Unable to get wordfilters")
					return wordfilters, err
				}
				var editFilter *gcsql.Wordfilter
				if editIDstr != "" {
					editID := gcutil.HackyStringToInt(editIDstr)
					for _, filter := range wordfilters {
						if filter.ID == editID {
							editFilter = &filter
							break
						}
					}
				}
				filterMap := map[string]interface{}{
					"wordfilters": wordfilters,
					"edit":        editFilter,
				}

				err = serverutil.MinifyTemplate(gctemplates.ManageWordfilters,
					filterMap, managePageBuffer, "text/html")
				if err != nil {
					errEv.Err(err).Str("template", "manage_wordfilters.html").Caller().Send()

				}
				infoEv.Send()
				return managePageBuffer.String(), err
			},
		},
		Action{
			ID:          "viewlog",
			Title:       "View log",
			Permissions: AdminPerms,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event,
				errEv *zerolog.Event) (output interface{}, err error) {
				logPath := path.Join(config.GetSystemCriticalConfig().LogDir, "gochan.log")
				logFile, err := os.Open(logPath)
				if err != nil {
					errEv.Err(err).Caller().Send()
					return "", errors.New("unable to open log file")
				}
				defer logFile.Close()
				ba, err := io.ReadAll(logFile)
				if err != nil {
					errEv.Err(err).Caller().Send()
					return "", err
				}
				buf := bytes.NewBufferString("")
				err = serverutil.MinifyTemplate(gctemplates.ManageViewLog, map[string]interface{}{
					"logText": string(ba),
				}, buf, "text/html")
				return buf.String(), err
			},
		},
	)
}

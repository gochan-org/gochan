package manage

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

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

func updateAnnouncementsCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, _ *zerolog.Event, errEv *zerolog.Event) (interface{}, error) {
	announcements, err := getAllAnnouncements()
	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to get staff announcements")
		return "", err
	}
	data := map[string]any{}
	editIdStr := request.FormValue("edit")
	var editID int
	deleteIdStr := request.FormValue("delete")
	var deleteID int
	var announcement announcementWithName
	if editIdStr != "" {
		if editID, err = strconv.Atoi(editIdStr); err != nil {
			errEv.Err(err).Str("editID", editIdStr).Send()
			return "", err
		}
		data["editID"] = editID
		for _, ann := range announcements {
			if ann.ID == uint(editID) {
				announcement = ann
				break
			}
		}
		if announcement.ID < 1 {
			return "", fmt.Errorf("no announcement found with id %d", editID)
		}
		if request.PostFormValue("doedit") == "Submit" {
			// announcement update submitted
			announcement.Subject = request.PostFormValue("subject")
			announcement.Message = request.PostFormValue("message")
			if announcement.Message == "" {
				errEv.Err(errMissingAnnouncementMessage).Caller().Send()
				return "", errMissingAnnouncementMessage
			}
			updateSQL := `UPDATE DBPREFIXannouncements SET subject = ?, message = ?, timestamp = CURRENT_TIMESTAMP WHERE id = ?`
			if _, err = gcsql.ExecSQL(updateSQL,
				announcement.Subject,
				announcement.Message,
				announcement.ID); err != nil {
				errEv.Err(err).Caller().
					Str("subject", announcement.Subject).
					Str("message", announcement.Message).
					Uint("id", announcement.ID).
					Msg("Unable to update announcement")
				return "", errors.New("unable to update announcement")
			}
			fmt.Printf("Updated announcement #%d, message = %s\n", announcement.ID, announcement.Message)
		}
	} else if deleteIdStr != "" {
		if deleteID, err = strconv.Atoi(deleteIdStr); err != nil {
			errEv.Err(err).Str("deleteID", deleteIdStr).Send()
			return "", err
		}
		deleteSQL := `DELETE FROM DBPREFIXannouncements WHERE id = ?`
		if _, err = gcsql.ExecSQL(deleteSQL, deleteID); err != nil {
			errEv.Err(err).Caller().
				Int("deleteID", deleteID).
				Msg("Unable to delete announcement")
			return "", errors.New("unable to delete announcement")
		}
	} else if request.PostFormValue("newannouncement") == "Submit" {
		insertSQL := `INSERT INTO DBPREFIXannouncements (staff_id, subject, message) VALUES(?, ?, ?)`
		announcement.Subject = request.PostFormValue("subject")
		announcement.Message = request.PostFormValue("message")
		if _, err = gcsql.ExecSQL(insertSQL, staff.ID, announcement.Subject, announcement.Message); err != nil {
			errEv.Err(err).Caller().
				Str("subject", announcement.Subject).
				Str("message", announcement.Message).
				Msg("Unable to submit new announcement")
			return "", errors.New("unable to submit announcement")
		}
	}
	// update announcements array in data so the creation/edit/deletion shows up immediately
	if data["announcements"], err = getAllAnnouncements(); err != nil {
		errEv.Err(err).Caller().Msg("Unable to get staff announcements")
		return "", err
	}
	data["announcement"] = announcement
	pageBuffer := bytes.NewBufferString("")
	err = serverutil.MinifyTemplate(gctemplates.ManageAnnouncements, data,
		pageBuffer, "tex/thtml")
	return pageBuffer.String(), err
}

func boardsCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
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
			errEv.Err(err).Caller().
				Int("boardID", boardID).
				Msg("Unable to get board info")
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
}

func boardSectionsCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, _ *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
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
}

func cleanupCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, _ *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
	outputStr := ""
	if request.FormValue("run") == "Run Cleanup" {
		outputStr += "Removing deleted posts from the database.<hr />"
		if err = gcsql.PermanentlyRemoveDeletedPosts(); err != nil {
			errEv.Err(err).Caller().
				Str("cleanup", "removeDeletedPosts").Send()
			err = errors.New("unable to remove deleted posts from database")
			return outputStr + "<tr><td>" + err.Error() + "</td></tr></table>", err
		}

		outputStr += "Optimizing all tables in database.<hr />"
		err = gcsql.OptimizeDatabase()
		if err != nil {
			errEv.Err(err).Caller().
				Str("sql", "optimization").Send()
			err = errors.New("Error optimizing SQL tables: " + err.Error())
			return outputStr + "<tr><td>" + err.Error() + "</td></tr></table>", err
		}
		outputStr += "Cleanup finished"
	} else {

		outputStr += `<form action="` + config.WebPath("manage/cleanup") + `" method="post">` +
			`<input name="run" id="run" type="submit" value="Run Cleanup" />` +
			`</form>`
	}
	return outputStr, nil
}

func fixThumbnailsCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, _, errEv *zerolog.Event) (output interface{}, err error) {
	board := request.FormValue("board")
	var uploads []uploadInfo
	if board != "" {
		const query = "SELECT * FROM DBPREFIXv_upload_info WHERE dir = ? ORDER BY created_on DESC"
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
}

func templatesCallback(writer http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, infoEv, errEv *zerolog.Event) (output interface{}, err error) {
	buf := bytes.NewBufferString("")

	selectedTemplate := request.FormValue("override")
	templatesDir := config.GetSystemCriticalConfig().TemplateDir
	var overriding string
	var templateStr string
	var templatePath string
	var successStr string
	if selectedTemplate != "" {
		gcutil.LogStr("selectedTemplate", selectedTemplate, infoEv, errEv)

		if templatePath, err = gctemplates.GetTemplatePath(selectedTemplate); err != nil {
			errEv.Err(err).Caller().Msg("unable to load selected template")
			return "", fmt.Errorf("template %q does not exist", selectedTemplate)
		}
		errEv.Str("templatePath", templatePath)
		ba, err := os.ReadFile(templatePath)
		if err != nil {
			errEv.Err(err).Caller().Send()
			return "", fmt.Errorf("unable to load selected template %q", selectedTemplate)
		}
		templateStr = string(ba)
	} else if overriding = request.PostFormValue("overriding"); overriding != "" {
		if templateStr = request.PostFormValue("templatetext"); templateStr == "" {
			writer.WriteHeader(http.StatusBadRequest)
			errEv.Caller().Int("status", http.StatusBadRequest).
				Msg("received an empty template string")
			return "", errors.New("received an empty template string")
		}
		if _, err = gctemplates.ParseTemplate(selectedTemplate, templateStr); err != nil {
			// unable to parse submitted template
			errEv.Err(err).Caller().Int("status", http.StatusBadRequest).Send()
			writer.WriteHeader(http.StatusBadRequest)
			return "", err
		}
		overrideDir := path.Join(templatesDir, "override")
		overridePath := path.Join(overrideDir, overriding)
		gcutil.LogStr("overridePath", overridePath, infoEv, errEv)

		if _, err = os.Stat(overrideDir); os.IsNotExist(err) {
			// override dir doesn't exist, create it
			if err = os.Mkdir(overrideDir, config.DirFileMode); err != nil {
				errEv.Err(err).Caller().
					Int("status", http.StatusInternalServerError).
					Msg("Unable to create override directory")
				writer.WriteHeader(http.StatusInternalServerError)
				return "", err
			}
		} else if err != nil {
			// got an error checking for override dir
			errEv.Err(err).Caller().
				Int("status", http.StatusInternalServerError).
				Msg("Unable to check if override directory exists")
			writer.WriteHeader(http.StatusInternalServerError)
			return "", err
		}

		// get the original template file, or the latest override if there are any
		templatePath, err := gctemplates.GetTemplatePath(overriding)
		if err != nil {
			errEv.Err(err).Caller().
				Int("status", http.StatusInternalServerError).
				Msg("Unable to get original template path")
			writer.WriteHeader(http.StatusInternalServerError)
			return "", err
		}

		// read original template path into []byte to be backed up
		ba, err := os.ReadFile(templatePath)
		if err != nil {
			errEv.Err(err).Caller().
				Int("status", http.StatusInternalServerError).
				Msg("Unable to read original template file")
			writer.WriteHeader(http.StatusInternalServerError)
			return "", err
		}

		// back up template to override/<overriding>-<timestamp>.bkp
		backupPath := path.Join(overrideDir, overriding) + time.Now().Format("-2006-01-02_15-04-05.bkp")
		gcutil.LogStr("backupPath", backupPath, infoEv, errEv)
		if err = os.WriteFile(backupPath, ba, config.NormalFileMode); err != nil {
			errEv.Err(err).Caller().
				Int("status", http.StatusInternalServerError).
				Msg("Unable to back up template file")
			writer.WriteHeader(http.StatusInternalServerError)
			return "", errors.New("unable to back up original template file")
		}

		// write changes to disk
		if err = os.WriteFile(overridePath, []byte(templateStr), config.NormalFileMode); err != nil {
			errEv.Err(err).Caller().
				Int("status", http.StatusInternalServerError).
				Msg("Unable to save changes")
			writer.WriteHeader(http.StatusInternalServerError)
			return "", err
		}

		// reload template
		if err = gctemplates.InitTemplates(overriding); err != nil {
			errEv.Err(err).Caller().
				Int("status", http.StatusInternalServerError).
				Msg("Unable to reinitialize template")
			writer.WriteHeader(http.StatusInternalServerError)
			return "", err
		}
		successStr = fmt.Sprintf("%q saved successfully.\n Original backed up to %s",
			overriding, backupPath)
		infoEv.Msg("Template successfully saved and reloaded")
	}

	data := map[string]any{
		"templates":        gctemplates.GetTemplateList(),
		"templatesDir":     templatesDir,
		"templatePath":     templatePath,
		"selectedTemplate": selectedTemplate,
		"success":          successStr,
	}
	if templateStr != "" && successStr == "" {
		data["templateText"] = templateStr
	}
	serverutil.MinifyTemplate(gctemplates.ManageTemplates, data, buf, "text/html")
	return buf.String(), nil
}

func rebuildFrontCallback(_ http.ResponseWriter, _ *http.Request, _ *gcsql.Staff, wantsJSON bool, _ *zerolog.Event, _ *zerolog.Event) (output interface{}, err error) {
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
}

func rebuildAllCallback(_ http.ResponseWriter, _ *http.Request, _ *gcsql.Staff, wantsJSON bool, _ *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
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
}

func rebuildBoardsCallback(_ http.ResponseWriter, _ *http.Request, _ *gcsql.Staff, wantsJSON bool, _ *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
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
}

func reparseHTMLCallback(_ http.ResponseWriter, _ *http.Request, _ *gcsql.Staff, _ bool, _ *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
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
		if formatted, err := posting.FormatMessage(messageRaw, boardDir); err != nil {
			errEv.Err(err).Caller().Msg("Unable to format message")
			return "", err
		} else {
			gcsql.ExecSQL(updateQuery, formatted, postID)
		}
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
}

func viewLogCallback(_ http.ResponseWriter, _ *http.Request, _ *gcsql.Staff, _ bool, _ *zerolog.Event,
	errEv *zerolog.Event) (output interface{}, err error) {
	logPath := path.Join(config.GetSystemCriticalConfig().LogDir, "gochan.log")
	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return "", errors.New("unable to open log file")
	}
	buf := bytes.NewBufferString("")
	err = serverutil.MinifyTemplate(gctemplates.ManageViewLog, map[string]interface{}{
		"logText": string(logBytes),
	}, buf, "text/html")
	return buf.String(), err
}

func registerAdminPages() {
	RegisterManagePage("updateannouncements", "Update staff announcements", AdminPerms, NoJSON, updateAnnouncementsCallback)
	RegisterManagePage("boards", "Boards", AdminPerms, NoJSON, boardsCallback)
	RegisterManagePage("boardsections", "Board sections", AdminPerms, OptionalJSON, boardSectionsCallback)
	RegisterManagePage("cleanup", "Cleanup", AdminPerms, NoJSON, cleanupCallback)
	RegisterManagePage("fixthumbnails", "Regenerate thumbnails", AdminPerms, NoJSON, fixThumbnailsCallback)
	RegisterManagePage("templates", "Override templates", AdminPerms, NoJSON, templatesCallback)
	RegisterManagePage("rebuildfront", "Rebuild front page", AdminPerms, OptionalJSON, rebuildFrontCallback)
	RegisterManagePage("rebuildboards", "Rebuild boards", AdminPerms, OptionalJSON, rebuildBoardsCallback)
	RegisterManagePage("rebuildall", "Rebuild everything", AdminPerms, OptionalJSON, rebuildAllCallback)
	RegisterManagePage("reparsehtml", "Reparse HTML", AdminPerms, NoJSON, reparseHTMLCallback)
	RegisterManagePage("viewlog", "View log", AdminPerms, NoJSON, viewLogCallback)
}

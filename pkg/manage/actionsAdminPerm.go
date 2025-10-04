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

	"github.com/Eggbertx/go-forms"
	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"
)

var (
	ErrInsufficientPermission = server.NewServerError("insufficient account permission", http.StatusForbidden)
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

// updateAnnouncementsCallback handles requests to /manage/updateannouncements for creating, editing, and deleting staff announcements
func updateAnnouncementsCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, logger zerolog.Logger) (any, error) {
	announcements, err := getAllAnnouncements()
	if err != nil {
		logger.Err(err).Caller().Msg("Unable to get staff announcements")
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
			logger.Err(err).Str("editID", editIdStr).Send()
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
				logger.Err(errMissingAnnouncementMessage).Caller().Send()
				return "", errMissingAnnouncementMessage
			}
			updateSQL := `UPDATE DBPREFIXannouncements SET subject = ?, message = ?, timestamp = CURRENT_TIMESTAMP WHERE id = ?`
			if _, err = gcsql.ExecSQL(updateSQL,
				announcement.Subject,
				announcement.Message,
				announcement.ID); err != nil {
				logger.Err(err).Caller().
					Str("subject", announcement.Subject).
					Str("message", announcement.Message).
					Uint("id", announcement.ID).
					Msg("Unable to update announcement")
				return "", errors.New("unable to update announcement")
			}
			gcutil.LogInfo().
				Int("announcementID", int(announcement.ID)).
				Msg("Updated announcement")
		}
	} else if deleteIdStr != "" {
		if deleteID, err = strconv.Atoi(deleteIdStr); err != nil {
			logger.Err(err).Str("deleteID", deleteIdStr).Send()
			return "", err
		}
		deleteSQL := `DELETE FROM DBPREFIXannouncements WHERE id = ?`
		if _, err = gcsql.ExecSQL(deleteSQL, deleteID); err != nil {
			logger.Err(err).Caller().
				Int("deleteID", deleteID).
				Msg("Unable to delete announcement")
			return "", errors.New("unable to delete announcement")
		}
	} else if request.PostFormValue("newannouncement") == "Submit" {
		insertSQL := `INSERT INTO DBPREFIXannouncements (staff_id, subject, message) VALUES(?, ?, ?)`
		announcement.Subject = request.PostFormValue("subject")
		announcement.Message = request.PostFormValue("message")
		if _, err = gcsql.ExecSQL(insertSQL, staff.ID, announcement.Subject, announcement.Message); err != nil {
			logger.Err(err).Caller().
				Str("subject", announcement.Subject).
				Str("message", announcement.Message).
				Msg("Unable to submit new announcement")
			return "", errors.New("unable to submit announcement")
		}
	}
	// update announcements array in data so the creation/edit/deletion shows up immediately
	if data["announcements"], err = getAllAnnouncements(); err != nil {
		logger.Err(err).Caller().Msg("Unable to get staff announcements")
		return "", err
	}
	data["announcement"] = announcement
	pageBuffer := bytes.NewBufferString("")
	err = serverutil.MinifyTemplate(gctemplates.ManageAnnouncements, data,
		pageBuffer, "tex/thtml")
	return pageBuffer.String(), err
}

// boardsCallback handles calls to /manage/boards, showing boards by section and a form to create a new board
func boardsCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	requestType := boardsRequestType(request)

	logger = logger.With().Str("requestType", requestType.String()).Logger()

	if requestType == boardRequestTypeCreate {
		var board gcsql.Board
		var form createOrModifyBoardForm
		if err = forms.FillStructFromForm(request, &form); err != nil {
			logger.Warn().Err(err).Caller().Msg("Error parsing board form")
			return nil, server.NewServerError(err, http.StatusBadRequest)
		}
		if err = form.validate(logger.Warn()); err != nil {
			return nil, err
		}
		form.fillBoard(&board)
		if err = gcsql.CreateBoard(&board, true); err != nil {
			logger.Err(err).Caller().Send()
			return "", err
		}
	}

	sections, err := gcsql.GetAllSections(false)
	if err != nil {
		logger.Err(err).Caller().Send()
		return "", server.NewServerError("unable to get board sections", http.StatusInternalServerError)
	}
	boards, err := gcsql.GetAllBoards(false)
	if err != nil {
		logger.Err(err).Caller().Send()
		return "", server.NewServerError("unable to get board list", http.StatusInternalServerError)
	}

	var buf bytes.Buffer
	if err = serverutil.MinifyTemplate(gctemplates.ManageBoards,
		map[string]any{
			"siteConfig": config.GetSiteConfig(),
			"sections":   sections,
			"boards":     boards,
			"board": gcsql.Board{
				AnonymousName:    "Anonymous",
				MaxFilesize:      1000 * 1000 * 15,
				EnableCatalog:    true,
				AutosageAfter:    200,
				NoImagesAfter:    -1,
				MaxMessageLength: 1500,
			},
			"boardConfig": config.GetBoardConfig(""),
			"editing":     false,
		}, &buf, "text/html"); err != nil {
		logger.Err(err).Str("template", gctemplates.ManageBoards).Caller().Send()
		return "", err
	}

	return buf.String(), nil
}

// modifyBoardCallback handles requests to /manage/boards/<boardDir> for modifying or deleting a board
func modifyBoardCallback(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	params, _ := request.Context().Value(requestContextKey{}).(bunrouter.Params)
	boardDir := params.ByName("board")

	var form createOrModifyBoardForm
	if err = forms.FillStructFromForm(request, &form); err != nil {
		logger.Warn().Err(err).Caller().Send()
		return nil, server.NewServerError(err, http.StatusBadRequest)
	}
	var requestType boardRequestType
	if request.Method == http.MethodGet {
		requestType = boardRequestTypeViewSingleBoard
	} else {
		requestType = form.requestType()
	}
	logger = logger.With().Str("requestType", requestType.String()).Str("boardDir", boardDir).Logger()
	if requestType == boardRequestTypeCancel {
		http.Redirect(writer, request, config.WebPath("/manage/boards"), http.StatusFound)
		return
	}
	if err = form.validate(logger.Warn()); err != nil {
		return nil, err
	}
	var board *gcsql.Board
	switch requestType {
	case boardRequestTypeViewSingleBoard:
		if board, err = gcsql.GetBoardFromDir(boardDir); err != nil {
			logger.Err(err).Caller().Send()
			return "", server.NewServerError("unable to get board info", http.StatusInternalServerError)
		}
	case boardRequestTypeModify:
		if board, err = gcsql.GetBoardFromDir(boardDir); err != nil {
			logger.Err(err).Caller().Send()
			return "", server.NewServerError("unable to get board info", http.StatusInternalServerError)
		}
		form.fillBoard(board)
		if err = board.ModifyInDB(); err != nil {
			logger.Err(err).Caller().Send()
			return "", server.NewServerError("unable to apply changes", http.StatusInternalServerError)
		}
		logger.Info().Msg("Modified board")
		http.Redirect(writer, request, config.WebPath("/manage/boards"), http.StatusFound)
	case boardRequestTypeDelete:
		board, err = gcsql.GetBoardFromDir(boardDir)
		if err != nil {
			logger.Err(err).Caller().Send()
			return "", server.NewServerError("unable to get board info", http.StatusInternalServerError)
		}
		if err = board.Delete(); err != nil {
			logger.Err(err).Caller().Send()
			return "", server.NewServerError("unable to delete board", http.StatusInternalServerError)
		}
		http.Redirect(writer, request, config.WebPath("/manage/boards"), http.StatusFound)
	}

	sections, err := gcsql.GetAllSections(false)
	if err != nil {
		logger.Err(err).Caller().Send()
		return "", server.NewServerError("unable to get board sections", http.StatusInternalServerError)
	}
	boards, err := gcsql.GetAllBoards(false)
	if err != nil {
		logger.Err(err).Caller().Send()
		return "", server.NewServerError("unable to get board list", http.StatusInternalServerError)
	}

	var buf bytes.Buffer
	if err = serverutil.MinifyTemplate(gctemplates.ManageBoards,
		map[string]any{
			"siteConfig":  config.GetSiteConfig(),
			"sections":    sections,
			"boards":      boards,
			"board":       board,
			"boardConfig": config.GetBoardConfig(""),
			"editing":     requestType == boardRequestTypeViewSingleBoard || requestType == boardRequestTypeModify,
		}, &buf, "text/html"); err != nil {
		logger.Err(err).Str("template", gctemplates.ManageBoards).Caller().Send()
		return "", err
	}

	return buf.String(), nil
}

// boardSectionsCallback handles requests to /manage/boardsections for creating, editing, and deleting board sections
func boardSectionsCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	section := &gcsql.Section{}
	editID := request.Form.Get("edit")
	updateID := request.Form.Get("updatesection")
	deleteID := request.Form.Get("delete")
	if editID != "" {
		if section, err = gcsql.GetSectionFromID(gcutil.HackyStringToInt(editID)); err != nil {
			logger.Err(err).Caller().Send()
			return "", &ErrStaffAction{
				ErrorField: "db",
				Action:     "boardsections",
				Message:    err.Error(),
			}
		}
	} else if updateID != "" {
		if section, err = gcsql.GetSectionFromID(gcutil.HackyStringToInt(updateID)); err != nil {
			logger.Err(err).Caller().Send()
			return "", &ErrStaffAction{
				ErrorField: "db",
				Action:     "boardsections",
				Message:    err.Error(),
			}
		}
	} else if deleteID != "" {
		if err = gcsql.DeleteSection(gcutil.HackyStringToInt(deleteID)); err != nil {
			logger.Err(err).Caller().Send()
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
			logger.Err(err).Caller().Send()
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
			logger.Err(err).Caller().Send()
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
		logger.Err(err).Caller().Send()
		return "", err
	}
	pageBuffer := bytes.NewBufferString("")
	pageMap := map[string]any{
		"siteConfig": config.GetSiteConfig(),
		"sections":   sections,
	}
	if section.ID > 0 {
		pageMap["edit_section"] = section
	}
	if err = serverutil.MinifyTemplate(gctemplates.ManageSections, pageMap, pageBuffer, "text/html"); err != nil {
		logger.Err(err).Caller().Str("template", gctemplates.ManageSections).Send()
		return "", err
	}
	output = pageBuffer.String()
	return
}

// cleanupCallback handles requests to /manage/cleanup for performing database cleanup tasks
func cleanupCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	outputStr := ""
	if request.FormValue("run") == "Run Cleanup" {
		outputStr += "Removing deleted posts from the database.<hr />"
		if err = gcsql.PermanentlyRemoveDeletedPosts(); err != nil {
			logger.Err(err).Caller().
				Str("cleanup", "removeDeletedPosts").Send()
			err = errors.New("unable to remove deleted posts from database")
			return outputStr + "<tr><td>" + err.Error() + "</td></tr></table>", err
		}

		outputStr += "Optimizing all tables in database.<hr />"
		err = gcsql.OptimizeDatabase()
		if err != nil {
			logger.Err(err).Caller().
				Str("sql", "optimization").Send()
			err = fmt.Errorf("failed optimizing SQL tables: %w", err)
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

// fixThumbnailsCallback handles requests to /manage/fixthumbnails for regenerating missing or broken thumbnails
// TODO: potentially merge this with rebuild callbacks
func fixThumbnailsCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	board := request.FormValue("board")
	var uploads []uploadInfo
	if board != "" {
		const query = `SELECT id, op, filename, is_spoilered, width, height, thumbnail_width, thumbnail_height
		FROM DBPREFIXv_upload_info WHERE dir = ? ORDER BY created_on DESC`
		rows, err := gcsql.Query(nil, query, board)
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
				logger.Err(err).Caller().Send()
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
		logger.Err(err).Str("template", gctemplates.ManageFixThumbnails).Caller().Send()
		return "", err
	}
	return buffer.String(), nil
}

// templatesCallback handles requests to /manage/templates for overriding templates to be applied immediately without needing to restart gochan
func templatesCallback(writer http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	buf := bytes.NewBufferString("")

	selectedTemplate := request.FormValue("override")
	templatesDir := config.GetSystemCriticalConfig().TemplateDir
	var overriding string
	var templateStr string
	var templatePath string
	var successStr string
	if selectedTemplate != "" {
		logger = logger.With().Str("selectedTemplate", selectedTemplate).Logger()
		if templatePath, err = gctemplates.GetTemplatePath(selectedTemplate); err != nil {
			logger.Err(err).Caller().Msg("unable to load selected template")
			return "", fmt.Errorf("template %q does not exist", selectedTemplate)
		}
		logger = logger.With().Str("templatePath", templatePath).Logger()
		ba, err := os.ReadFile(templatePath)
		if err != nil {
			logger.Err(err).Caller().Send()
			return "", fmt.Errorf("unable to load selected template %q", selectedTemplate)
		}
		templateStr = string(ba)
	} else if overriding = request.PostFormValue("overriding"); overriding != "" {
		if templateStr = request.PostFormValue("templatetext"); templateStr == "" {
			writer.WriteHeader(http.StatusBadRequest)
			logger.Warn().Caller().Int("status", http.StatusBadRequest).
				Msg("received an empty template string")
			return "", errors.New("received an empty template string")
		}
		if _, err = gctemplates.ParseTemplate(selectedTemplate, templateStr); err != nil {
			// unable to parse submitted template
			logger.Err(err).Caller().Int("status", http.StatusBadRequest).Send()
			writer.WriteHeader(http.StatusBadRequest)
			return "", err
		}
		overrideDir := path.Join(templatesDir, "override")
		overridePath := path.Join(overrideDir, overriding)
		logger = logger.With().Str("overridePath", overridePath).Logger()

		if _, err = os.Stat(overrideDir); os.IsNotExist(err) {
			// override dir doesn't exist, create it
			if err = os.Mkdir(overrideDir, config.DirFileMode); err != nil {
				logger.Err(err).Caller().
					Int("status", http.StatusInternalServerError).
					Msg("Unable to create override directory")
				writer.WriteHeader(http.StatusInternalServerError)
				return "", err
			}
		} else if err != nil {
			// got an error checking for override dir
			logger.Err(err).Caller().
				Int("status", http.StatusInternalServerError).
				Msg("Unable to check if override directory exists")
			writer.WriteHeader(http.StatusInternalServerError)
			return "", err
		}

		// get the original template file, or the latest override if there are any
		templatePath, err := gctemplates.GetTemplatePath(overriding)
		if err != nil {
			logger.Err(err).Caller().
				Int("status", http.StatusInternalServerError).
				Msg("Unable to get original template path")
			writer.WriteHeader(http.StatusInternalServerError)
			return "", err
		}

		// read original template path into []byte to be backed up
		ba, err := os.ReadFile(templatePath)
		if err != nil {
			logger.Err(err).Caller().
				Int("status", http.StatusInternalServerError).
				Msg("Unable to read original template file")
			writer.WriteHeader(http.StatusInternalServerError)
			return "", err
		}

		// back up template to override/<overriding>-<timestamp>.bkp
		backupPath := path.Join(overrideDir, overriding) + time.Now().Format("-2006-01-02_15-04-05.bkp")
		logger = logger.With().Str("backupPath", backupPath).Logger()
		if err = os.WriteFile(backupPath, ba, config.NormalFileMode); err != nil {
			logger.Err(err).Caller().
				Int("status", http.StatusInternalServerError).
				Msg("Unable to back up template file")
			writer.WriteHeader(http.StatusInternalServerError)
			return "", errors.New("unable to back up original template file")
		}

		// write changes to disk
		if err = os.WriteFile(overridePath, []byte(templateStr), config.NormalFileMode); err != nil {
			logger.Err(err).Caller().
				Int("status", http.StatusInternalServerError).
				Msg("Unable to save changes")
			writer.WriteHeader(http.StatusInternalServerError)
			return "", err
		}

		// reload template
		if err = gctemplates.InitTemplates(overriding); err != nil {
			logger.Err(err).Caller().
				Int("status", http.StatusInternalServerError).
				Msg("Unable to reinitialize template")
			writer.WriteHeader(http.StatusInternalServerError)
			return "", err
		}
		successStr = fmt.Sprintf("%q saved successfully.\n Original backed up to %s",
			overriding, backupPath)
		logger.Info().Msg("Template successfully saved and reloaded")
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
	if err = serverutil.MinifyTemplate(gctemplates.ManageTemplates, data, buf, "text/html"); err != nil {
		logger.Err(err).Str("template", gctemplates.ManageTemplates).Caller().Send()
		return "", err
	}
	return buf.String(), nil
}

// rebuildFrontCallback handles requests to /manage/rebuildfront for rebuilding the front page
// TODO: merge all rebuild callbacks into one
func rebuildFrontCallback(_ http.ResponseWriter, _ *http.Request, _ *gcsql.Staff, wantsJSON bool, logger zerolog.Logger) (output any, err error) {
	if err = gctemplates.InitTemplates(); err != nil {
		logger.Err(err).Msg("Unable to initialize templates")
		return "", err
	}
	err = building.BuildFrontPage()
	if err != nil {
		logger.Err(err).Caller().Msg("Unable to build front page")
		return "", err
	}
	if wantsJSON {
		return map[string]string{"front": "Built front page successfully"}, err
	}
	return "Built front page successfully", err
}

// rebuildAllCallback handles requests to /manage/rebuildall for rebuilding the front page, board list, and all boards
// TODO: merge all rebuild callbacks into one
func rebuildAllCallback(_ http.ResponseWriter, _ *http.Request, _ *gcsql.Staff, wantsJSON bool, logger zerolog.Logger) (output any, err error) {
	gctemplates.InitTemplates()
	if err = gcsql.ResetBoardSectionArrays(); err != nil {
		logger.Err(err).Caller().Msg("Unable to reset board section arrays")
		return "", server.NewServerError(err, http.StatusInternalServerError)
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
		return nil, buildErr
	}
	buildMap["front"] = "Built front page successfully"
	logger.Info().Msg(buildMap["front"])

	if err = building.BuildBoardListJSON(); err != nil {
		buildErr.Message = "Error building board list: " + err.Error()
		if wantsJSON {
			return nil, buildErr
		}
		return nil, buildErr
	}
	buildMap["boardlist"] = "Built board list successfully"
	logger.Info().Msg(buildMap["boardlist"])

	if err = building.BuildBoards(false); err != nil {
		buildErr.Message = "Error building boards: " + err.Error()
		if wantsJSON {
			return nil, buildErr
		}
		return nil, buildErr
	}
	buildMap["boards"] = "Built boards successfully"
	logger.Info().Msg(buildMap["boards"])

	if err = building.BuildJS(); err != nil {
		buildErr.Message = "Error building consts.js: " + err.Error()
		if wantsJSON {
			return nil, buildErr
		}
		return nil, buildErr
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

// rebuildBoardsCallback handles requests to /manage/rebuildboards for rebuilding the board pages
// TODO: merge all rebuild callbacks into one
func rebuildBoardsCallback(_ http.ResponseWriter, _ *http.Request, _ *gcsql.Staff, wantsJSON bool, logger zerolog.Logger) (output any, err error) {
	if err = gctemplates.InitTemplates(); err != nil {
		logger.Err(err).Caller().Msg("Unable to initialize templates")
		return "", err
	}
	err = building.BuildBoards(false)
	if err != nil {
		logger.Err(err).Caller().Msg("Unable to build boards")
		return "", err
	}
	if wantsJSON {
		return map[string]any{
			"success": true,
			"message": "Boards built successfully",
		}, nil
	}
	return "Boards built successfully", nil
}

// reparseHTMLCallback handles requests to /manage/reparsehtml for reparsing all post text into HTML
// TODO: merge this and rebuild callbacks into one
func reparseHTMLCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	var outputStr string
	tx, err := gcsql.BeginTx()
	if err != nil {
		logger.Err(err).Msg("Unable to begin transaction")
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
		logger.Err(err).Caller().Msg("Unable to prepare SQL query")
		return "", err
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		logger.Err(err).Msg("Unable to query the database")
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var postID, threadID, opID, boardID int
		var messageRaw, boardDir string
		if err = rows.Scan(&postID, &messageRaw, &threadID, &opID, &boardID, &boardDir); err != nil {
			logger.Err(err).Caller().Msg("Unable to scan SQL row")
			return "", err
		}
		if formatted, err := posting.FormatMessage(messageRaw, boardDir, logger.Warn(), logger.Error()); err != nil {
			logger.Err(err).Caller().Msg("Unable to format message")
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

// viewLogCallback handles requests to /manage/viewlog for viewing the gochan log file
func viewLogCallback(_ http.ResponseWriter, _ *http.Request, _ *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	logPath := path.Join(config.GetSystemCriticalConfig().LogDir, "gochan.log")
	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		logger.Err(err).Caller().Send()
		return "", errors.New("unable to open log file")
	}
	buf := bytes.NewBufferString("")
	err = serverutil.MinifyTemplate(gctemplates.ManageViewLog, map[string]any{
		"logText": string(logBytes),
	}, buf, "text/html")
	if err != nil {
		logger.Err(err).Str("template", gctemplates.ManageViewLog).Caller().Send()
		return "", err
	}
	return buf.String(), nil
}

func registerAdminPages() {
	RegisterManagePage("updateannouncements", "Update staff announcements", AdminPerms, NoJSON, updateAnnouncementsCallback)
	RegisterManagePage("boards", "Boards", AdminPerms, NoJSON, boardsCallback)
	RegisterManagePageWithMethods("boards/:board", "Modify Board", AdminPerms, NoJSON, true, modifyBoardCallback, http.MethodGet, http.MethodPost)
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

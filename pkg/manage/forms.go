package manage

import (
	"net/http"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/rs/zerolog"
)

var (
	ErrMessageLength   = server.NewServerError("maximum message length must be greater than minimum message length", http.StatusBadRequest)
	ErrBoardDirSlashes = server.NewServerError("board directory field must not contain slashes", http.StatusBadRequest)
)

const (
	boardRequestTypeViewBoards boardRequestType = iota
	boardRequestTypeViewSingleBoard
	boardRequestTypeCancel
	boardRequestTypeCreate
	boardRequestTypeModify
	boardRequestTypeDelete
)

type boardRequestType int

func (brt boardRequestType) String() string {
	switch brt {
	case boardRequestTypeViewBoards:
		return "viewBoards"
	case boardRequestTypeViewSingleBoard:
		return "viewSingleBoard"
	case boardRequestTypeCancel:
		return "cancel"
	case boardRequestTypeCreate:
		return "create"
	case boardRequestTypeModify:
		return "modify"
	case boardRequestTypeDelete:
		return "delete"
	}
	return "unknown"
}

func boardsRequestType(request *http.Request) boardRequestType {
	if request.PostFormValue("docreate") != "" {
		return boardRequestTypeCreate
	}
	if request.FormValue("dodelete") != "" {
		return boardRequestTypeDelete
	}
	if request.PostFormValue("domodify") != "" {
		return boardRequestTypeModify
	}
	if request.PostFormValue("docancel") != "" {
		return boardRequestTypeCancel
	}
	if request.URL.Path != config.WebPath("/manage/boards") {
		return boardRequestTypeViewSingleBoard
	}
	return boardRequestTypeViewBoards
}

type createOrModifyBoardForm struct {
	Dir              string `form:"dir,notempty" method:"POST"`
	Title            string `form:"title,required,notempty" method:"POST"`
	Subtitle         string `form:"subtitle,required" method:"POST"`
	Description      string `form:"description,required" method:"POST"`
	Section          int    `form:"section,required" method:"POST"`
	NavBarPosition   int    `form:"navbarposition,required" method:"POST"`
	MaxFileSize      int    `form:"maxfilesize,required" method:"POST"`
	MaxThreads       int    `form:"maxthreads,required" method:"POST"`
	DefaultStyle     string `form:"defaultstyle,required,notempty" method:"POST"`
	Locked           bool   `form:"locked" method:"POST"`
	AnonName         string `form:"anonname" method:"POST"`
	AutosageAfter    int    `form:"autosageafter,required" method:"POST"`
	NoUploadsAfter   int    `form:"nouploadsafter,required" method:"POST"`
	MaxMessageLength int    `form:"maxmessagelength,required" method:"POST"`
	MinMessageLength int    `form:"minmessagelength,required" method:"POST"`
	EmbedsAllowed    bool   `form:"embedsallowed" method:"POST"`
	RedirectToThread bool   `form:"redirecttothread" method:"POST"`
	RequireFile      bool   `form:"requirefile" method:"POST"`
	EnableCatalog    bool   `form:"enablecatalog" method:"POST"`
	Rebuild          bool   `form:"rebuild" method:"POST"`

	// create or modify submit button
	DoCreate string `form:"docreate" method:"POST"`
	DoModify string `form:"domodify" method:"POST"`
	DoDelete string `form:"dodelete" method:"POST"`
	DoCancel string `form:"docancel" method:"POST"`
}

func (brf *createOrModifyBoardForm) validate(warnEv *zerolog.Event) (err error) {
	defer func() {
		if err != nil {
			warnEv.Err(err).Caller(1).Send()
		}
	}()
	if strings.Contains(brf.Dir, "/") || strings.Contains(brf.Dir, "\\") {
		warnEv.Str("dir", brf.Dir)
		return ErrBoardDirSlashes
	}
	if brf.MaxMessageLength < brf.MinMessageLength {
		warnEv.Int("maxMessageLength", brf.MaxMessageLength).Int("minMessageLength", brf.MinMessageLength)
		return ErrMessageLength
	}

	return nil
}

func (brf *createOrModifyBoardForm) requestType() boardRequestType {
	if brf.DoCreate != "" {
		return boardRequestTypeCreate
	}
	if brf.DoModify != "" {
		return boardRequestTypeModify
	}
	if brf.DoDelete != "" {
		return boardRequestTypeDelete
	}
	if brf.DoCancel != "" {
		return boardRequestTypeCancel
	}
	return boardRequestTypeViewBoards
}

func (brf *createOrModifyBoardForm) fillBoard(board *gcsql.Board) {
	board.Dir = brf.Dir
	board.Title = brf.Title
	board.Subtitle = brf.Subtitle
	board.Description = brf.Description
	board.SectionID = brf.Section
	board.NavbarPosition = brf.NavBarPosition
	board.MaxFilesize = brf.MaxFileSize
	board.MaxThreads = brf.MaxThreads
	board.DefaultStyle = brf.DefaultStyle
	board.Locked = brf.Locked
	board.AnonymousName = brf.AnonName
	board.AutosageAfter = brf.AutosageAfter
	board.NoImagesAfter = brf.NoUploadsAfter
	board.MaxMessageLength = brf.MaxMessageLength
	board.MinMessageLength = brf.MinMessageLength
	board.AllowEmbeds = brf.EmbedsAllowed
	board.RedirectToThread = brf.RedirectToThread
	board.RequireFile = brf.RequireFile
	board.EnableCatalog = brf.EnableCatalog
}

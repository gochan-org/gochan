package initsql

import (
	"errors"
	"net/http"
	"path"
	"strconv"
	"text/template"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
)

func banMaskTmplFunc(ban gcsql.IPBan) string {
	if ban.RangeStart == ban.RangeEnd {
		return ban.RangeStart
	}
	ipn, err := gcutil.GetIPRangeSubnet(ban.RangeStart, ban.RangeEnd)
	if err != nil {
		return "?"
	}
	return ipn.String()
}

func getBoardDirFromIDTmplFunc(id int) string {
	dir, _ := gcsql.GetBoardDir(id)
	return dir
}

func getStaffNameFromIDTmplFunc(id int) string {
	username, err := gcsql.GetStaffUsernameFromID(id)
	if err != nil {
		return "?"
	}
	return username
}

func getAppealBanIPTmplFunc(appealID int) string {
	ban, err := gcsql.GetIPBanByID(appealID)
	if err != nil || ban == nil {
		return "?"
	}
	if ban.RangeStart == ban.RangeEnd {
		return ban.RangeStart
	}
	ipn, err := gcutil.GetIPRangeSubnet(ban.RangeStart, ban.RangeEnd)
	if err != nil {
		return "?"
	}
	return ipn.String()
}

func getTopPostIDTmplFunc(post *gcsql.Post) int {
	id, _ := post.TopPostID()
	return id
}

func numRepliesTmplFunc(_, opID int) int {
	num, err := gcsql.GetThreadReplyCountFromOP(opID)
	if err != nil {
		return 0
	}
	return num
}

func getBoardDirTmplFunc(id int) string {
	dir, err := gcsql.GetBoardDir(id)
	if err != nil {
		return ""
	}
	return dir
}

func boardPagePathTmplFunc(board *gcsql.Board, page int) string {
	return config.WebPath(board.Dir, strconv.Itoa(page)+".html")
}

func getBoardDefaultStyleTmplFunc(dir string) string {
	boardCfg := config.GetBoardConfig(dir)
	if !boardCfg.IsGlobal() {
		// /<board>/board.json exists, overriding the default them and theme set in SQL
		return boardCfg.DefaultStyle
	}
	var defaultStyle string
	err := gcsql.QueryRowTimeoutSQL(nil, "SELECT default_style FROM DBPREFIXboards WHERE dir = ?",
		[]any{dir}, []any{&defaultStyle})
	if err != nil || defaultStyle == "" {
		gcutil.LogError(err).Caller().
			Str("board", dir).
			Msg("Unable to get default style attribute of board")
		return boardCfg.DefaultStyle
	}
	return defaultStyle
}

func sectionBoardsTmplFunc(sectionID int) []gcsql.Board {
	var boards []gcsql.Board
	for _, board := range gcsql.AllBoards {
		if board.SectionID == sectionID && !board.IsHidden(false) {
			boards = append(boards, board)
		}
	}
	return boards
}

func init() {
	events.RegisterEvent([]string{"reset-boards-sections"}, func(_ string, _ ...any) error {
		return gcsql.ResetBoardSectionArrays()
	})
	gctemplates.AddTemplateFuncs(template.FuncMap{
		"banMask":              banMaskTmplFunc,
		"getBoardDirFromID":    getBoardDirFromIDTmplFunc,
		"getStaffNameFromID":   getStaffNameFromIDTmplFunc,
		"getAppealBanIP":       getAppealBanIPTmplFunc,
		"getTopPostID":         getTopPostIDTmplFunc,
		"numReplies":           numRepliesTmplFunc,
		"getBoardDir":          getBoardDirTmplFunc,
		"boardPagePath":        boardPagePathTmplFunc,
		"getBoardDefaultStyle": getBoardDefaultStyleTmplFunc,
		"sectionBoards":        sectionBoardsTmplFunc,
	})
	gcsql.RegisterStringConditionHandler("ahash", func(r *http.Request, _ *gcsql.Post, u *gcsql.Upload, fc *gcsql.FilterCondition) (bool, error) {
		if u == nil {
			return false, nil
		}
		boardID, err := strconv.Atoi(r.PostFormValue("boardid"))
		if err != nil {
			// boardid is assumed to have already been checked, but just in case...
			return false, err
		}
		dir, err := gcsql.GetBoardDir(boardID)
		if err != nil {
			return false, err
		}
		fingerprint, err := uploads.GetFileFingerprint(path.Join(
			config.GetSystemCriticalConfig().DocumentRoot,
			dir, "src", u.Filename))
		if errors.Is(err, uploads.ErrVideoThumbFingerprint) {
			// admin hasn't enabled video thumbnail fingerprinting in the config, let it through
			return false, nil
		}
		return fingerprint == fc.Search, err
	})
}

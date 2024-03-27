package initsql

import (
	"strconv"
	"text/template"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
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

func intPtrToBoardDirTmplFunc(id *int, ifNil string, ifErr string) string {
	if id == nil {
		return ifNil
	}
	dir, err := gcsql.GetBoardDir(*id)
	if err != nil {
		return ifErr
	}
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
	err := gcsql.QueryRowSQL(`SELECT default_style FROM DBPREFIXboards WHERE dir = ?`,
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
	events.RegisterEvent([]string{"reset-boards-sections"}, func(trigger string, i ...interface{}) error {
		return gcsql.ResetBoardSectionArrays()
	})
	gctemplates.AddTemplateFuncs(template.FuncMap{
		"banMask":              banMaskTmplFunc,
		"getBoardDirFromID":    getBoardDirFromIDTmplFunc,
		"intPtrToBoardDir":     intPtrToBoardDirTmplFunc,
		"getStaffNameFromID":   getStaffNameFromIDTmplFunc,
		"getAppealBanIP":       getAppealBanIPTmplFunc,
		"getTopPostID":         getTopPostIDTmplFunc,
		"numReplies":           numRepliesTmplFunc,
		"getBoardDir":          getBoardDirTmplFunc,
		"boardPagePath":        boardPagePathTmplFunc,
		"getBoardDefaultStyle": getBoardDefaultStyleTmplFunc,
		"sectionBoards":        sectionBoardsTmplFunc,
	})
}

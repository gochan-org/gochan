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

func init() {
	events.RegisterEvent([]string{"reset-boards-sections"}, func(trigger string, i ...interface{}) error {
		return gcsql.ResetBoardSectionArrays()
	})
	gctemplates.AddTemplateFuncs(template.FuncMap{
		"banMask": func(ban gcsql.IPBan) string {
			if ban.ID < 1 {
				if ban.RangeStart == ban.RangeEnd {
					return ban.RangeStart
				}
				return ""
			}
			ipn, err := gcutil.GetIPRangeSubnet(ban.RangeStart, ban.RangeEnd)
			if err != nil {
				return "?"
			}
			return ipn.String()
		},
		"getBoardDirFromID": func(id int) string {
			dir, _ := gcsql.GetBoardDir(id)
			return dir
		},
		"intPtrToBoardDir": func(id *int, ifNil string, ifErr string) string {
			if id == nil {
				return ifNil
			}
			dir, err := gcsql.GetBoardDir(*id)
			if err != nil {
				return ifErr
			}
			return dir
		},
		"getStaffNameFromID": func(id int) string {
			username, err := gcsql.GetStaffUsernameFromID(id)
			if err != nil {
				return "?"
			}
			return username
		},
		"getAppealBanIP": func(appealID int) string {
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
		},
		"getTopPostID": func(post *gcsql.Post) int {
			id, _ := post.TopPostID()
			return id
		},
		"numReplies": func(boardid, opID int) int {
			num, err := gcsql.GetThreadReplyCountFromOP(opID)
			if err != nil {
				return 0
			}
			return num
		},
		"getBoardDir": func(id int) string {
			dir, err := gcsql.GetBoardDir(id)
			if err != nil {
				return ""
			}
			return dir
		},
		"boardPagePath": func(board *gcsql.Board, page int) string {
			return config.WebPath(board.Dir, strconv.Itoa(page)+".html")
		},
		"getBoardDefaultStyle": func(dir string) string {
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
		},
		"sectionBoards": func(sectionID int) []gcsql.Board {
			var boards []gcsql.Board
			for _, board := range gcsql.AllBoards {
				if board.SectionID == sectionID && !board.IsHidden(false) {
					boards = append(boards, board)
				}
			}
			return boards
		},
	})
}

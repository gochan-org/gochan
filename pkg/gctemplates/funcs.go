package gctemplates

import (
	"errors"
	"fmt"
	"html"
	"html/template"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
)

var (
	ErrInvalidKey = errors.New("template map expects string keys")
	ErrInvalidMap = errors.New("invalid template map call")
	maxFilename   = 10
)

var funcMap = template.FuncMap{
	// Arithmetic functions
	"add": func(a, b int) int {
		return a + b
	},
	"subtract": func(a, b int) int {
		return a - b
	},

	"isNil": func(i interface{}) bool {
		return i == nil
	},

	// Array functions
	"getSlice": func(arr []interface{}, start, length int) []interface{} {
		if start < 0 {
			start = 0
		}
		if length > len(arr) {
			length = len(arr)
		}
		return arr[start:length]
	},

	// String functions
	"intToString":  strconv.Itoa,
	"escapeString": html.EscapeString,
	"formatFilesize": func(sizeInt int) string {
		size := float32(sizeInt)
		if size < 1000 {
			return fmt.Sprintf("%d B", sizeInt)
		} else if size <= 100000 {
			return fmt.Sprintf("%0.1f KB", size/1024)
		} else if size <= 100000000 {
			return fmt.Sprintf("%0.2f MB", size/1024/1024)
		}
		return fmt.Sprintf("%0.2f GB", size/1024/1024/1024)
	},
	"formatTimestamp": func(t time.Time) string {
		return t.Format(config.GetBoardConfig("").DateTimeFormat)
	},
	"stringAppend": func(strings ...string) string {
		var appended string
		for _, str := range strings {
			appended += str
		}
		return appended
	},
	"truncateFilename": func(filename string) string {
		if len(filename) <= maxFilename {
			return filename
		}
		arr := strings.Split(filename, ".")
		if len(arr) == 1 {
			return arr[0][:maxFilename]
		}
		base := strings.Join(arr[:len(arr)-1], ".")
		if len(base) >= maxFilename {
			base = base[:maxFilename]
		}
		ext := arr[len(arr)-1:][0]
		return base + "." + ext
	},
	"truncateMessage": func(msg string, limit int, maxLines int) string {
		var truncated bool
		split := strings.Split(msg, "<br />")

		if len(split) > maxLines {
			split = split[:maxLines]
			msg = strings.Join(split, "<br />")
			truncated = true
		}

		if len(msg) < limit {
			if truncated {
				msg = msg + "..."
			}
			return msg
		}
		msg = msg[:limit]
		truncated = true

		if truncated {
			msg = msg + "..."
		}
		return msg
	},
	"truncateHTMLMessage": truncateHTML,
	"stripHTML": func(htmlStr template.HTML) string {
		return gcutil.StripHTML(string(htmlStr))
	},
	"truncateString": func(msg string, limit int, ellipsis bool) string {
		if len(msg) > limit {
			if ellipsis {
				return msg[:limit] + "..."
			}
			return msg[:limit]
		}
		return msg
	},
	"map": func(values ...interface{}) (map[string]interface{}, error) {
		dict := make(map[string]interface{})
		if len(values)%2 != 0 {
			return nil, ErrInvalidMap
		}
		for k := 0; k < len(values); k += 2 {
			key, ok := values[k].(string)
			if !ok {
				return nil, ErrInvalidKey
			}
			dict[key] = values[k+1]
		}
		return dict, nil
	},
	"until": func(t time.Time) string {
		return time.Until(t).String()
	},
	"dereference": func(a *int) int {
		if a == nil {
			return 0
		}
		return *a
	},

	// Imageboard functions
	"bannedForever": func(ban *gcsql.IPBan) bool {
		return ban.IsActive && ban.Permanent && !ban.CanAppeal
	},
	"isBanned": func(ban *gcsql.IPBan, board string) bool {
		return ban.IsActive && ban.BoardID != nil
	},
	"banMask": func(ban gcsql.IPBan) string {
		if ban.ID < 1 {
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
	"getCatalogThumbnail": func(img string) string {
		_, catalogThumb := uploads.GetThumbnailFilenames(img)
		return catalogThumb
	},
	"getTopPostID": func(post *gcsql.Post) int {
		id, _ := post.TopPostID()
		return id
	},
	"getThreadThumbnail": func(img string) string {
		thumb, _ := uploads.GetThumbnailFilenames(img)
		return thumb
	},
	"getUploadType": func(name string) string {
		return uploads.GetThumbnailExtension(path.Ext(name))
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
	"boardPagePath": func(board *gcsql.Board, page int) string {
		return config.WebPath(board.Dir, strconv.Itoa(page)+".html")
	},
	"webPath": func(part ...string) string {
		return config.WebPath(part...)
	},
	"webPathDir": func(part ...string) string {
		dir := config.WebPath(part...)
		if len(dir) > 0 && dir[len(dir)-1] != '/' {
			dir += "/"
		}
		return dir
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
	// Template convenience functions
	"makeLoop": func(n int, offset int) []int {
		loopArr := make([]int, n)
		for i := range loopArr {
			loopArr[i] = i + offset
		}
		return loopArr
	},
	"isStyleDefault": func(style string) bool {
		return style == config.GetBoardConfig("").DefaultStyle
	},
	"version": func() string {
		return config.GetVersion().String()
	},
}

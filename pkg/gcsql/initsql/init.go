package initsql

import (
	"database/sql"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"path"
	"strconv"
	"text/template"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/mattn/go-sqlite3"
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
	ban, err := gcsql.GetIPBanByID(nil, appealID)
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

func getBoardDefaultStyleTmplFunc(dir string) (string, error) {
	boardCfg := config.GetBoardConfig(dir)
	if !boardCfg.IsGlobal() {
		// /<board>/board.json exists, overriding the default them and theme set in SQL
		return boardCfg.DefaultStyle, nil
	}
	var defaultStyle string
	err := gcsql.QueryRowTimeoutSQL(nil, "SELECT default_style FROM DBPREFIXboards WHERE dir = ?",
		[]any{dir}, []any{&defaultStyle})
	if err != nil || defaultStyle == "" {
		gcutil.LogError(err).Caller().
			Str("board", dir).
			Msg("Unable to get default style attribute of board")
		return boardCfg.DefaultStyle, err
	}
	return defaultStyle, nil
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
	sql.Register("sqlite3-inet6", &sqlite3.SQLiteDriver{
		ConnectHook: func(sc *sqlite3.SQLiteConn) error {
			sc.RegisterFunc("inet6_aton", func(a string) []byte {
				ip := net.ParseIP(a)
				if ip == nil {
					return nil
				}
				ip = ip.To16()
				return ip
			}, true)

			sc.RegisterFunc("inet6_ntoa", func(n any) any {
				var ip net.IP
				switch v := n.(type) {
				case []byte:
					ip = net.IP(v).To16()
				case int64:
					ip = net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v)).To16()
				}

				if ip == nil {
					return nil
				}
				return ip.String()
			}, true)

			sc.RegisterFunc("inet_aton", func(a string) []byte {
				ip := net.ParseIP(a)
				if ip == nil {
					return nil
				}
				ip = ip.To16()
				if ip.To4() == nil {
					return nil // not a IPv4 address
				}
				return ip
			}, true)

			sc.RegisterFunc("inet_ntoa", func(n any) any {
				var ip net.IP
				switch i := n.(type) {
				case int64:
					ip = net.IPv4(byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
				case []byte:
					ip = net.IP(i).To4()
				}
				if ip == nil {
					return nil
				}
				return ip.String()
			}, true)

			sc.RegisterFunc("ip_cmp", func(ip1 any, ip2 any) any {
				var netIP1, netIP2 net.IP
				var netIPAddr1, netIPAddr2 netip.Addr
				switch v := ip1.(type) {
				case []byte:
					netIP1 = net.IP(v)
				case int64:
					netIP1 = net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v)).To16()
				case string:
					netIP1 = net.ParseIP(v)
					if netIP1 != nil {
						netIP1 = netIP1.To16()
					}
				default:
					return nil
				}
				if netIP1.To4() != nil {
					netIP1 = netIP1.To4()
				}
				switch v := ip2.(type) {
				case []byte:
					netIP2 = net.IP(v).To16()
				case int64:
					netIP2 = net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v)).To16()
				case string:
					netIP2 = net.ParseIP(v)
					if netIP2 != nil {
						netIP2 = netIP2.To16()
					}
				default:
					return nil
				}
				if netIP2.To4() != nil {
					netIP2 = netIP2.To4()
				}
				if netIP1 == nil || netIP2 == nil {
					return nil // one or both are invalid
				}
				if len(netIP1) != len(netIP2) {
					return nil // can't compare different IP classes
				}

				netIPAddr1, _ = netip.AddrFromSlice(netIP1)
				netIPAddr2, _ = netip.AddrFromSlice(netIP2)
				return netIPAddr1.Compare(netIPAddr2)
			}, true)

			return nil
		},
	})

	events.RegisterEvent([]string{"reset-boards-sections"}, func(_ string, _ ...any) error {
		if config.GetSQLConfig().DBhost != "" {
			// Only reset if SQL is configured
			return gcsql.ResetBoardSectionArrays()
		}
		return nil
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

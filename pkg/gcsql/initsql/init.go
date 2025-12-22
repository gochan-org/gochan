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

func getBoardDefaultStyleTmplFunc(dir string) string {
	boardCfg := config.GetBoardConfig(dir)
	return boardCfg.DefaultStyle
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

func ipToNetIP(ip any) (net.IP, bool) {
	var ipOut net.IP
	switch v := any(ip).(type) {
	case []byte:
		ipOut = net.IP(v).To16()
	case int64:
		ipOut = net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v)).To16()
	case string:
		parsedIP := net.ParseIP(v)
		if parsedIP == nil {
			return nil, false
		}
		ipOut = parsedIP.To16()
	}
	if ipOut == nil {
		return nil, false
	}
	return ipOut, ipOut.To4() != nil
}

func ipToString(ip any, v4 bool) string {
	netIP, isV4 := ipToNetIP(ip)
	if netIP == nil || v4 && !isV4 {
		return ""
	}
	return netIP.String()
}

func init() {
	sql.Register("sqlite3-inet6", &sqlite3.SQLiteDriver{
		ConnectHook: func(sc *sqlite3.SQLiteConn) error {
			sc.RegisterFunc("inet6_aton", func(a string) []byte {
				ip, _ := ipToNetIP(a)
				return ip
			}, true)

			sc.RegisterFunc("inet6_ntoa", func(n any) any {
				ip, _ := ipToNetIP(n)
				if ip == nil {
					return nil
				}
				return ip.String()
			}, true)

			sc.RegisterFunc("inet_aton", func(a string) []byte {
				ip, isV4 := ipToNetIP(a)
				if !isV4 {
					return nil // not a IPv4 address
				}
				return ip
			}, true)

			sc.RegisterFunc("inet_ntoa", func(n any) any {
				ipStr := ipToString(n, true)
				if ipStr == "" {
					return nil
				}
				return ipStr
			}, true)

			sc.RegisterFunc("ip_cmp", func(ip1 any, ip2 any) any {
				// var netIP1, netIP2 net.IP
				var netIPAddr1, netIPAddr2 netip.Addr
				netIP1, v4_1 := ipToNetIP(ip1)
				netIP2, v4_2 := ipToNetIP(ip2)
				if netIP1 == nil || netIP2 == nil {
					return nil // one or both are invalid
				}
				if v4_1 != v4_2 {
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

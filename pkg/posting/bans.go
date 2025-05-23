package posting

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

func showBanpage(ban *gcsql.IPBan, post *gcsql.Post, postBoard *gcsql.Board, writer http.ResponseWriter, _ *http.Request) {
	banPageBuffer := bytes.NewBufferString("")
	err := serverutil.MinifyTemplate(gctemplates.BanPage, map[string]any{
		"systemCritical": config.GetSystemCriticalConfig(),
		"siteConfig":     config.GetSiteConfig(),
		"boardConfig":    config.GetBoardConfig(postBoard.Dir),
		"ip":             post.IP,
		"ban":            ban,
		"board":          postBoard,
		"permanent":      ban.Permanent,
		"expires":        ban.ExpiresAt,
	}, banPageBuffer, "text/html")
	if err != nil {
		gcutil.LogError(err).
			Str("IP", post.IP).
			Str("building", "minifier").
			Str("template", "banpage.html").Send()
		server.ServeErrorPage(writer, "Error minifying page: "+err.Error())
		return
	}
	writer.Write(banPageBuffer.Bytes())
	gcutil.LogWarning().
		Str("IP", post.IP).
		Str("boardDir", postBoard.Dir).
		Msg("Rejected post from banned IP")
}

// checks the post IP against the IP range ban list. It returns true if a ban page or an error page was served (causing MakePost() to return)
func checkIpBan(post *gcsql.Post, postBoard *gcsql.Board, writer http.ResponseWriter, request *http.Request) bool {
	ipBan, err := gcsql.CheckIPBan(post.IP, postBoard.ID)
	if err != nil {
		gcutil.LogError(err).Caller().
			Str("IP", post.IP).
			Str("boardDir", postBoard.Dir).
			Msg("Error getting IP banned status")
		server.ServeErrorPage(writer, "Error checking banned status: "+err.Error())
		return true
	}
	if ipBan == nil {
		return false // ip is not banned and there were no errors, keep going
	}
	// IP is banned
	showBanpage(ipBan, post, postBoard, writer, request)
	return true
}

func handleAppeal(writer http.ResponseWriter, request *http.Request, infoEv *zerolog.Event, errEv *zerolog.Event) {
	banIDstr := request.FormValue("banid")
	if banIDstr == "" {
		errEv.Caller().Msg("Appeal sent without banid field")
		server.ServeErrorPage(writer, "Missing banid value")
		return
	}
	appealMsg := request.FormValue("appealmsg")
	if appealMsg == "" {
		errEv.Caller().Msg("Missing appealmsg value")
		server.ServeErrorPage(writer, "Missing or empty appeal")
		return
	}
	banID, err := strconv.Atoi(banIDstr)
	if err != nil {
		errEv.Err(err).Caller().
			Str("banIDstr", banIDstr).Send()
		server.ServeErrorPage(writer, fmt.Sprintf("Invalid banid value %q", banIDstr))
		return
	}
	gcutil.LogInt("banID", banID, infoEv, errEv)

	ban, err := gcsql.GetIPBanByID(banID)
	if err != nil {
		errEv.Err(err).Caller().Send()
		server.ServeErrorPage(writer, "Error getting ban info: "+err.Error())
		return
	}
	if ban == nil {
		infoEv.Caller().Msg("GetIPBanByID returned a nil ban (presumably not banned)")
		server.ServeErrorPage(writer, fmt.Sprintf("Invalid ban ID %d", banID))
		return
	}
	gcutil.LogStr("rangeStart", ban.RangeStart, infoEv, errEv)
	gcutil.LogStr("rangeEnd", ban.RangeEnd, infoEv, errEv)

	isCorrectIP, err := ban.IsBanned(gcutil.GetRealIP(request))
	if err != nil {
		errEv.Err(err).Caller().Send()
		server.ServeErrorPage(writer, err.Error())
	}
	if !isCorrectIP {
		errEv.Caller().
			Msg("User tried to appeal a ban from a different IP")
		server.ServeErrorPage(writer, fmt.Sprintf("Invalid ban id: %d", banID))
		return
	}
	if !ban.IsActive {
		errEv.Caller().Msg("Requested ban is not active")
		server.ServeErrorPage(writer, "Requested ban is not active")
		return
	}
	if !ban.CanAppeal {
		errEv.Caller().Msg("Rejected appeal submission, appeals denied for this ban")
		server.ServeErrorPage(writer, "You can not appeal this ban")
	}
	if ban.AppealAt.After(time.Now()) {
		errEv.Caller().
			Time("appealAt", ban.AppealAt).
			Msg("Rejected appeal submission, can't appeal yet")
		server.ServeErrorPage(writer, "You are not able to appeal this ban until "+ban.AppealAt.Format(config.GetBoardConfig("").DateTimeFormat))
	}
	if err = ban.Appeal(appealMsg); err != nil {
		errEv.Err(err).Caller().
			Str("appealMsg", appealMsg).
			Msg("Unable to submit appeal")
		server.ServeErrorPage(writer, "Unable to submit appeal")
		return
	}
	board := request.FormValue("board")
	infoEv.Str("board", board).Msg("Appeal submitted")
	http.Redirect(writer, request, config.WebPath(request.FormValue("board")), http.StatusFound)
}

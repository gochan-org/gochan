package posting

import (
	"net/http"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/serverutil"
)

func showBanpage(ban gcsql.Ban, banType string, filename string, post *gcsql.Post, postBoard *gcsql.Board, writer http.ResponseWriter, request *http.Request) {
	// TODO: possibly split file/username/filename bans into separate page template
	err := serverutil.MinifyTemplate(gctemplates.Banpage, map[string]interface{}{
		"systemCritical": config.GetSystemCriticalConfig(),
		"siteConfig":     config.GetSiteConfig(),
		"boardConfig":    config.GetBoardConfig(postBoard.Dir),
		"ban":            ban,
		"board":          postBoard,
	}, writer, "text/html")
	if err != nil {
		gcutil.LogError(err).
			Str("IP", post.IP).
			Str("building", "minifier").
			Str("banType", banType).
			Str("template", "banpage.html").Send()
		serverutil.ServeErrorPage(writer, "Error minifying page: "+err.Error())
		return
	}
	ev := gcutil.LogInfo().
		Str("IP", post.IP).
		Str("boardDir", postBoard.Dir).
		Str("banType", banType)
	switch banType {
	case "ip":
		ev.Msg("Rejected post from banned IP")
	case "username":
		ev.
			Str("name", post.Name).
			Str("tripcode", post.Tripcode).
			Msg("Rejected post with banned name/tripcode")
	case "filename":
		ev.
			Str("filename", filename).
			Msg("Rejected post with banned filename")
	}
}

// func BanHandler(writer http.ResponseWriter, request *http.Request) {
// 	ip := gcutil.GetRealIP(request)
// 	ipBan, err := gcsql.CheckIPBan(ip, 0)
// 	if err != nil {
// 		gcutil.LogError(err).
// 			Str("IP", ip).
// 			Msg("Error checking IP banned status (/banned request)")
// 		serverutil.ServeErrorPage(writer, "Error checking banned status: "+err.Error())
// 		return
// 	}

// }

// checks the post for spam. It returns true if a ban page or an error page was served (causing MakePost() to return)
func checkIpBan(post *gcsql.Post, postBoard *gcsql.Board, writer http.ResponseWriter, request *http.Request) bool {
	ipBan, err := gcsql.CheckIPBan(post.IP, postBoard.ID)
	if err != nil {
		gcutil.LogError(err).
			Str("IP", post.IP).
			Str("boardDir", postBoard.Dir).
			Msg("Error getting IP banned status")
		serverutil.ServeErrorPage(writer, "Error getting ban info"+err.Error())
		return true
	}
	if ipBan == nil {
		return false // ip is not banned and there were no errors, keep going
	}
	// IP is banned
	showBanpage(ipBan, "ip", "", post, postBoard, writer, request)
	return true
}

func checkUsernameBan(formName string, post *gcsql.Post, postBoard *gcsql.Board, writer http.ResponseWriter, request *http.Request) bool {
	if formName == "" {
		return false
	}

	nameBan, err := gcsql.CheckNameBan(formName, postBoard.ID)
	if err != nil {
		gcutil.LogError(err).
			Str("IP", post.IP).
			Str("name", formName).
			Str("boardDir", postBoard.Dir).
			Msg("Error getting name banned status")
		serverutil.ServeErrorPage(writer, "Error getting name ban info")
		return true
	}
	if nameBan == nil {
		return false // name is not banned
	}
	showBanpage(nameBan, "username", "", post, postBoard, writer, request)
	return true
}

func checkFilenameBan(filename string, post *gcsql.Post, postBoard *gcsql.Board, writer http.ResponseWriter, request *http.Request) bool {
	if filename == "" {
		return false
	}
	filenameBan, err := gcsql.CheckFilenameBan(filename, postBoard.ID)
	if err != nil {
		gcutil.LogError(err).
			Str("IP", post.IP).
			Str("filename", filename).
			Str("boardDir", postBoard.Dir).
			Msg("Error getting name banned status")
		serverutil.ServeErrorPage(writer, "Error getting filename ban info")
		return true
	}
	if filenameBan == nil {
		return false
	}
	showBanpage(filenameBan, "filename", filename, post, postBoard, writer, request)
	return true
}

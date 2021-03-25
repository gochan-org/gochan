package posting

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/serverutil"
)

const (
	_ = iota
	ThreadBan
	ImageBan
	FullBan
)

// BanHandler is used for serving ban pages
func BanHandler(writer http.ResponseWriter, request *http.Request) {
	appealMsg := request.FormValue("appealmsg")
	// banStatus, err := getBannedStatus(request) TODO refactor to use ipban
	var banStatus gcsql.BanInfo
	var err error

	if appealMsg != "" {
		if banStatus.BannedForever() {
			fmt.Fprint(writer, "No.")
			return
		}
		escapedMsg := html.EscapeString(appealMsg)
		if err = gcsql.AddBanAppeal(banStatus.ID, escapedMsg); err != nil {
			serverutil.ServeErrorPage(writer, err.Error())
		}
		fmt.Fprint(writer,
			"Appeal sent. It will (hopefully) be read by a staff member. check "+config.Config.SiteWebfolder+"banned occasionally for a response",
		)
		return
	}

	if err != nil && err != sql.ErrNoRows {
		serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
			"Error getting banned status:", err.Error()))
		return
	}

	if err = serverutil.MinifyTemplate(gctemplates.Banpage, map[string]interface{}{
		"config": config.Config, "ban": banStatus, "banBoards": banStatus.Boards, "post": gcsql.Post{},
	}, writer, "text/html"); err != nil {
		serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
			"Error minifying page template: ", err.Error()))
		return
	}
}

// Checks check poster's name/tripcode/file checksum (from Post post) for banned status
// returns ban table if the user is banned or sql.ErrNoRows if they aren't
func getBannedStatus(request *http.Request) (*gcsql.BanInfo, error) {
	formName := request.FormValue("postname")
	var tripcode string
	if formName != "" {
		parsedName := gcutil.ParseName(formName)
		tripcode += parsedName["name"]
		if tc, ok := parsedName["tripcode"]; ok {
			tripcode += "!" + tc
		}
	}
	ip := gcutil.GetRealIP(request)

	var filename string
	var checksum string
	file, fileHandler, err := request.FormFile("imagefile")
	if err == nil {
		html.EscapeString(fileHandler.Filename)
		if data, err2 := ioutil.ReadAll(file); err2 == nil {
			checksum = fmt.Sprintf("%x", md5.Sum(data))
		}
		file.Close()
	}
	return gcsql.CheckBan(ip, tripcode, filename, checksum)
}

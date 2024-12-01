package building

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
)

var (
	bbcodeTagRE = regexp.MustCompile(`\[/?[^\[\]\s]+\]`)
)

type frontPagePost struct {
	Board         string
	URL           string
	ThumbURL      string
	Filename      string
	FileDeleted   bool
	MessageSample string
}

func getFrontPagePosts() ([]frontPagePost, error) {
	siteCfg := config.GetSiteConfig()
	var query string

	if siteCfg.RecentPostsWithNoFile {
		// get recent posts, including those with no file
		query = "SELECT * FROM DBPREFIXv_front_page_posts"
	} else {
		query = "SELECT * FROM DBPREFIXv_front_page_posts_with_file"
	}
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(siteCfg.MaxRecentPosts)

	rows, cancel, err := gcsql.QueryTimeoutSQL(nil, query)
	defer cancel()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recentPosts []frontPagePost
	for rows.Next() {
		var post frontPagePost
		var id, topPostID string
		var message, boardDir, filename string
		err = rows.Scan(&id, &message, &boardDir, &filename, &topPostID)
		if err != nil {
			return nil, err
		}
		message = bbcodeTagRE.ReplaceAllString(message, "")
		if len(message) > 40 {
			message = message[:37] + "..."
		}
		thumbnail, _ := uploads.GetThumbnailFilenames(filename)
		post = frontPagePost{
			Board:         boardDir,
			URL:           config.WebPath(boardDir, "res", topPostID+".html") + "#" + id,
			ThumbURL:      config.WebPath(boardDir, "thumb", thumbnail),
			Filename:      filename,
			FileDeleted:   filename == "deleted",
			MessageSample: message,
		}

		recentPosts = append(recentPosts, post)
	}
	return recentPosts, rows.Close()
}

// BuildFrontPage builds the front page using templates/front.html
func BuildFrontPage() error {
	errEv := gcutil.LogError(nil).
		Str("template", "front")
	defer errEv.Discard()
	err := gctemplates.InitTemplates(gctemplates.FrontPage)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Error loading front page template: " + err.Error())
	}
	criticalCfg := config.GetSystemCriticalConfig()
	frontFile, err := os.OpenFile(path.Join(criticalCfg.DocumentRoot, "index.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.NormalFileMode)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Failed opening front page for writing: " + err.Error())
	}

	if err = config.TakeOwnershipOfFile(frontFile); err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Failed setting file ownership for front page: " + err.Error())
	}

	var recentPostsArr []frontPagePost
	siteCfg := config.GetSiteConfig()
	recentPostsArr, err = getFrontPagePosts()
	if err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Failed loading recent posts: " + err.Error())
	}
	if err = serverutil.MinifyTemplate(gctemplates.FrontPage, map[string]interface{}{
		"siteConfig":  siteCfg,
		"sections":    gcsql.AllSections,
		"boards":      gcsql.AllBoards,
		"boardConfig": config.GetBoardConfig(""),
		"recentPosts": recentPostsArr,
	}, frontFile, "text/html"); err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Failed executing front page template: " + err.Error())
	}
	return frontFile.Close()
}

// BuildPageHeader is a convenience function for automatically generating the top part
// of every normal HTML page
func BuildPageHeader(writer io.Writer, pageTitle string, board string, misc map[string]interface{}) error {
	phMap := map[string]interface{}{
		"pageTitle":     pageTitle,
		"documentTitle": pageTitle + " - " + config.GetSiteConfig().SiteName,
		"siteConfig":    config.GetSiteConfig(),
		"sections":      gcsql.AllSections,
		"boards":        gcsql.AllBoards,
		"boardConfig":   config.GetBoardConfig(board),
	}
	for k, val := range misc {
		phMap[k] = val
	}
	return serverutil.MinifyTemplate(gctemplates.PageHeader, phMap, writer, "text/html")
}

// BuildPageFooter is a convenience function for automatically generating the bottom
// of every normal HTML page
func BuildPageFooter(writer io.Writer) (err error) {
	return serverutil.MinifyTemplate(gctemplates.PageFooter,
		map[string]interface{}{}, writer, "text/html")
}

// BuildJS minifies (if enabled) consts.js, which is built from a template
func BuildJS() error {
	// build consts.js from template
	err := gctemplates.InitTemplates(gctemplates.JsConsts)
	errEv := gcutil.LogError(nil).Str("building", "consts.js")
	defer errEv.Discard()
	if err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Error loading consts.js template:" + err.Error())
	}

	boardCfg := config.GetBoardConfig("")
	criticalCfg := config.GetSystemCriticalConfig()
	constsJSPath := path.Join(criticalCfg.DocumentRoot, "js", "consts.js")
	constsJSFile, err := os.OpenFile(constsJSPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, config.NormalFileMode)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("error opening consts.js for writing: %s", err.Error())
	}

	if err = config.TakeOwnershipOfFile(constsJSFile); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("unable to update file ownership for consts.js: %s", err.Error())
	}

	if err = serverutil.MinifyTemplate(gctemplates.JsConsts, map[string]any{
		"styles":       boardCfg.Styles,
		"defaultStyle": boardCfg.DefaultStyle,
		"webroot":      criticalCfg.WebRoot,
		"timezone":     criticalCfg.TimeZone,
		"fileTypes":    boardCfg.AllowOtherExtensions,
	}, constsJSFile, "text/javascript"); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("error building consts.js: %s", err.Error())
	}
	return constsJSFile.Close()
}

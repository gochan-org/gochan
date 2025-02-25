package building

import (
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
		query = "SELECT id, message_raw, dir, filename, op_id FROM DBPREFIXv_front_page_posts"
	} else {
		query = "SELECT id, message_raw, dir, filename, op_id FROM DBPREFIXv_front_page_posts_with_file"
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
		return fmt.Errorf("failed loading front page template: %w", err)
	}
	criticalCfg := config.GetSystemCriticalConfig()
	frontFile, err := os.OpenFile(path.Join(criticalCfg.DocumentRoot, "index.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.NormalFileMode)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed opening front page for writing: %w", err)
	}

	if err = config.TakeOwnershipOfFile(frontFile); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed setting file ownership for front page: %w", err)
	}

	var recentPostsArr []frontPagePost
	siteCfg := config.GetSiteConfig()
	recentPostsArr, err = getFrontPagePosts()
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed loading recent posts: %w", err)
	}
	if err = serverutil.MinifyTemplate(gctemplates.FrontPage, map[string]any{
		"siteConfig":  siteCfg,
		"sections":    gcsql.AllSections,
		"boards":      gcsql.AllBoards,
		"boardConfig": config.GetBoardConfig(""),
		"recentPosts": recentPostsArr,
	}, frontFile, "text/html"); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed executing front page template: %w", err)
	}
	return frontFile.Close()
}

// BuildPageHeader is a convenience function for automatically generating the top part
// of every normal HTML page
func BuildPageHeader(writer io.Writer, pageTitle string, board string, misc map[string]any) error {
	phMap := map[string]any{
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
		map[string]any{}, writer, "text/html")
}

// BuildJS minifies (if enabled) consts.js, which is built from a template
func BuildJS() error {
	// build consts.js from template
	err := gctemplates.InitTemplates(gctemplates.JsConsts)
	errEv := gcutil.LogError(nil).Str("building", "consts.js")
	defer errEv.Discard()
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed loading consts.js template: %w", err)
	}

	boardCfg := config.GetBoardConfig("")
	criticalCfg := config.GetSystemCriticalConfig()
	constsJSPath := path.Join(criticalCfg.DocumentRoot, "js", "consts.js")
	constsJSFile, err := os.OpenFile(constsJSPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, config.NormalFileMode)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed opening consts.js for writing: %w", err)
	}

	if err = config.TakeOwnershipOfFile(constsJSFile); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("unable to update file ownership for consts.js: %w", err)
	}

	if err = serverutil.MinifyTemplate(gctemplates.JsConsts, map[string]any{
		"styles":       boardCfg.Styles,
		"defaultStyle": boardCfg.DefaultStyle,
		"webroot":      criticalCfg.WebRoot,
		"timezone":     criticalCfg.TimeZone,
		"fileTypes":    boardCfg.AllowOtherExtensions,
	}, constsJSFile, "text/javascript"); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed building consts.js: %w", err)
	}
	return constsJSFile.Close()
}

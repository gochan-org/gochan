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

type recentPost struct {
	Board         string
	URL           string
	ThumbURL      string
	Filename      string
	FileDeleted   bool
	MessageSample string
}

func getRecentPosts() ([]recentPost, error) {
	siteCfg := config.GetSiteConfig()
	query := `SELECT
	DBPREFIXposts.id, DBPREFIXposts.message_raw,
	(SELECT dir FROM DBPREFIXboards WHERE id = t.board_id),
	COALESCE(f.filename, ''), op.id
	FROM DBPREFIXposts
	LEFT JOIN (SELECT id, board_id FROM DBPREFIXthreads) t ON t.id = DBPREFIXposts.thread_id
	LEFT JOIN (SELECT post_id, filename FROM DBPREFIXfiles) f on f.post_id = DBPREFIXposts.id
	INNER JOIN (SELECT id, thread_id FROM DBPREFIXposts WHERE is_top_post) op ON op.thread_id = DBPREFIXposts.thread_id
	WHERE DBPREFIXposts.is_deleted = FALSE`
	if !siteCfg.RecentPostsWithNoFile {
		query += " AND f.filename IS NOT NULL AND f.filename != '' AND f.filename != 'deleted'"
	}
	query += " ORDER BY DBPREFIXposts.id DESC LIMIT " + strconv.Itoa(siteCfg.MaxRecentPosts)

	rows, err := gcsql.QuerySQL(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recentPosts []recentPost
	for rows.Next() {
		var post recentPost
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
		post = recentPost{
			Board:         boardDir,
			URL:           config.WebPath(boardDir, "res", topPostID+".html") + "#" + id,
			ThumbURL:      config.WebPath(boardDir, "thumb", thumbnail),
			Filename:      filename,
			FileDeleted:   filename == "deleted",
			MessageSample: message,
		}

		recentPosts = append(recentPosts, post)
	}
	return recentPosts, nil
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
	os.Remove(path.Join(criticalCfg.DocumentRoot, "index.html"))

	frontFile, err := os.OpenFile(path.Join(criticalCfg.DocumentRoot, "index.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.GC_FILE_MODE)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Failed opening front page for writing: " + err.Error())
	}

	if err = config.TakeOwnershipOfFile(frontFile); err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Failed setting file ownership for front page: " + err.Error())
	}

	var recentPostsArr []recentPost
	siteCfg := config.GetSiteConfig()
	recentPostsArr, err = getRecentPosts()
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
		"pageTitle":   pageTitle,
		"siteConfig":  config.GetSiteConfig(),
		"sections":    gcsql.AllSections,
		"boards":      gcsql.AllBoards,
		"boardConfig": config.GetBoardConfig(board),
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
	constsJSFile, err := os.OpenFile(constsJSPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, config.GC_FILE_MODE)
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
	}, constsJSFile, "text/javascript"); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("error building consts.js: %s", err.Error())
	}
	return constsJSFile.Close()
}

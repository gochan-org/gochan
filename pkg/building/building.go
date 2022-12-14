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
	"github.com/gochan-org/gochan/pkg/serverutil"
)

var (
	bbcodeTagRE = regexp.MustCompile(`\[/?[^\[\]\s]+\]`)
)

type recentPost struct {
	Board         string
	URL           string
	ThumbURL      string
	FileDeleted   bool
	MessageSample string
}

func getRecentPosts() ([]recentPost, error) {
	siteCfg := config.GetSiteConfig()
	query := `SELECT
		DBPREFIXposts.id,
		DBPREFIXposts.message,
		DBPREFIXposts.message_raw,
		(SELECT dir FROM DBPREFIXboards WHERE id = t.board_id) AS dir,
		p.id AS top_post
	FROM
		DBPREFIXposts
	LEFT JOIN (
		SELECT id, board_id FROM DBPREFIXthreads
	) t ON t.id = DBPREFIXposts.thread_id
	INNER JOIN (
		SELECT
			id, thread_id FROM DBPREFIXposts WHERE is_top_post
	) p ON p.thread_id = DBPREFIXposts.thread_id
	WHERE DBPREFIXposts.is_deleted = FALSE LIMIT ` + strconv.Itoa(siteCfg.MaxRecentPosts)
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
		if filename == "" && !siteCfg.RecentPostsWithNoFile {
			continue
		}
		message = bbcodeTagRE.ReplaceAllString(message, "")
		if len(message) > 40 {
			message = message[:37] + "..."
		}
		post = recentPost{
			Board:         boardDir,
			URL:           config.WebPath(boardDir, "res", topPostID+".html") + "#" + id,
			ThumbURL:      config.WebPath(boardDir, "thumb", gcutil.GetThumbnailPath("post", filename)),
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
	err := gctemplates.InitTemplates("front")
	if err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Error loading front page template: " + err.Error())
	}
	criticalCfg := config.GetSystemCriticalConfig()
	os.Remove(path.Join(criticalCfg.DocumentRoot, "index.html"))

	frontFile, err := os.OpenFile(path.Join(criticalCfg.DocumentRoot, "index.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Failed opening front page for writing: " + err.Error())
	}
	defer frontFile.Close()

	var recentPostsArr []recentPost
	siteCfg := config.GetSiteConfig()
	recentPostsArr, err = getRecentPosts()
	if err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Failed loading recent posts: " + err.Error())
	}

	if err = serverutil.MinifyTemplate(gctemplates.FrontPage, map[string]interface{}{
		"webroot":      criticalCfg.WebRoot,
		"site_config":  siteCfg,
		"sections":     gcsql.AllSections,
		"boards":       gcsql.AllBoards,
		"board_config": config.GetBoardConfig(""),
		"recent_posts": recentPostsArr,
	}, frontFile, "text/html"); err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Failed executing front page template: " + err.Error())
	}
	return nil
}

// BuildPageHeader is a convenience function for automatically generating the top part
// of every normal HTML page
func BuildPageHeader(writer io.Writer, pageTitle string, board string, misc map[string]interface{}) error {
	phMap := map[string]interface{}{
		"page_title":   pageTitle,
		"webroot":      config.GetSystemCriticalConfig().WebRoot,
		"site_config":  config.GetSiteConfig(),
		"sections":     gcsql.AllSections,
		"boards":       gcsql.AllBoards,
		"board_config": config.GetBoardConfig(board),
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
		map[string]interface{}{
			"webroot": config.GetSystemCriticalConfig().WebRoot,
		}, writer, "text/html")
}

// BuildJS minifies (if enabled) consts.js, which is built from a template
func BuildJS() error {
	// build consts.js from template
	err := gctemplates.InitTemplates("js")
	if err != nil {
		gcutil.LogError(err).Str("template", "consts.js").Send()
		return errors.New("Error loading consts.js template:" + err.Error())
	}

	boardCfg := config.GetBoardConfig("")
	criticalCfg := config.GetSystemCriticalConfig()
	constsJSPath := path.Join(criticalCfg.DocumentRoot, "js", "consts.js")
	constsJSFile, err := os.OpenFile(constsJSPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		gcutil.LogError(err).
			Str("building", "consts.js").
			Str("filePath", constsJSPath).Send()
		return fmt.Errorf("Error opening %q for writing: %s", constsJSPath, err.Error())
	}
	defer constsJSFile.Close()

	if err = serverutil.MinifyTemplate(gctemplates.JsConsts,
		map[string]interface{}{
			"webroot":       criticalCfg.WebRoot,
			"styles":        boardCfg.Styles,
			"default_style": boardCfg.DefaultStyle,
			"timezone":      criticalCfg.TimeZone,
		},
		constsJSFile, "text/javascript"); err != nil {
		gcutil.LogError(err).
			Str("building", "consts.js").
			Str("filePath", constsJSPath).Send()
		return fmt.Errorf("Error building %q: %s", constsJSPath, err.Error())
	}
	return nil
}

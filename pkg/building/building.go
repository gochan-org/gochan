package building

import (
	"encoding/json"
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
	query := `SELECT id, thread_id AS threadid, message_raw,
		(SELECT dir FROM DBPREFIXboards WHERE id = (
			SELECT board_id FROM DBPREFIXthreads WHERE id = threadid)
		) AS board,
		COALESCE(
			(SELECT filename FROM DBPREFIXfiles WHERE post_id = DBPREFIXposts.id LIMIT 1),
		"") AS filename,
		(SELECT id FROM DBPREFIXposts WHERE is_top_post = TRUE AND thread_id = threadid) AS top_post
		FROM DBPREFIXposts WHERE is_deleted = FALSE LIMIT ` + strconv.Itoa(siteCfg.MaxRecentPosts)
	rows, err := gcsql.QuerySQL(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recentPosts []recentPost

	for rows.Next() {
		var post recentPost
		var id, threadID, topPostID string
		var message, boardDir, filename string
		err = rows.Scan(&id, &threadID, &message, &boardDir, &filename, &topPostID)
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
	err := gctemplates.InitTemplates("front")
	if err != nil {
		gcutil.LogError(err).
			Str("template", "front").Send()
		return errors.New("Error loading front page template: " + err.Error())
	}
	criticalCfg := config.GetSystemCriticalConfig()
	os.Remove(path.Join(criticalCfg.DocumentRoot, "index.html"))

	frontFile, err := os.OpenFile(path.Join(criticalCfg.DocumentRoot, "index.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		gcutil.LogError(err).
			Str("building", "front").Send()
		return errors.New("Failed opening front page for writing: " + err.Error())
	}
	defer frontFile.Close()

	var recentPostsArr []recentPost
	siteCfg := config.GetSiteConfig()
	recentPostsArr, err = getRecentPosts()
	if err != nil {
		gcutil.LogError(err).
			Str("building", "recent").Send()
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
		gcutil.LogError(err).
			Str("template", "front").Send()
		return errors.New("Failed executing front page template: " + err.Error())
	}
	return nil
}

// BuildBoardListJSON generates a JSON file with info about the boards
func BuildBoardListJSON() error {
	criticalCfg := config.GetSystemCriticalConfig()
	boardListFile, err := os.OpenFile(path.Join(criticalCfg.DocumentRoot, "boards.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		gcutil.LogError(err).
			Str("building", "boardsList").Send()
		return errors.New("Failed opening boards.json for writing: " + err.Error())
	}
	defer boardListFile.Close()

	boardsMap := map[string][]gcsql.Board{
		"boards": {},
	}

	// TODO: properly check if the board is in a hidden section
	boardsMap["boards"] = gcsql.AllBoards
	boardJSON, err := json.Marshal(boardsMap)
	if err != nil {
		gcutil.LogError(err).Str("building", "boards.json").Send()
		return errors.New("Failed to create boards.json: " + err.Error())
	}

	if _, err = serverutil.MinifyWriter(boardListFile, boardJSON, "application/json"); err != nil {
		gcutil.LogError(err).Str("building", "boards.json").Send()
		return errors.New("Failed writing boards.json file: " + err.Error())
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

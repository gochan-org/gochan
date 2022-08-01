package building

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/serverutil"
)

// BuildFrontPage builds the front page using templates/front.html
func BuildFrontPage() error {
	err := gctemplates.InitTemplates("front")
	if err != nil {
		return errors.New(gclog.Print(gclog.LErrorLog,
			"Error loading front page template: ", err.Error()))
	}
	criticalCfg := config.GetSystemCriticalConfig()
	os.Remove(path.Join(criticalCfg.DocumentRoot, "index.html"))
	frontFile, err := os.OpenFile(path.Join(criticalCfg.DocumentRoot, "index.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)

	if err != nil {
		return errors.New(gclog.Print(gclog.LErrorLog,
			"Failed opening front page for writing: ", err.Error()))
	}
	defer frontFile.Close()

	var recentPostsArr []gcsql.RecentPost
	siteCfg := config.GetSiteConfig()
	recentPostsArr, err = gcsql.GetRecentPostsGlobal(siteCfg.MaxRecentPosts, !siteCfg.RecentPostsWithNoFile)
	if err != nil {
		return errors.New(gclog.Print(gclog.LErrorLog,
			"Failed loading recent posts: "+err.Error()))
	}

	for b := range gcsql.AllBoards {
		if gcsql.AllBoards[b].Section == 0 {
			gcsql.AllBoards[b].Section = 1
		}
	}

	if err = serverutil.MinifyTemplate(gctemplates.FrontPage, map[string]interface{}{
		"webroot":      criticalCfg.WebRoot,
		"site_config":  siteCfg,
		"sections":     gcsql.AllSections,
		"boards":       gcsql.AllBoards,
		"board_config": config.GetBoardConfig(""),
		"recent_posts": recentPostsArr,
	}, frontFile, "text/html"); err != nil {
		return errors.New(gclog.Print(gclog.LErrorLog,
			"Failed executing front page template: "+err.Error()))
	}
	return nil
}

// BuildBoardListJSON generates a JSON file with info about the boards
func BuildBoardListJSON() error {
	criticalCfg := config.GetSystemCriticalConfig()
	boardListFile, err := os.OpenFile(path.Join(criticalCfg.DocumentRoot, "boards.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		return errors.New(
			gclog.Print(gclog.LErrorLog, "Failed opening boards.json for writing: ", err.Error()))
	}
	defer boardListFile.Close()

	boardsMap := map[string][]gcsql.Board{
		"boards": {},
	}

	boardCfg := config.GetBoardConfig("")
	// Our cooldowns are site-wide currently.
	cooldowns := gcsql.BoardCooldowns{
		NewThread:  boardCfg.NewThreadDelay,
		Reply:      boardCfg.ReplyDelay,
		ImageReply: boardCfg.ReplyDelay}

	for b := range gcsql.AllBoards {
		gcsql.AllBoards[b].Cooldowns = cooldowns
		boardsMap["boards"] = append(boardsMap["boards"], gcsql.AllBoards[b])
	}

	boardJSON, err := json.Marshal(boardsMap)
	if err != nil {
		return errors.New(
			gclog.Print(gclog.LErrorLog, "Failed to create boards.json: ", err.Error()))
	}

	if _, err = serverutil.MinifyWriter(boardListFile, boardJSON, "application/json"); err != nil {
		return errors.New(
			gclog.Print(gclog.LErrorLog, "Failed writing boards.json file: ", err.Error()))
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
		return errors.New(gclog.Println(gclog.LErrorLog,
			"Error loading consts.js template:", err.Error()))
	}

	boardCfg := config.GetBoardConfig("")
	criticalCfg := config.GetSystemCriticalConfig()
	constsJSPath := path.Join(criticalCfg.DocumentRoot, "js", "consts.js")
	constsJSFile, err := os.OpenFile(constsJSPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return errors.New(gclog.Printf(gclog.LErrorLog,
			"Error opening %q for writing: %s", constsJSPath, err.Error()))
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
		return errors.New(gclog.Printf(gclog.LErrorLog,
			"Error building %q: %s", constsJSPath, err.Error()))
	}
	return nil
}

// paginate returns a 2d array of a specified interface from a 1d array passed in,
// with a specified number of values per array in the 2d array.
// interfaceLength is the number of interfaces per array in the 2d array (e.g, threads per page)
// interf is the array of interfaces to be split up.
func paginate(interfaceLength int, interf []interface{}) [][]interface{} {
	// paginatedInterfaces = the finished interface array
	// numArrays = the current number of arrays (before remainder overflow)
	// interfacesRemaining = if greater than 0, these are the remaining interfaces
	// 	that will be added to the super-interface

	var paginatedInterfaces [][]interface{}
	numArrays := len(interf) / interfaceLength
	interfacesRemaining := len(interf) % interfaceLength
	currentInterface := 0
	for l := 0; l < numArrays; l++ {
		paginatedInterfaces = append(paginatedInterfaces,
			interf[currentInterface:currentInterface+interfaceLength])
		currentInterface += interfaceLength
	}
	if interfacesRemaining > 0 {
		paginatedInterfaces = append(paginatedInterfaces, interf[len(interf)-interfacesRemaining:])
	}
	return paginatedInterfaces
}

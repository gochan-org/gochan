package building

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

// BuildFrontPage builds the front page using templates/front.html
func BuildFrontPage() *gcutil.GcError {
	err := gctemplates.InitTemplates("front")
	if err != nil {
		return gcutil.NewError(gclog.Print(gclog.LErrorLog,
			"Error loading front page template: ", err.Error()), false)
	}
	os.Remove(path.Join(config.Config.DocumentRoot, "index.html"))
	frontFile, gErr := os.OpenFile(path.Join(config.Config.DocumentRoot, "index.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)

	if gErr != nil {
		return gcutil.NewError(gclog.Print(gclog.LErrorLog,
			"Failed opening front page for writing: ", err.Error()), false)
	}
	defer frontFile.Close()

	var recentPostsArr []gcsql.RecentPost
	recentPostsArr, err = gcsql.GetRecentPostsGlobal(config.Config.MaxRecentPosts, !config.Config.RecentPostsWithNoFile)
	if err != nil {
		return gcutil.NewError(gclog.Print(gclog.LErrorLog,
			"Failed loading recent posts: "+err.Error()), false)
	}

	for b := range gcsql.AllBoards {
		if gcsql.AllBoards[b].Section == 0 {
			gcsql.AllBoards[b].Section = 1
		}
	}

	if err = gcutil.MinifyTemplate(gctemplates.FrontPage, map[string]interface{}{
		"config":       config.Config,
		"sections":     gcsql.AllSections,
		"boards":       gcsql.AllBoards,
		"recent_posts": recentPostsArr,
	}, frontFile, "text/html"); err != nil {
		return gcutil.NewError(gclog.Print(gclog.LErrorLog,
			"Failed executing front page template: "+err.Error()), false)
	}
	return nil
}

// BuildBoardListJSON generates a JSON file with info about the boards
func BuildBoardListJSON() *gcutil.GcError {
	boardListFile, gErr := os.OpenFile(path.Join(config.Config.DocumentRoot, "boards.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if gErr != nil {
		return gcutil.NewError(
			gclog.Print(gclog.LErrorLog, "Failed opening boards.json for writing: ", gErr.Error()), false)
	}
	defer boardListFile.Close()

	boardsMap := map[string][]gcsql.Board{
		"boards": []gcsql.Board{},
	}

	// Our cooldowns are site-wide currently.
	cooldowns := gcsql.BoardCooldowns{
		NewThread:  config.Config.NewThreadDelay,
		Reply:      config.Config.ReplyDelay,
		ImageReply: config.Config.ReplyDelay}

	for b := range gcsql.AllBoards {
		gcsql.AllBoards[b].Cooldowns = cooldowns
		boardsMap["boards"] = append(boardsMap["boards"], gcsql.AllBoards[b])
	}

	boardJSON, gErr := json.Marshal(boardsMap)
	if gErr != nil {
		return gcutil.NewError(gclog.Print(gclog.LErrorLog, "Failed to create boards.json: ", gErr.Error()), false)
	}

	_, err := gcutil.MinifyWriter(boardListFile, boardJSON, "application/json")
	if err != nil {
		err.Message = gclog.Print(gclog.LErrorLog, "Failed writing boards.json file: ", err.Message)
		return err
	}
	return nil
}

// BuildJS minifies (if enabled) gochan.js and consts.js (the latter is built from a template)
func BuildJS() *gcutil.GcError {
	// minify gochan.js (if enabled)
	gochanMinJSPath := path.Join(config.Config.DocumentRoot, "javascript", "gochan.min.js")
	gochanMinJSFile, gErr := os.OpenFile(gochanMinJSPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if gErr != nil {
		return gcutil.NewError(gclog.Printf(gclog.LErrorLog,
			"Error opening %q for writing: %s", gochanMinJSPath, gErr.Error()), false)
	}
	defer gochanMinJSFile.Close()

	gochanJSPath := path.Join(config.Config.DocumentRoot, "javascript", "gochan.js")
	gochanJSBytes, gErr := ioutil.ReadFile(gochanJSPath)
	if gErr != nil {
		return gcutil.NewError(gclog.Printf(gclog.LErrorLog,
			"Error opening %q for writing: %s", gochanJSPath, gErr.Error()), false)
	}
	_, err := gcutil.MinifyWriter(gochanMinJSFile, gochanJSBytes, "text/javascript")
	if err != nil {
		config.Config.UseMinifiedGochanJS = false
		return gcutil.NewError(gclog.Printf(gclog.LErrorLog,
			"Error minifying %q: %s:", gochanMinJSPath, err.Error()), false)
	}
	config.Config.UseMinifiedGochanJS = true

	// build consts.js from template
	if err = gctemplates.InitTemplates("js"); err != nil {
		err.Message = gclog.Println(gclog.LErrorLog,
			"Error loading consts.js template:", err.Error())
		return err
	}
	constsJSPath := path.Join(config.Config.DocumentRoot, "javascript", "consts.js")
	constsJSFile, gErr := os.OpenFile(constsJSPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if gErr != nil {
		return gcutil.NewError(gclog.Printf(gclog.LErrorLog,
			"Error opening %q for writing: %s", constsJSPath, gErr.Error()), false)
	}
	defer constsJSFile.Close()

	if err = gcutil.MinifyTemplate(gctemplates.JsConsts, config.Config, constsJSFile, "text/javascript"); err != nil {
		return gcutil.NewError(gclog.Printf(gclog.LErrorLog,
			"Error building %q: %s", constsJSPath, err.Error()), true)
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

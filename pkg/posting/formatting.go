package posting

import (
	"strconv"
	"strings"
	"time"

	"github.com/frustra/bbcode"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

var (
	msgfmtr *MessageFormatter
)

// InitPosting prepares the formatter and the temp post pruner
func InitPosting() {
	msgfmtr = new(MessageFormatter)
	msgfmtr.InitBBcode()
	tempCleanerTicker = time.NewTicker(time.Minute * 5)
	go tempCleaner()
}

type MessageFormatter struct {
	// Go's garbage collection does weird things with bbcode's internal tag map.
	// Moving the bbcode compiler isntance (and eventually a Markdown compiler) to a struct
	// appears to fix this
	bbCompiler bbcode.Compiler
}

func (mf *MessageFormatter) InitBBcode() {
	if config.Config.DisableBBcode {
		return
	}
	mf.bbCompiler = bbcode.NewCompiler(true, true)
	mf.bbCompiler.SetTag("center", nil)
	mf.bbCompiler.SetTag("code", nil)
	mf.bbCompiler.SetTag("color", nil)
	mf.bbCompiler.SetTag("img", nil)
	mf.bbCompiler.SetTag("quote", nil)
	mf.bbCompiler.SetTag("size", nil)
}

func (mf *MessageFormatter) Compile(msg string) string {
	if config.Config.DisableBBcode {
		return msg
	}
	return mf.bbCompiler.Compile(msg)
}

func FormatMessage(message string) string {
	message = msgfmtr.Compile(message)
	// prepare each line to be formatted
	postLines := strings.Split(message, "<br>")
	for i, line := range postLines {
		trimmedLine := strings.TrimSpace(line)
		lineWords := strings.Split(trimmedLine, " ")
		isGreentext := false // if true, append </span> to end of line
		for w, word := range lineWords {
			if strings.LastIndex(word, "&gt;&gt;") == 0 {
				//word is a backlink
				if postID, err := strconv.Atoi(word[8:]); err == nil {
					// the link is in fact, a valid int
					var boardDir string
					var linkParent int

					if boardDir, err = gcsql.GetBoardFromPostID(postID); err != nil {
						gclog.Print(gclog.LErrorLog, "Error getting board dir for backlink: ", err.Error())
					}
					if linkParent, err = gcsql.GetThreadIDZeroIfTopPost(postID); err != nil {
						gclog.Print(gclog.LErrorLog, "Error getting post parent for backlink: ", err.Error())
					}

					// get post board dir
					if boardDir == "" {
						lineWords[w] = `<a href="javascript:;"><strike>` + word + `</strike></a>`
					} else if linkParent == 0 {
						lineWords[w] = `<a href="` + config.Config.SiteWebfolder + boardDir + `/res/` + word[8:] + `.html" class="postref">` + word + `</a>`
					} else {
						lineWords[w] = `<a href="` + config.Config.SiteWebfolder + boardDir + `/res/` + strconv.Itoa(linkParent) + `.html#` + word[8:] + `" class="postref">` + word + `</a>`
					}
				}
			} else if strings.Index(word, "&gt;") == 0 && w == 0 {
				// word is at the beginning of a line, and is greentext
				isGreentext = true
				lineWords[w] = "<span class=\"greentext\">" + word
			}
		}
		line = strings.Join(lineWords, " ")
		if isGreentext {
			line += "</span>"
		}
		postLines[i] = line
	}
	return strings.Join(postLines, "<br />")
}

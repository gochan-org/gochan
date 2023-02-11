package posting

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/frustra/bbcode"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
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
	mf.bbCompiler = bbcode.NewCompiler(true, true)
	mf.bbCompiler.SetTag("center", nil)
	// mf.bbCompiler.SetTag("code", nil)
	mf.bbCompiler.SetTag("color", nil)
	mf.bbCompiler.SetTag("img", nil)
	mf.bbCompiler.SetTag("quote", nil)
	mf.bbCompiler.SetTag("size", nil)
}

func (*MessageFormatter) ApplyWordFilters(message string, boardDir string) (string, error) {
	var filters []gcsql.Wordfilter
	var err error
	if boardDir == "" {
		filters, err = gcsql.GetWordfilters()
	} else {
		filters, err = gcsql.GetBoardWordFilters(boardDir)
	}
	if err != nil {
		return message, err
	}
	for _, wf := range filters {
		if message, err = wf.Apply(message); err != nil {
			return message, err
		}
	}
	return message, nil
}

func (mf *MessageFormatter) Compile(msg string, boardDir string) string {
	if config.GetBoardConfig(boardDir).DisableBBcode {
		return msg
	}
	return mf.bbCompiler.Compile(msg)
}

func ApplyWordFilters(message string, boardDir string) (string, error) {
	return msgfmtr.ApplyWordFilters(message, boardDir)
}

func FormatMessage(message string, boardDir string) template.HTML {
	message = msgfmtr.Compile(message, boardDir)
	// prepare each line to be formatted
	postLines := strings.Split(message, "<br>")
	for i, line := range postLines {
		trimmedLine := strings.TrimSpace(line)
		lineWords := strings.Split(trimmedLine, " ")
		isGreentext := false // if true, append </span> to end of line
		WebRoot := config.GetSystemCriticalConfig().WebRoot
		for w, word := range lineWords {
			if strings.LastIndex(word, "&gt;&gt;") == 0 {
				//word is a backlink
				if postID, err := strconv.Atoi(word[8:]); err == nil {
					// the link is in fact, a valid int
					var boardDir string
					var linkParent int
					var p gcsql.Post
					p.GetTopPost()
					if boardDir, err = gcsql.GetBoardDirFromPostID(postID); err != nil {
						gcutil.LogError(err).
							Int("postid", postID).
							Msg("Error getting board dir for backlink")
					}
					if err == gcsql.ErrBoardDoesNotExist {
						lineWords[w] = `<a href="javascript:;"><strike>` + word + `</strike></a>`
						continue
					}
					linkParent, err := gcsql.GetTopPostInThread(postID)
					if err != nil {
						gcutil.LogError(err).
							Int("postid", postID).
							Msg("Error getting post parent for backlink")
						lineWords[w] = `<a href="javascript:;"><strike>` + word + `</strike></a>`
					} else {
						lineWords[w] = fmt.Sprintf(`<a href="%s%s/res/%d.html#%d" class="postref">%s</a>`, WebRoot, boardDir, linkParent, word[8:], word)
					}
				}
			} else if strings.Index(word, "&gt;") == 0 && w == 0 {
				// word is at the beginning of a line, and is greentext
				isGreentext = true
				lineWords[w] = `<span class="greentext">` + word
			}
		}
		line = strings.Join(lineWords, " ")
		if isGreentext {
			line += "</span>"
		}
		postLines[i] = line
	}
	return template.HTML(strings.Join(postLines, "<br />"))
}

package posting

import (
	"fmt"
	"html/template"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/frustra/bbcode"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

var (
	msgfmtr         MessageFormatter
	urlRE           = regexp.MustCompile(`https?://(\S+)`)
	unsetBBcodeTags = []string{"center", "color", "img", "quote", "size"}
	diceRoller      = regexp.MustCompile(`(?i)\[(\d*)d(\d+)(?:([+-])(\d+))?\]`)
)

// InitPosting prepares the formatter and the temp post pruner
func InitPosting() {
	msgfmtr.Init()
	go tempCleaner()
}

type MessageFormatter struct {
	// Go's garbage collection does weird things with bbcode's internal tag map.
	// Moving the bbcode compiler isntance (and eventually a Markdown compiler) to a struct
	// appears to fix this
	bbCompiler bbcode.Compiler
	linkFixer  *strings.Replacer // used for fixing [url=http://...] being turned into [url=[url]http://...
}

func (mf *MessageFormatter) Init() {
	mf.bbCompiler = bbcode.NewCompiler(true, true)
	for _, tag := range unsetBBcodeTags {
		mf.bbCompiler.SetTag(tag, nil)
	}
	mf.bbCompiler.SetTag("?", func(_ *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		return &bbcode.HTMLTag{Name: "span", Attrs: map[string]string{"class": "spoiler"}}, true
	})
	mf.bbCompiler.SetTag("hide", func(_ *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		return &bbcode.HTMLTag{Name: "div", Attrs: map[string]string{"class": "hideblock hidden"}}, true
	})
	mf.linkFixer = strings.NewReplacer(
		"[url=[url]", "[url=",
		"[/url][/url]", "[/url]",
		"[url][url]", "[url]",
	)
}

func (*MessageFormatter) ApplyWordFilters(message string, boardDir string) (string, error) {
	var filters []gcsql.Wordfilter
	var err error
	if boardDir == "" {
		filters, err = gcsql.GetWordfilters(gcsql.OnlyTrue)
	} else {
		filters, err = gcsql.GetBoardWordfilters(boardDir)
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

func wrapLinksInURL(urlStr string) string {
	return "[url]" + urlStr + "[/url]"
}

func FormatMessage(message string, boardDir string) (template.HTML, error) {
	if config.GetBoardConfig(boardDir).RenderURLsAsLinks {
		message = urlRE.ReplaceAllStringFunc(message, wrapLinksInURL)
		message = msgfmtr.linkFixer.Replace(message)
	}
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
					if linkParent, boardDir, err = gcsql.GetTopPostAndBoardDirFromPostID(postID); err != nil {
						gcutil.LogError(err).Caller().Int("childPostID", postID).Msg("Unable to get top post and board")
						return "", fmt.Errorf("unable to get top post and board for post #%d", postID)
					}

					if linkParent == 0 {
						// board or op not found
						lineWords[w] = `<a href="javascript:;"><strike>` + word + `</strike></a>`
					} else {
						lineWords[w] = fmt.Sprintf(`<a href="%s%s/res/%d.html#%s" class="postref">%s</a>`, WebRoot, boardDir, linkParent, word[8:], word)
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
	return template.HTML(strings.Join(postLines, "<br />")), nil // skipcq: GSC-G203
}

func ApplyDiceRoll(p *gcsql.Post) (rollSum int, err error) {
	words := strings.Split(string(p.Message), " ")
	for w, word := range words {
		roll := diceRoller.FindStringSubmatch(word)
		if len(roll) == 0 {
			continue
		}
		numDice := 1
		if roll[1] != "" {
			numDice, err = strconv.Atoi(roll[1])
			if err != nil {
				return 0, err
			}
		}
		dieSize, err := strconv.Atoi(roll[2])
		if err != nil {
			return 0, err
		}
		if numDice < 1 || dieSize < 1 {
			return 0, fmt.Errorf("dice roll too small")
		}
		for i := 0; i < numDice; i++ {
			rollSum += rand.Intn(dieSize) + 1 // skipcq: GSC-G404
			switch roll[3] {
			case "+":
				mod, err := strconv.Atoi(roll[4])
				if err != nil {
					return 0, err
				}
				rollSum += mod
			case "-":
				mod, err := strconv.Atoi(roll[4])
				if err != nil {
					return 0, err
				}
				rollSum -= mod
			}
		}
		words[w] = fmt.Sprintf(`<span class="dice-roll">%dd%d`, numDice, dieSize)
		if roll[3] != "" {
			words[w] += roll[3] + roll[4]
		}
		words[w] += fmt.Sprintf(" = %d</span>", rollSum)
	}
	p.Message = template.HTML(strings.Join(words, " "))
	return
}

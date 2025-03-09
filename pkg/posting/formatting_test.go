package posting

import (
	"html/template"
	"regexp"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
)

const (
	versionStr         = "4.0.0"
	bbcodeMsgPreRender = `[b]Bold[/b] [i]Italics[/i] [u]Underline[/u] [url=https://gochan.org]URL[/url] [?]Spoiler[/?]
[code]Code[/code]
[hide]Hidden[/hide]`
	bbcodeMsgExpected = `<b>Bold</b> <i>Italics</i> <u>Underline</u> <a href="https://gochan.org">URL</a> <span class="spoiler">Spoiler</span><br>` +
		`<pre>Code</pre><br>` +
		`<div class="hideblock hidden">Hidden</div>`

	linkTestPreRender = `gochan.org: https://gochan.org
gochan.org with path: https://gochan.org/a
gochan.org with bad link: https://gochan.org/a">:)</a>`
	linkTestExpected = `gochan.org: <a href="https://gochan.org">https://gochan.org</a><br>` +
		`gochan.org with path: <a href="https://gochan.org/a">https://gochan.org/a</a><br>` +
		`gochan.org with bad link: <a href="https://gochan.org/a%22%3E:%29%3C/a%3E">https://gochan.org/a&#34;&gt;:)&lt;/a&gt;</a>`

	doubleTagPreRender = `[url=https://gochan.org]Gochan[/url] [url]https://gochan.org[/url]`
	doubleTagExpected  = `<a href="https://gochan.org">Gochan</a> <a href="https://gochan.org">https://gochan.org</a>`
	luaBBCodeTest      = `local bbcode = require("bbcode")
local msg = "[lua]Lua test[/lua]"
bbcode.set_tag("lua", function(node)
	return {name="span", attrs={class="lua"}}
end)`
	luaBBCodeTestExpected = `<span class="lua">Lua test</span>`
)

var (
	diceTestCases = []diceRollerTestCase{
		{
			desc: "[2d6]",
			post: gcsql.Post{
				MessageRaw: "[2d6]",
			},
			matcher:   regexp.MustCompile(`<span class="dice-roll">2d6 = \d{1,2}</span>`),
			expectMin: 2,
			expectMax: 12,
		},
		{
			desc: "[2d6+1]",
			post: gcsql.Post{
				MessageRaw: "[2d6+1]",
			},
			matcher:   regexp.MustCompile(`<span class="dice-roll">2d6\+1 = \d{1,2}</span>`),
			expectMin: 3,
			expectMax: 13,
		},
		{
			desc: "[2d6-1]",
			post: gcsql.Post{
				MessageRaw: "[2d6-1]",
			},
			matcher:   regexp.MustCompile(`<span class="dice-roll">2d6-1 = \d{1,2}</span>`),
			expectMin: 1,
			expectMax: 11,
		},
		{
			desc: "[d8]",
			post: gcsql.Post{
				MessageRaw: "[d8]",
			},
			matcher:   regexp.MustCompile(`<span class="dice-roll">1d8 = \d</span>`),
			expectMin: 1,
			expectMax: 8,
		},
		{
			desc: "before[1d6]after, no space",
			post: gcsql.Post{
				MessageRaw: "before[1d6]after",
			},
			matcher:   regexp.MustCompile(`before<span class="dice-roll">1d6 = \d</span>after`),
			expectMin: 1,
			expectMax: 6,
		},
		{
			desc: "before [1d6] after, no space (test for injection)",
			post: gcsql.Post{
				MessageRaw: `<script>alert("lol")</script>[1d6]<script>alert("lmao")</script>`,
			},
			matcher:   regexp.MustCompile(`&lt;script&gt;alert\(&#34;lol&#34;\)&lt;/script&gt;<span class="dice-roll">1d6 = \d</span>&lt;script&gt;alert\(&#34;lmao&#34;\)&lt;/script&gt;`),
			expectMin: 1,
			expectMax: 6,
		},
		{
			desc: "two dice rolls, no space",
			post: gcsql.Post{
				MessageRaw: "[d6][2d6]",
			},
			matcher:   regexp.MustCompile(`<span class="dice-roll">1d6 = \d</span><span class="dice-roll">2d6 = \d{1,2}</span>`),
			expectMin: 0,
			expectMax: 7,
		},
		{
			desc: "multiple dice rolls, no space",
			post: gcsql.Post{
				MessageRaw: "[d6][2d20-2][3d8+1]",
			},
			matcher:   regexp.MustCompile(`<span class="dice-roll">1d6 = \d</span><span class="dice-roll">2d20-2 = \d{1,2}</span><span class="dice-roll">3d8\+1 = \d{1,2}</span>`),
			expectMin: 0,
			expectMax: 38,
		},
		{
			desc: "invalid number of dice",
			post: gcsql.Post{
				MessageRaw: "[0d6]",
			},
			expectError: true,
		},
		{
			desc: "invalid die size",
			post: gcsql.Post{
				MessageRaw: "[1d0]",
			},
			expectError: true,
		},
		{
			desc: "invalid modifier",
			post: gcsql.Post{
				MessageRaw: "[1d6+]",
			},
			matcher: regexp.MustCompile(`\[1d6\+\]`),
		},
	}
)

type diceRollerTestCase struct {
	desc        string
	post        gcsql.Post
	expectError bool
	matcher     *regexp.Regexp
	expectMin   int
	expectMax   int
}

func TestBBCode(t *testing.T) {
	config.SetVersion(versionStr)
	var testFmtr MessageFormatter
	testFmtr.Init()
	rendered := testFmtr.Compile(bbcodeMsgPreRender, "")
	assert.Equal(t, bbcodeMsgExpected, rendered, "Testing BBcode rendering")
}

func TestLinks(t *testing.T) {
	config.SetVersion(versionStr)
	var testFmtr MessageFormatter
	testFmtr.Init()
	rendered := urlRE.ReplaceAllStringFunc(linkTestPreRender, wrapLinksInURL)
	rendered = testFmtr.Compile(rendered, "")
	assert.Equal(t, linkTestExpected, rendered)
}

func TestNoDoubleTags(t *testing.T) {
	config.SetVersion(versionStr)
	msgfmtr.Init()
	rendered, err := FormatMessage(doubleTagPreRender, "")
	assert.NoError(t, err)
	assert.EqualValues(t, doubleTagExpected, rendered)
}

func TestLuaBBCode(t *testing.T) {
	config.SetVersion(versionStr)
	msgfmtr.Init()
	l := lua.NewState()
	defer l.Close()
	l.PreloadModule("bbcode", PreloadBBCodeModule)
	assert.NoError(t, l.DoString(luaBBCodeTest))
	compiled := msgfmtr.bbCompiler.Compile("[lua]Lua test[/lua]")
	assert.Equal(t, luaBBCodeTestExpected, compiled)
	assert.NoError(t, l.DoString(`require("bbcode").set_tag("b", nil)`))
	assert.Equal(t, "[b]Lua test[/b]", msgfmtr.bbCompiler.Compile("[b]Lua test[/b]"))
	assert.Error(t, l.DoString(`bbcode.set_tag("lua", 1)`))
}

func diceRollRunner(t *testing.T, tC *diceRollerTestCase) {
	var err error
	tC.post.Message, err = FormatMessage(tC.post.MessageRaw, "")
	assert.NoError(t, err)
	err = ApplyDiceRoll(&tC.post)
	if tC.expectError {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
		assert.Regexp(t, tC.matcher, tC.post.Message)
	}
	if t.Failed() {
		t.FailNow()
	}
}

func TestDiceRoll(t *testing.T) {
	config.SetVersion(versionStr)
	msgfmtr.Init()
	for _, tC := range diceTestCases {
		t.Run(tC.desc, func(t *testing.T) {
			for range 100 {
				// Run the test case multiple times to account for randomness
				diceRollRunner(t, &tC)
			}
		})
	}
}

func TestHashTags(t *testing.T) {
	config.SetVersion(versionStr)
	msgfmtr.Init()
	msg := `[#tag]
[#t a g]
[ #tag]
[#tag ]
[# tag]
>greentext [#tag]
[#line
test]
[#js<script>alert("lol")</script>injection]`
	msgHTML, err := FormatMessage(msg, "test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, template.HTML(
		`<span class="hashtag">#tag</span><br />`+
			`<span class="hashtag">#t a g</span><br />`+
			`[ #tag]<br />`+
			`<span class="hashtag">#tag </span><br />`+
			`<span class="hashtag"># tag</span><br />`+
			`<span class="greentext">&gt;greentext <span class="hashtag">#tag</span></span><br />`+
			`[#line<br />`+
			`test]<br />`+
			`<span class="hashtag">#js&lt;script&gt;alert(&#34;lol&#34;)&lt;/script&gt;injection</span>`,
	), msgHTML)
}

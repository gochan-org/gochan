package posting

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/stretchr/testify/assert"
)

const (
	bbcodeMsgPreRender = `[b]Bold[/b]
[i]Italics[/i]
[u]Underline[/u]
[url=https://gochan.org]URL[/url]
[code]Code[/code]`
	bbcodeMsgPostRender = `<b>Bold</b><br>` +
		`<i>Italics</i><br>` +
		`<u>Underline</u><br>` +
		`<a href="https://gochan.org">URL</a><br>` +
		`<pre>Code</pre>`

	linkTestPreRender = `gochan.org: https://gochan.org
gochan.org with path: https://gochan.org/a
gochan.org with bad link: https://gochan.org/a">:)</a>`
	linkTestExpected = `gochan.org: <a href="https://gochan.org">https://gochan.org</a><br>` +
		`gochan.org with path: <a href="https://gochan.org/a">https://gochan.org/a</a><br>` +
		`gochan.org with bad link: <a href="https://gochan.org/a%22%3E:%29%3C/a%3E">https://gochan.org/a&#34;&gt;:)&lt;/a&gt;</a>`
)

func TestBBCode(t *testing.T) {
	config.SetDefaults()
	var testFmtr MessageFormatter
	testFmtr.InitBBcode()
	rendered := testFmtr.Compile(bbcodeMsgPreRender, "")
	assert.Equal(t, bbcodeMsgPostRender, rendered, "Testing BBcode rendering")
}

func TestLinks(t *testing.T) {
	config.SetDefaults()
	var testFmtr MessageFormatter
	testFmtr.InitBBcode()
	rendered := urlRE.ReplaceAllStringFunc(linkTestPreRender, wrapLinksInURL)
	rendered = testFmtr.Compile(rendered, "")
	assert.Equal(t, linkTestExpected, rendered)
}

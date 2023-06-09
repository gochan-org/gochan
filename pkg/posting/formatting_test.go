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
	bbcodeMsgPostRender = `<b>Bold</b><br><i>Italics</i><br><u>Underline</u><br><a href="https://gochan.org">URL</a><br><pre>Code</pre>`
)

func TestBBCode(t *testing.T) {
	config.SetDefaults()
	var testFmtr MessageFormatter
	testFmtr.InitBBcode()
	rendered := testFmtr.Compile(bbcodeMsgPreRender, "")
	assert.Equal(t, bbcodeMsgPostRender, rendered, "Testing BBcode rendering")
}

func TestLinks(t *testing.T) {

}

package gctemplates

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/gochan-org/gochan/pkg/gclog"
	x_html "golang.org/x/net/html"
)

//TruncateHTML truncates a template.HTML string to a certain visible character limit and line limit
func truncateHTML(htmlText template.HTML, characterLimit int, maxLines int) template.HTML {
	dom, err := x_html.Parse(strings.NewReader(string(htmlText)))
	if err != nil {
		gclog.Println(gclog.LErrorLog, err.Error())
		return template.HTML("Server error truncating HTML, failed to parse html")
	}
	truncateHTMLNodes(dom, characterLimit, maxLines)
	buf := new(bytes.Buffer)
	x_html.Render(buf, dom)
	return template.HTML(buf.String())
}

func removeNextSiblings(node *x_html.Node) {
	if node == nil {
		return
	}
	removeNextSiblings(node.NextSibling)
	node.Parent.RemoveChild(node)
}

func truncateHTMLNodes(node *x_html.Node, charactersLeft int, linesLeft int) (charsLeft int, lineLeft int) {
	//Uses a depth first search to map nodes and remove the rest.

	if node == nil {
		return charactersLeft, linesLeft
	}

	//if either value is 0 or less, remove next siblings and self
	if charactersLeft <= 0 || linesLeft <= 0 {
		removeNextSiblings(node)
	}

	switch node.Type {
	case x_html.ElementNode:
		//This is a tag node. If tag node is br, redude amount of lines by 1
		if strings.ToLower(node.Data) == "br" {
			linesLeft--
		}
		//Acts as normal node for the rest
		fallthrough
	case x_html.CommentNode:
		fallthrough
	case x_html.DoctypeNode:
		fallthrough
	case x_html.RawNode:
		fallthrough
	case x_html.ErrorNode:
		fallthrough
	case x_html.DocumentNode:
		//None of the nodes directly contain text
		//truncate children first
		charactersLeft, linesLeft = truncateHTMLNodes(node.FirstChild, charactersLeft, linesLeft)
		//Pass values to siblings (sibling code will immediately exit if no more chars allowed, and remove self)
		return truncateHTMLNodes(node.NextSibling, charactersLeft, linesLeft)
	case x_html.TextNode:
		if len(node.Data) > charactersLeft {
			node.Data = node.Data[0:charactersLeft-1] + "\n..."
		}
		charactersLeft -= len(node.Data)
		return truncateHTMLNodes(node.NextSibling, charactersLeft, linesLeft)
	}
	gclog.Println(gclog.LErrorLog, "Did not match any known node type, possible error?: ", node)
	return charactersLeft, linesLeft
}

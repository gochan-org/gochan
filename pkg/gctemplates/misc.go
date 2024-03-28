package gctemplates

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/gochan-org/gochan/pkg/gcutil"
	x_html "golang.org/x/net/html"
)

// truncateHTML truncates a template.HTML string to a certain visible character limit and line limit
func truncateHTML(htmlText template.HTML, characterLimit, maxLines int) template.HTML {
	if htmlText == "" {
		return ""
	}
	node, err := x_html.Parse(strings.NewReader(string(htmlText)))
	if err != nil {
		gcutil.LogError(err).Send()
		return template.HTML("Server error truncating HTML, failed to parse html")
	}
	buf := new(bytes.Buffer)
	if node.Type == x_html.DocumentNode {
		node = node.FirstChild.LastChild // FirstChild is <html>, has <head> and <body> children
	}
	truncateHTMLNodes(node, characterLimit, maxLines)
	for node = node.FirstChild; node != nil; node = node.NextSibling {
		// render all nodes inside body node
		x_html.Render(buf, node)
	}
	return template.HTML(buf.String()) // skipcq: GSC-G203
}

func removeNextSiblings(node *x_html.Node) {
	if node == nil {
		return
	}
	removeNextSiblings(node.NextSibling)
	node.Parent.RemoveChild(node)
}

func truncateHTMLNodes(node *x_html.Node, charactersLeft, linesLeft int) (charsLeft, lineLeft int) {
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
		//This is a tag node. If tag node is br, reduce amount of lines by 1
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
			node.Data = node.Data[0:charactersLeft-1] + "..."
		}
		charactersLeft -= len(node.Data)
		return truncateHTMLNodes(node.NextSibling, charactersLeft, linesLeft)
	}
	gcutil.LogError(nil).
		Interface("node", node).
		Msg("Did not match any known node type, possible unhandled error?")
	return charactersLeft, linesLeft
}

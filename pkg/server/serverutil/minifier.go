package serverutil

import (
	"html/template"
	"io"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/tdewolff/minify"
	minifyHTML "github.com/tdewolff/minify/html"
	minifyJS "github.com/tdewolff/minify/js"
	minifyJSON "github.com/tdewolff/minify/json"
)

var minifier *minify.M

// InitMinifier sets up the HTML/JS/JSON minifier if enabled in gochan.json
func InitMinifier() {
	siteConfig := config.GetSiteConfig()
	if !siteConfig.MinifyHTML && !siteConfig.MinifyJS {
		return
	}
	minifier = minify.New()
	if siteConfig.MinifyHTML {
		minifier.AddFunc("text/html", minifyHTML.Minify)
	}
	if siteConfig.MinifyJS {
		minifier.AddFunc("text/javascript", minifyJS.Minify)
		minifier.AddFunc("application/json", minifyJSON.Minify)
	}
}

func canMinify(mediaType string) bool {
	siteConfig := config.GetSiteConfig()
	if mediaType == "text/html" && siteConfig.MinifyHTML {
		return true
	}
	if (mediaType == "application/json" || mediaType == "text/javascript") && siteConfig.MinifyJS {
		return true
	}
	return false
}

// MinifyTemplate minifies the given template/data (if enabled) and returns any errors
func MinifyTemplate(tmpl *template.Template, data interface{}, writer io.Writer, mediaType string) error {
	if !canMinify(mediaType) {
		return tmpl.Execute(writer, data)
	}

	minWriter := minifier.Writer(mediaType, writer)
	defer minWriter.Close()
	return tmpl.Execute(minWriter, data)
}

// MinifyWriter minifies the given writer/data (if enabled) and returns the number of bytes written and any errors
func MinifyWriter(writer io.Writer, data []byte, mediaType string) (int, error) {
	if !canMinify(mediaType) {
		n, err := writer.Write(data)
		return n, err
	}

	minWriter := minifier.Writer(mediaType, writer)
	defer minWriter.Close()
	n, err := minWriter.Write(data)
	return n, err
}

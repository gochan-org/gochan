package gcutil

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
	if !config.Config.MinifyHTML && !config.Config.MinifyJS {
		return
	}
	minifier = minify.New()
	if config.Config.MinifyHTML {
		minifier.AddFunc("text/html", minifyHTML.Minify)
	}
	if config.Config.MinifyJS {
		minifier.AddFunc("text/javascript", minifyJS.Minify)
		minifier.AddFunc("application/json", minifyJSON.Minify)
	}
}

func canMinify(mediaType string) bool {
	if mediaType == "text/html" && config.Config.MinifyHTML {
		return true
	}
	if (mediaType == "application/json" || mediaType == "text/javascript") && config.Config.MinifyJS {
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

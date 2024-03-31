package serverutil

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/stretchr/testify/assert"
)

const (
	unminifiedHTML = `<!DOCTYPE html>
<html>
<body>
<a href="#">blah</a>

<a href="#">blah blah</a>
</body>
</html`
	unminifiedJS = `let varname = "blah";

function n(a, b, c) {
	doStuff();
}`

	unminifiedJSON = `{
	"key": "value",
	"key2": "value2",
	"key3": [

	]
}`
)

type testCaseCanMinify struct {
	desc       string
	minifyHTML bool
	minifyJS   bool
}

type testCaseMinifyWriter struct {
	testCaseCanMinify
	mediaType    string
	data         []byte
	expectOutput string
	expectError  bool
}

type argsMinifyTemplate struct {
	tmpl      string
	data      any
	mediaType string
	isTmplRef bool
}

type testCaseMinifyTemplate struct {
	testCaseCanMinify
	args               argsMinifyTemplate
	expectWriterString string
	wantErr            bool
}

func TestCanMinify(t *testing.T) {
	testCases := []testCaseCanMinify{
		{
			desc:       "minify HTML and JS",
			minifyHTML: true,
			minifyJS:   true,
		},
		{
			desc:       "minify HTML, don't minify JS",
			minifyHTML: true,
		},
		{
			desc:     "don't minify HTML, minify JS",
			minifyJS: true,
		},
		{
			desc: "don't minify HTML or JS",
		},
	}
	config.SetVersion("3.10.1")
	siteCfg := config.GetSiteConfig()
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			siteCfg.MinifyHTML = tC.minifyHTML
			siteCfg.MinifyJS = tC.minifyJS
			InitMinifier()
			assert.Equal(t, tC.minifyHTML, canMinify("text/html"))
			assert.Equal(t, tC.minifyJS, canMinify("application/json"))
			assert.Equal(t, tC.minifyJS, canMinify("text/javascript"))
		})
	}
}

func TestMinifyWriter(t *testing.T) {
	testCases := []testCaseMinifyWriter{
		{
			testCaseCanMinify: testCaseCanMinify{
				desc:       "minify HTML",
				minifyHTML: true,
				minifyJS:   true,
			},
			mediaType:    "text/html",
			data:         []byte(unminifiedHTML),
			expectOutput: "<!doctype html><a href=#>blah</a>\n<a href=#>blah blah</a>",
		},
		{
			testCaseCanMinify: testCaseCanMinify{
				desc:     "don't minify HTML",
				minifyJS: true,
			},
			mediaType:    "text/html",
			data:         []byte(unminifiedHTML),
			expectOutput: unminifiedHTML,
		},
		{
			testCaseCanMinify: testCaseCanMinify{
				desc:       "minify JavaScript",
				minifyHTML: false,
				minifyJS:   true,
			},
			mediaType:    "text/javascript",
			data:         []byte(unminifiedJS),
			expectOutput: `let varname="blah";function n(a,b,c){doStuff();}`,
		},
		{
			testCaseCanMinify: testCaseCanMinify{
				desc: "don't minify JavaScript",
			},
			mediaType:    "text/javascript",
			data:         []byte(unminifiedJS),
			expectOutput: unminifiedJS,
		},
		{
			testCaseCanMinify: testCaseCanMinify{
				desc:     "minify JSON",
				minifyJS: true,
			},
			mediaType:    "application/json",
			data:         []byte(unminifiedJSON),
			expectOutput: `{"key":"value","key2":"value2","key3":[]}`,
		},
		{
			testCaseCanMinify: testCaseCanMinify{
				desc: "don't minify JSON",
			},
			mediaType:    "application/json",
			data:         []byte(unminifiedJSON),
			expectOutput: unminifiedJSON,
		},
	}
	config.SetVersion("3.10.1")
	siteCfg := config.GetSiteConfig()
	buf := new(bytes.Buffer)
	var err error
	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			siteCfg.MinifyHTML = tC.minifyHTML
			siteCfg.MinifyJS = tC.minifyJS
			InitMinifier()

			_, err = MinifyWriter(buf, tC.data, tC.mediaType)
			if tC.expectError {
				if !assert.Error(t, err) {
					return
				}
			} else {
				if !assert.NoError(t, err) {
					return
				}
			}
			assert.Equal(t, tC.expectOutput, buf.String())
		})
	}
}

func handleWantErr(t *testing.T, tC *testCaseMinifyTemplate, err error) bool {
	t.Helper()
	if tC.wantErr {
		return assert.Error(t, err)
	}
	return assert.NoError(t, err)
}

func runMinifyTemplateTestCase(t *testing.T, tC *testCaseMinifyTemplate) {
	t.Helper()
	buf := new(bytes.Buffer)

	var tmpl *template.Template
	var err error
	if tC.args.isTmplRef {
		err = MinifyTemplate(tC.args.tmpl, tC.args.data, buf, tC.args.mediaType)
	} else {
		tmpl, err = template.New("name").Parse(tC.args.tmpl)
		if !handleWantErr(t, tC, err) {
			return
		}
		err = MinifyTemplate(tmpl, tC.args.data, buf, tC.args.mediaType)
	}
	handleWantErr(t, tC, err)
	assert.Equal(t, tC.expectWriterString, buf.String())
}

func TestMinifyTemplate(t *testing.T) {
	_, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		return
	}
	config.SetTestTemplateDir("templates")
	config.SetVersion("3.10.1")

	tmplRefStringTests := []testCaseMinifyTemplate{
		{
			testCaseCanMinify: testCaseCanMinify{
				desc:       "basic HTML template minify (compile template from tmpl)",
				minifyHTML: true,
			},
			args: argsMinifyTemplate{
				tmpl: `<!DOCTYPE html>
<html>
<head>
	<title>{{.title}}</title>
</head>
<body>
<a href="{{.url}}">{{.text}}</a>
</body>
</html>`,
				data: map[string]string{
					"url":   "https://gochan.org",
					"text":  "gochan",
					"title": "Gochan",
				},
				mediaType: "text/html",
			},
			expectWriterString: `<!doctype html><title>Gochan</title><a href=https://gochan.org>gochan</a>`,
		},
		{
			testCaseCanMinify: testCaseCanMinify{
				desc:       "basic HTML template minify (string tmplref)",
				minifyHTML: true,
			},
			args: argsMinifyTemplate{
				tmpl: gctemplates.ErrorPage,
				data: map[string]string{
					"errorTitle":  "Error",
					"errorHeader": "Error",
					"errorText":   "Error",
				},
				mediaType: "text/html",
				isTmplRef: true,
			},
			expectWriterString: `<!doctype html><meta charset=utf-8><title>Error</title><h1>Error</h1><p>Error<hr><address>Site powered by Gochan 3.10.1</address>`,
		},
		{
			testCaseCanMinify: testCaseCanMinify{
				desc: "basic HTML template no minify (string tmplref)",
			},
			args: argsMinifyTemplate{
				tmpl: gctemplates.ErrorPage,
				data: map[string]string{
					"errorTitle":  "Error",
					"errorHeader": "Error",
					"errorText":   "Error",
				},
				mediaType: "text/html",
				isTmplRef: true,
			},
			expectWriterString: `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Error</title>
</head>
<body>
<h1>Error</h1>
<p>Error</p>
<hr><address>Site powered by Gochan 3.10.1</address>
</body>
</html>`,
		},
	}

	siteCfg := config.GetSiteConfig()

	for _, tC := range tmplRefStringTests {
		t.Run(tC.desc, func(t *testing.T) {
			siteCfg.MinifyHTML = tC.minifyHTML
			siteCfg.MinifyJS = tC.minifyJS
			InitMinifier()
			gctemplates.InitTemplates()
			runMinifyTemplateTestCase(t, &tC)
		})
	}
}

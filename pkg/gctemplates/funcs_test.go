package gctemplates

import (
	"bytes"
	"html/template"
	"testing"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestArithmeticTmplFuncs(t *testing.T) {
	testCases := []struct {
		desc     string
		tmplStr  string
		expected string
		A        int
		B        int
	}{
		{desc: "addition", tmplStr: "{{add .A .B}}", expected: "-5", A: -2, B: -3},
		{desc: "subtraction", tmplStr: "{{subtract .A .B}}", expected: "1", A: -2, B: -3},
	}
	buf := bytes.NewBuffer(nil)
	var tmpl *template.Template
	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			tmpl = template.Must(template.New("name").Funcs(funcMap).Parse(tC.tmplStr))
			assert.NoError(t, tmpl.Execute(buf, tC))
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}

func TestDereferenceTmplFunc(t *testing.T) {
	const tmplStr = "{{dereference .Val}}"
	testCases := []struct {
		desc     string
		Val      *int
		expected string
	}{
		{
			desc:     "nil int",
			Val:      nil,
			expected: "0",
		},
		{
			desc:     "non-nil int",
			expected: "32",
			Val:      new(int),
		},
	}
	*testCases[1].Val = 32
	tmpl := template.Must(template.New("name").Funcs(funcMap).Parse(tmplStr))
	buf := bytes.NewBuffer(nil)
	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			assert.NoError(t, tmpl.Execute(buf, tC))
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}
func TestIsNilTmplFunc(t *testing.T) {
	const tmplStr = "{{isNil .Val}}"
	testCases := []struct {
		desc     string
		Val      any
		expected string
	}{
		{
			desc:     "nil value",
			Val:      nil,
			expected: "true",
		},
		{
			desc:     "non-nil value",
			Val:      42,
			expected: "false",
		},
		{
			desc:     "non-nil pointer",
			Val:      new(int),
			expected: "false",
		},
	}
	tmpl := template.Must(template.New("name").Funcs(funcMap).Parse(tmplStr))
	buf := bytes.NewBuffer(nil)
	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			assert.NoError(t, tmpl.Execute(buf, tC))
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}

func TestGetSliceTmplFunc(t *testing.T) {
	testCases := []struct {
		desc     string
		tmplStr  string
		expected string
		Slice    []any
		Start    int
		End      int
	}{
		{
			desc:     "nil slice",
			tmplStr:  "{{getSlice .Slice 0 10}}",
			expected: "[]",
			Slice:    nil,
		},
		{
			desc:     "normalize start and length",
			tmplStr:  "{{getSlice .Slice .Start .End}}",
			expected: "[1 2 3 4]",
			Start:    -4,
			End:      15,
			Slice:    []any{1, 2, 3, 4},
		},
		{
			desc:     "regular slice",
			tmplStr:  "{{getSlice .Slice .Start .End}}",
			expected: "[2 3]",
			Start:    1,
			End:      3,
			Slice:    []any{1, 2, 3, 4},
		},
	}
	var tmpl *template.Template
	buf := bytes.NewBuffer(nil)
	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			tmpl = template.Must(template.New("name").Funcs(funcMap).Parse(tC.tmplStr))
			assert.NoError(t, tmpl.Execute(buf, tC))
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}

func TestFormatFilesizeTmplFunc(t *testing.T) {
	const tmplStr = "{{formatFilesize .Size}}"
	testCases := []struct {
		desc     string
		Size     int
		expected string
	}{
		{
			desc:     "bytes",
			Size:     500,
			expected: "500 B",
		},
		{
			desc:     "kilobytes",
			Size:     4096,
			expected: "4.0 KB",
		},
		{
			desc:     "megabytes",
			Size:     5242880,
			expected: "5.00 MB",
		},
		{
			desc:     "gigabytes",
			Size:     5368709120,
			expected: "5.00 GB",
		},
	}
	tmpl := template.Must(template.New("name").Funcs(funcMap).Parse(tmplStr))
	buf := bytes.NewBuffer(nil)

	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			assert.NoError(t, tmpl.Execute(buf, tC))
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}

func TestFormatTimestampTmplFunc(t *testing.T) {
	config.InitTestConfig()

	tmpl := template.Must(template.New("name").Funcs(funcMap).Parse("{{formatTimestamp .Time}}"))
	buf := bytes.NewBuffer(nil)
	timeVal, err := time.Parse("01", "01")
	assert.NoError(t, err)
	tC := struct{ Time time.Time }{timeVal}
	assert.NoError(t, tmpl.Execute(buf, tC))
	assert.Equal(t, "Sat, January 01, 0000 12:00:00 AM", buf.String())
}

func TestStringAppendTmplFunc(t *testing.T) {
	testCases := []struct {
		desc     string
		tmplStr  string
		expected string
	}{
		{
			desc:     "no strings",
			tmplStr:  "{{stringAppend}}",
			expected: "",
		},
		{
			desc:     "single string",
			tmplStr:  `{{stringAppend "a"}}`,
			expected: "a",
		},
		{
			desc:     "multiple strings",
			tmplStr:  `{{stringAppend "a" "a"}}`,
			expected: "aa",
		},
	}
	var tmpl *template.Template
	buf := bytes.NewBuffer(nil)
	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			tmpl = template.Must(template.New("name").Funcs(funcMap).Parse(tC.tmplStr))
			assert.NoError(t, tmpl.Execute(buf, nil))
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}

func TestTruncateFilenameTmplFunc(t *testing.T) {
	const tmplStr = "{{truncateFilename .Filename}}"
	testCases := []struct {
		desc     string
		Filename string
		expected string
	}{
		{
			desc: "empty filename",
		},
		{
			desc:     "filename smaller than max",
			Filename: "file.png",
			expected: "file.png",
		},
		{
			desc:     "filename larger than max",
			Filename: "filename larger than max.png",
			expected: "filename l.png",
		},
		{
			desc:     "filename larger than max (no ext)",
			Filename: "filename larger than max",
			expected: "filename l",
		},
	}
	tmpl := template.Must(template.New("name").Funcs(funcMap).Parse(tmplStr))
	buf := bytes.NewBuffer(nil)

	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			assert.NoError(t, tmpl.Execute(buf, tC))
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}

func TestTruncateMessageTmplFunc(t *testing.T) {
	const tmplStr = "{{truncateMessage .Message .Limit .MaxLines}}"
	testCases := []struct {
		desc     string
		expected string
		Message  string
		Limit    int
		MaxLines int
	}{
		{
			desc: "empty string",
		},
		{
			desc:     "size within limit (single line)",
			expected: "test test",
			Message:  "test test",
			Limit:    10,
			MaxLines: 1,
		},
		{
			desc:     "size = limit (single line)",
			expected: "0123456789",
			Message:  "0123456789",
			Limit:    10,
			MaxLines: 1,
		},
		{
			desc:     "size > limit (single line)",
			expected: "blah blah...",
			Message:  "blah blah blah",
			Limit:    10,
			MaxLines: 1,
		},
		{
			desc:     "lines > max lines",
			expected: "blah\nblah...",
			Message:  "blah\nblah\nblah",
			Limit:    999,
			MaxLines: 2,
		},
	}
	tmpl := template.Must(template.New("name").Funcs(funcMap).Parse(tmplStr))
	buf := bytes.NewBuffer(nil)
	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			assert.NoError(t, tmpl.Execute(buf, tC))
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}

func TestTruncateHTMLMessageTmplFunc(t *testing.T) {
	const tmplStr = "{{truncateHTMLMessage .Message .Limit .MaxLines}}"
	testCases := []struct {
		desc     string
		expected string
		Message  template.HTML
		Limit    int
		MaxLines int
	}{
		{
			desc: "empty string",
		},
		{
			desc:     "size within limit (single line)",
			expected: "test test",
			Message:  "test test",
			Limit:    40,
			MaxLines: 1,
		},
		{
			desc:     "size = limit (single line)",
			expected: "0123456789",
			Message:  "0123456789",
			Limit:    10,
			MaxLines: 1,
		},
		{
			desc:     "size > limit (single line)",
			expected: "blah blah...",
			Message:  "blah blah blah",
			Limit:    10,
			MaxLines: 1,
		},
		{
			desc:     "lines > max lines",
			expected: "blah<br/>blah<br/>",
			Message:  "blah<br/>blah<br/>blah",
			Limit:    999,
			MaxLines: 2,
		},
		{
			desc:     "end tag after limit",
			expected: "<div>blah <div>b...</div></div>",
			Message:  "<div>blah <div>blah</div></div>",
			Limit:    7,
			MaxLines: 1,
		},
	}
	tmpl := template.Must(template.New("name").Funcs(funcMap).Parse(tmplStr))
	buf := bytes.NewBuffer(nil)
	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			assert.NoError(t, tmpl.Execute(buf, tC))
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}

func TestTruncateStringTmplFunc(t *testing.T) {
	const tmplStr = "{{truncateString .Message .Limit .Ellipsis}}"
	testCases := []struct {
		desc     string
		expected string
		Message  string
		Limit    int
		Ellipsis bool
	}{
		{
			desc:     "empty string",
			Ellipsis: true,
		},
		{
			desc:     "string within limit",
			expected: "string",
			Message:  "string",
			Limit:    50,
			Ellipsis: true,
		},
		{
			desc:     "string larger than limit, no ellipses",
			expected: "string ",
			Message:  "string string",
			Limit:    7,
		},
		{
			desc:     "string larger than limit, ellipses",
			expected: "string ...",
			Message:  "string string",
			Limit:    7,
			Ellipsis: true,
		},
	}
	tmpl := template.Must(template.New("name").Funcs(funcMap).Parse(tmplStr))
	buf := bytes.NewBuffer(nil)
	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			assert.NoError(t, tmpl.Execute(buf, tC))
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}

type vertex struct {
	X, Y int
}

func TestMapTmplFunc(t *testing.T) {
	testCases := []struct {
		desc     string
		tmplStr  string
		expected string
		err      bool
		baseData map[string]any
	}{
		{
			desc: "nil map",
		},
		{
			desc:    "map with odd-numbered arguments returns error",
			tmplStr: "{{define `subTmpl`}}x: .x\ny: .y{{end}}{{template `subTmpl` map `x` 1 `y` 2 `z`}}",
			err:     true,
		},
		{
			desc:    "map only accepts string keys",
			tmplStr: "{{define `subTmpl`}}x: .x\ny: .y{{end}}{{template `subTmpl` map `x` 1 `y` 2 1 1}}",
			err:     true,
		},
		{
			desc:     "map with data",
			tmplStr:  "{{define `subTmpl`}}x: {{.pos.X}}\ny: {{.pos.Y}}\nrotation: {{.rot}}{{end}}{{template `subTmpl` map `pos` .vertex `rot` .rotation}}",
			expected: "x: 32\ny: 34\nrotation: 127",
			baseData: map[string]any{
				"vertex":   vertex{32, 34},
				"rotation": 127,
			},
		},
	}
	var tmpl *template.Template
	buf := bytes.NewBuffer(nil)
	var err error
	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			tmpl = template.Must(template.New("name").Funcs(funcMap).Parse(tC.tmplStr))
			err = tmpl.Execute(buf, tC.baseData)
			if tC.err {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}

func TestWebPathDirTmplFunc(t *testing.T) {
	config.InitTestConfig()
	testCases := []struct {
		desc     string
		tmplStr  string
		expected string
	}{
		{
			desc:     "no input returns slash",
			tmplStr:  "{{webPathDir}}",
			expected: "/",
		},
		{
			desc:     "empty string returns slash",
			tmplStr:  `{{webPathDir ""}}`,
			expected: "/",
		},
		{
			desc:     "single slash returns slash",
			tmplStr:  `{{webPathDir "/"}}`,
			expected: "/",
		},
		{
			desc:     "multiple slashes returns single slash",
			tmplStr:  `{{webPathDir "//"}}`,
			expected: "/",
		},
		{
			desc:     "multiple multi-slashes returns single slash",
			tmplStr:  `{{webPathDir "//" "//"}}`,
			expected: "/",
		},
		{
			desc:     "general webpath usage",
			tmplStr:  `{{webPathDir "test"}}`,
			expected: "/test/",
		},
	}
	var tmpl *template.Template
	buf := bytes.NewBuffer(nil)
	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			tmpl = template.Must(template.New("name").Funcs(funcMap).Parse(tC.tmplStr))
			assert.NoError(t, tmpl.Execute(buf, nil))
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}

func TestMakeLoopTmplFunc(t *testing.T) {
	config.InitTestConfig()
	testCases := []struct {
		desc     string
		tmplStr  string
		expected string
	}{
		{
			desc:     "makeLoop range",
			tmplStr:  "{{range $_,$i := makeLoop 4 1}}{{$i}}\n{{end}}",
			expected: "1\n2\n3\n4\n",
		},
		{
			desc:     "makeLoop with n = 0",
			tmplStr:  "{{range $_,$i := makeLoop 0 1}}{{$i}}\n{{end}}",
			expected: "",
		},
	}
	var tmpl *template.Template
	buf := bytes.NewBuffer(nil)
	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			tmpl = template.Must(template.New("name").Funcs(funcMap).Parse(tC.tmplStr))
			assert.NoError(t, tmpl.Execute(buf, nil))
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}

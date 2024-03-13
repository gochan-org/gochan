package gctemplates

import (
	"errors"
	"fmt"
	"html"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

var (
	ErrInvalidKey = errors.New("template map expects string keys")
	ErrInvalidMap = errors.New("invalid template map call")
	maxFilename   = 10
)

var funcMap = template.FuncMap{
	// Arithmetic functions
	"add": func(a, b int) int {
		return a + b
	},
	"subtract": func(a, b int) int {
		return a - b
	},

	"isNil": func(i interface{}) bool {
		return i == nil
	},

	// Array functions
	"getSlice": func(arr []interface{}, start, length int) []interface{} {
		if start < 0 {
			start = 0
		}
		if length > len(arr) {
			length = len(arr)
		}
		return arr[start:length]
	},

	// String functions
	"intToString":  strconv.Itoa,
	"escapeString": html.EscapeString,
	"formatFilesize": func(sizeInt int) string {
		size := float32(sizeInt)
		if size < 1000 {
			return fmt.Sprintf("%d B", sizeInt)
		} else if size <= 100000 {
			return fmt.Sprintf("%0.1f KB", size/1024)
		} else if size <= 100000000 {
			return fmt.Sprintf("%0.2f MB", size/1024/1024)
		}
		return fmt.Sprintf("%0.2f GB", size/1024/1024/1024)
	},
	"formatTimestamp": func(t time.Time) string {
		return t.Format(config.GetBoardConfig("").DateTimeFormat)
	},
	"stringAppend": func(strings ...string) string {
		var appended string
		for _, str := range strings {
			appended += str
		}
		return appended
	},
	"truncateFilename": func(filename string) string {
		if len(filename) <= maxFilename {
			return filename
		}
		arr := strings.Split(filename, ".")
		if len(arr) == 1 {
			return arr[0][:maxFilename]
		}
		base := strings.Join(arr[:len(arr)-1], ".")
		if len(base) >= maxFilename {
			base = base[:maxFilename]
		}
		ext := arr[len(arr)-1:][0]
		return base + "." + ext
	},
	"truncateMessage": func(msg string, limit int, maxLines int) string {
		var truncated bool
		split := strings.Split(msg, "<br />")

		if len(split) > maxLines {
			split = split[:maxLines]
			msg = strings.Join(split, "<br />")
			truncated = true
		}

		if len(msg) < limit {
			if truncated {
				msg = msg + "..."
			}
			return msg
		}
		msg = msg[:limit]
		truncated = true

		if truncated {
			msg = msg + "..."
		}
		return msg
	},
	"truncateHTMLMessage": truncateHTML,
	"stripHTML": func(htmlStr template.HTML) string {
		return gcutil.StripHTML(string(htmlStr))
	},
	"truncateString": func(msg string, limit int, ellipsis bool) string {
		if len(msg) > limit {
			if ellipsis {
				return msg[:limit] + "..."
			}
			return msg[:limit]
		}
		return msg
	},
	"map": func(values ...interface{}) (map[string]interface{}, error) {
		dict := make(map[string]interface{})
		if len(values)%2 != 0 {
			return nil, ErrInvalidMap
		}
		for k := 0; k < len(values); k += 2 {
			key, ok := values[k].(string)
			if !ok {
				return nil, ErrInvalidKey
			}
			dict[key] = values[k+1]
		}
		return dict, nil
	},
	"until": func(t time.Time) string {
		return time.Until(t).String()
	},
	"dereference": func(a *int) int {
		if a == nil {
			return 0
		}
		return *a
	},

	// Imageboard functions
	"customFlagsEnabled": func(board string) bool {
		return config.GetBoardConfig(board).CustomFlags != nil
	},
	"webPath": config.WebPath,
	"webPathDir": func(part ...string) string {
		dir := config.WebPath(part...)
		if len(dir) > 0 && dir[len(dir)-1] != '/' {
			dir += "/"
		}
		return dir
	},
	// Template convenience functions
	"makeLoop": func(n int, offset int) []int {
		loopArr := make([]int, n)
		for i := range loopArr {
			loopArr[i] = i + offset
		}
		return loopArr
	},
	"isStyleDefault": func(style string) bool {
		return style == config.GetBoardConfig("").DefaultStyle
	},
	"version": func() string {
		return config.GetVersion().String()
	},
}

// AddTemplateFuncs adds the functions in the given FuncMap (map[string]any, with "any" expected to be a function)
// to the map of functions available to templates
func AddTemplateFuncs(funcs template.FuncMap) {
	for key, tFunc := range funcs {
		funcMap[key] = tFunc
	}
}

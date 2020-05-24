package gctemplates

import (
	"fmt"
	"html"
	"html/template"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	x_html "golang.org/x/net/html"
)

var funcMap = template.FuncMap{
	// Arithmetic functions
	"add": func(a, b int) int {
		return a + b
	},
	"subtract": func(a, b int) int {
		return a - b
	},

	// Comparison functions (some copied from text/template for compatibility)
	"ge": func(a int, b int) bool {
		return a >= b
	},
	"gt": func(a int, b int) bool {
		return a > b
	},
	"le": func(a int, b int) bool {
		return a <= b
	},
	"lt": func(a int, b int) bool {
		return a < b
	},
	"intEq": func(a, b int) bool {
		return a == b
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
	"len": func(arr []interface{}) int {
		return len(arr)
	},

	// String functions
	// "arrToString": arrToString,
	"intToString": strconv.Itoa,
	"escapeString": func(a string) string {
		return html.EscapeString(a)
	},
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
		return t.Format(config.Config.DateTimeFormat)
	},
	"stringAppend": func(strings ...string) string {
		var appended string
		for _, str := range strings {
			appended += str
		}
		return appended
	},
	"truncateMessage": func(msg string, limit int, maxLines int) string {
		var truncated bool
		split := strings.SplitN(msg, "<br />", -1)

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
		dom := x_html.NewTokenizer(strings.NewReader(string(htmlStr)))
		for tokenType := dom.Next(); tokenType != x_html.ErrorToken; {
			if tokenType != x_html.TextToken {
				tokenType = dom.Next()
				continue
			}
			txtContent := strings.TrimSpace(x_html.UnescapeString(string(dom.Text())))
			if len(txtContent) > 0 {
				return x_html.EscapeString(txtContent)
			}
			tokenType = dom.Next()
		}
		return ""
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

	// Imageboard functions
	"bannedForever": func(banInfo *gcsql.BanInfo) bool {
		return banInfo.BannedForever()
	},
	"isBanned": func(banInfo *gcsql.BanInfo, board string) bool {
		return banInfo.IsBanned(board)
	},
	"getCatalogThumbnail": func(img string) string {
		return gcutil.GetThumbnailPath("catalog", img)
	},
	"getThreadID": func(postInterface interface{}) (thread int) {
		post, ok := postInterface.(gcsql.Post)
		if !ok {
			thread = 0
		} else if post.ParentID == 0 {
			thread = post.ID
		} else {
			thread = post.ParentID
		}
		return
	},
	"getPostURL": func(postInterface interface{}, typeOf string, withDomain bool) (postURL string) {
		if withDomain {
			postURL = config.Config.SiteDomain
		}
		postURL += config.Config.SiteWebfolder

		if typeOf == "recent" {
			post, ok := postInterface.(*gcsql.RecentPost)
			if !ok {
				return
			}
			postURL = post.GetURL(withDomain)
		} else {
			post, ok := postInterface.(*gcsql.Post)
			if !ok {
				return
			}
			postURL = post.GetURL(withDomain)
		}
		return
	},
	"getThreadThumbnail": func(img string) string {
		return gcutil.GetThumbnailPath("thread", img)
	},
	"getUploadType": func(name string) string {
		extension := gcutil.GetFileExtension(name)
		var uploadType string
		switch extension {
		case "":
			fallthrough
		case "deleted":
			uploadType = ""
		case "webm":
			fallthrough
		case "jpg":
			fallthrough
		case "jpeg":
			fallthrough
		case "gif":
			uploadType = "jpg"
		case "png":
			uploadType = "png"
		}
		return uploadType
	},
	"imageToThumbnailPath": func(thumbType string, img string) string {
		filetype := strings.ToLower(img[strings.LastIndex(img, ".")+1:])
		if filetype == "gif" || filetype == "webm" {
			filetype = "jpg"
		}
		index := strings.LastIndex(img, ".")
		if index < 0 || index > len(img) {
			return ""
		}
		thumbSuffix := "t." + filetype
		if thumbType == "catalog" {
			thumbSuffix = "c." + filetype
		}
		return img[0:index] + thumbSuffix
	},
	"numReplies": func(boardid, threadid int) int {
		num, err := gcsql.GetReplyCount(threadid)
		if err != nil {
			return 0
		}
		return num
	},
	"getBoardDir": func(id int) string {
		var board gcsql.Board
		if err := board.PopulateData(id); err != nil {
			return ""
		}
		return board.Dir
	},

	// Template convenience functions
	"makeLoop": func(n int, offset int) []int {
		loopArr := make([]int, n)
		for i := range loopArr {
			loopArr[i] = i + offset
		}
		return loopArr
	},
	"generateConfigTable": func() string {
		configType := reflect.TypeOf(config.Config)
		tableOut := `<table style="border-collapse: collapse;" id="config"><tr><th>Field name</th><th>Value</th><th>Type</th><th>Description</th></tr>`
		numFields := configType.NumField()
		for f := 17; f < numFields-2; f++ {
			// starting at Lockdown because the earlier fields can't be safely edited from a web interface
			field := configType.Field(f)
			if field.Tag.Get("critical") != "" {
				continue
			}
			name := field.Name
			tableOut += "<tr><th>" + name + "</th><td>"
			f := reflect.Indirect(reflect.ValueOf(config.Config)).FieldByName(name)

			kind := f.Kind()
			switch kind {
			case reflect.Int:
				tableOut += `<input name="` + name + `" type="number" value="` + html.EscapeString(fmt.Sprintf("%v", f)) + `" class="config-text"/>`
			case reflect.String:
				tableOut += `<input name="` + name + `" type="text" value="` + html.EscapeString(fmt.Sprintf("%v", f)) + `" class="config-text"/>`
			case reflect.Bool:
				checked := ""
				if f.Bool() {
					checked = "checked"
				}
				tableOut += `<input name="` + name + `" type="checkbox" ` + checked + " />"
			case reflect.Slice:
				tableOut += `<textarea name="` + name + `" rows="4" cols="28">`
				arrLength := f.Len()
				for s := 0; s < arrLength; s++ {
					newLine := "\n"
					if s == arrLength-1 {
						newLine = ""
					}
					tableOut += html.EscapeString(f.Slice(s, s+1).Index(0).String()) + newLine
				}
				tableOut += "</textarea>"
			default:
				tableOut += fmt.Sprintf("%v", kind)
			}
			tableOut += "</td><td>" + kind.String() + "</td><td>"
			defaultTag := field.Tag.Get("default")
			var defaultTagHTML string
			if defaultTag != "" {
				defaultTagHTML = " <b>Default: " + defaultTag + "</b>"
			}
			tableOut += field.Tag.Get("description") + defaultTagHTML + "</td>"
			tableOut += "</tr>"
		}
		tableOut += "</table>"
		return tableOut
	},
	"isStyleDefault": func(style string) bool {
		return style == config.Config.DefaultStyle
	},
	"version": func() string {
		return config.Config.Version.String()
	},
}

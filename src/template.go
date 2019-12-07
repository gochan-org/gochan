package main

import (
	"fmt"
	"html"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"text/template"

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
	"arrToString": arrToString,
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
	"formatTimestamp": humanReadableTime,
	"printf": func(v int, format string, a ...interface{}) string {
		printf(v, format, a...)
		return ""
	},
	"println": func(v int, i ...interface{}) string {
		println(v, i...)
		return ""
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
		} else {
			msg = msg[:limit]
			truncated = true
		}

		if truncated {
			msg = msg + "..."
		}
		return msg
	},
	"stripHTML": func(htmlStr string) string {
		dom := x_html.NewTokenizer(strings.NewReader(htmlStr))
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
	"bannedForever": bannedForever,
	"getCatalogThumbnail": func(img string) string {
		return getThumbnailPath("catalog", img)
	},
	"getThreadID": func(postInterface interface{}) (thread int) {
		post, ok := postInterface.(Post)
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
			postURL = config.SiteDomain
		}
		postURL += config.SiteWebfolder

		if typeOf == "recent" {
			post, ok := postInterface.(*RecentPost)
			if !ok {
				return
			}
			postURL = post.GetURL(withDomain)
		} else {
			post, ok := postInterface.(*Post)
			if !ok {
				return
			}
			postURL = post.GetURL(withDomain)
		}
		return
	},
	"getThreadThumbnail": func(img string) string {
		return getThumbnailPath("thread", img)
	},
	"getUploadType": func(name string) string {
		extension := getFileExtension(name)
		var uploadType string
		switch extension {
		case "":
		case "deleted":
			uploadType = ""
		case "webm":
		case "jpg":
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
	"isBanned":   isBanned,
	"numReplies": numReplies,
	"getBoardDir": func(id int) string {
		board, err := getBoardFromID(id)
		if err != nil {
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
		configType := reflect.TypeOf(config)
		tableOut := "<table style=\"border-collapse: collapse;\" id=\"config\"><tr><th>Field name</th><th>Value</th><th>Type</th><th>Description</th></tr>\n"
		numFields := configType.NumField()
		for f := 17; f < numFields-2; f++ {
			// starting at Lockdown because the earlier fields can't be safely edited from a web interface
			field := configType.Field(f)
			if field.Tag.Get("critical") != "" {
				continue
			}
			name := field.Name
			tableOut += "<tr><th>" + name + "</th><td>"
			f := reflect.Indirect(reflect.ValueOf(config)).FieldByName(name)

			kind := f.Kind()
			switch kind {
			case reflect.Int:
				tableOut += "<input name=\"" + name + "\" type=\"number\" value=\"" + html.EscapeString(fmt.Sprintf("%v", f)) + "\" class=\"config-text\"/>"
			case reflect.String:
				tableOut += "<input name=\"" + name + "\" type=\"text\" value=\"" + html.EscapeString(fmt.Sprintf("%v", f)) + "\" class=\"config-text\"/>"
			case reflect.Bool:
				checked := ""
				if f.Bool() {
					checked = "checked"
				}
				tableOut += "<input name=\"" + name + "\" type=\"checkbox\" " + checked + " />"
			case reflect.Slice:
				tableOut += "<textarea name=\"" + name + "\" rows=\"4\" cols=\"28\">"
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
			tableOut += "</tr>\n"
		}
		tableOut += "</table>\n"
		return tableOut
	},
	"isStyleDefault": func(style string) bool {
		return style == config.DefaultStyle
	},

	// Version functions
	"version": func() string {
		return version.String()
	},
}

var (
	banpageTmpl      *template.Template
	captchaTmpl      *template.Template
	catalogTmpl      *template.Template
	errorpageTmpl    *template.Template
	frontPageTmpl    *template.Template
	boardpageTmpl    *template.Template
	threadpageTmpl   *template.Template
	postEditTmpl     *template.Template
	manageBansTmpl   *template.Template
	manageBoardsTmpl *template.Template
	manageConfigTmpl *template.Template
	manageHeaderTmpl *template.Template
	jsTmpl           *template.Template
)

func loadTemplate(files ...string) (*template.Template, error) {
	var templates []string
	for i, file := range files {
		templates = append(templates, file)
		tmplPath := path.Join(config.TemplateDir, "override", file)

		if _, err := os.Stat(tmplPath); !os.IsNotExist(err) {
			files[i] = tmplPath
		} else {
			files[i] = path.Join(config.TemplateDir, file)
		}
	}

	return template.New(templates[0]).Funcs(funcMap).ParseFiles(files...)
}

func templateError(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("Failed loading template '%s/%s': %s", config.TemplateDir, name, err.Error())
}

func initTemplates(which ...string) error {
	var err error
	buildAll := len(which) == 0 || which[0] == "all"
	resetBoardSectionArrays()
	for _, t := range which {
		if buildAll || t == "banpage" {
			banpageTmpl, err = loadTemplate("banpage.html", "page_footer.html")
			if err != nil {
				return templateError("banpage.html", err)
			}
		}
		if buildAll || t == "captcha" {
			captchaTmpl, err = loadTemplate("captcha.html")
			if err != nil {
				return templateError("captcha.html", err)
			}
		}
		if buildAll || t == "catalog" {
			catalogTmpl, err = loadTemplate("catalog.html", "page_header.html", "page_footer.html")
			if err != nil {
				return templateError("catalog.html", err)
			}
		}
		if buildAll || t == "error" {
			errorpageTmpl, err = loadTemplate("error.html")
			if err != nil {
				return templateError("error.html", err)
			}
		}
		if buildAll || t == "front" {
			frontPageTmpl, err = loadTemplate("front.html", "front_intro.html", "page_header.html", "page_footer.html")
			if err != nil {
				return templateError("front.html", err)
			}
		}
		if buildAll || t == "boardpage" {
			boardpageTmpl, err = loadTemplate("boardpage.html", "page_header.html", "postbox.html", "page_footer.html")
			if err != nil {
				return templateError("boardpage.html", err)
			}
		}
		if buildAll || t == "threadpage" {
			threadpageTmpl, err = loadTemplate("threadpage.html", "page_header.html", "postbox.html", "page_footer.html")
			if err != nil {
				return templateError("threadpage.html", err)
			}
		}
		if buildAll || t == "postedit" {
			postEditTmpl, err = loadTemplate("post_edit.html", "page_header.html", "page_footer.html")
			if err != nil {
				return templateError("threadpage.html", err)
			}
		}
		if buildAll || t == "managebans" {
			manageBansTmpl, err = loadTemplate("manage_bans.html")
			if err != nil {
				return templateError("manage_bans.html", err)
			}
		}
		if buildAll || t == "manageboards" {
			manageBoardsTmpl, err = loadTemplate("manage_boards.html")
			if err != nil {
				return templateError("manage_boards.html", err)
			}
		}
		if buildAll || t == "manageconfig" {
			manageConfigTmpl, err = loadTemplate("manage_config.html")
			if err != nil {
				return templateError("manage_config.html", err)
			}
		}
		if buildAll || t == "manageheader" {
			manageHeaderTmpl, err = loadTemplate("manage_header.html")
			if err != nil {
				return templateError("manage_header.html", err)
			}
		}
		if buildAll || t == "js" {
			jsTmpl, err = loadTemplate("consts.js")
			if err != nil {
				return templateError("consts.js", err)
			}
		}
	}
	return nil
}

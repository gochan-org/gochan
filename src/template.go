package main

import (
	"errors"
	"fmt"
	"html"
	"reflect"
	"strconv"
	"strings"
	"text/template"
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
	"getSlice": func(arr []interface{}, start, end int) []interface{} {
		slice := arr[start:end]
		defer func() {
			if r := recover(); r != nil {
				slice = make([]interface{}, 1)
			}
		}()
		return slice
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
	"formatFilesize": func(size_int int) string {
		size := float32(size_int)
		if size < 1000 {
			return fmt.Sprintf("%d B", size_int)
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
	"truncateMessage": func(msg string, limit int, max_lines int) string {
		var truncated bool
		split := strings.SplitN(msg, "<br />", -1)

		if len(split) > max_lines {
			split = split[:max_lines]
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
	"getThreadID": func(post_i interface{}) (thread int) {
		post := post_i.(PostTable)
		if post.ParentID == 0 {
			thread = post.ID
		} else {
			thread = post.ParentID
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
		tableOut := "<table style=\"border-collapse: collapse;\"><tr><th>Field name</th><th>Value</th><th>Type</th><th>Description</th></tr>\n"
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
}

var (
	banpage_tmpl        *template.Template
	catalog_tmpl        *template.Template
	errorpage_tmpl      *template.Template
	front_page_tmpl     *template.Template
	img_boardpage_tmpl  *template.Template
	img_threadpage_tmpl *template.Template
	img_post_form_tmpl  *template.Template
	manage_bans_tmpl    *template.Template
	manage_boards_tmpl  *template.Template
	manage_config_tmpl  *template.Template
	manage_header_tmpl  *template.Template
	post_edit_tmpl      *template.Template
)

func loadTemplate(files ...string) (*template.Template, error) {
	if len(files) == 0 {
		return nil, errors.New("No files named in call to loadTemplate")
	}
	var templates []string
	for i, file := range files {
		templates = append(templates, file)
		files[i] = config.TemplateDir + "/" + files[i]
	}

	return template.New(templates[0]).Funcs(funcMap).ParseFiles(files...)
}

func templateError(name string, err error) error {
	return errors.New("Failed loading template \"" + config.TemplateDir + "/" + name + ": \"" + err.Error())
}

func initTemplates() error {
	var err error
	resetBoardSectionArrays()
	banpage_tmpl, err = loadTemplate("banpage.html", "global_footer.html")
	if err != nil {
		return templateError("banpage.html", err)
	}

	catalog_tmpl, err = loadTemplate("catalog.html", "img_header.html", "global_footer.html")
	if err != nil {
		return templateError("catalog.html", err)
	}

	errorpage_tmpl, err = loadTemplate("error.html")
	if err != nil {
		return templateError("error.html", err)
	}

	img_boardpage_tmpl, err = loadTemplate("img_boardpage.html", "img_header.html", "postbox.html", "global_footer.html")
	if err != nil {
		return templateError("img_boardpage.html", err)
	}

	img_threadpage_tmpl, err = loadTemplate("img_threadpage.html", "img_header.html", "postbox.html", "global_footer.html")
	if err != nil {
		return templateError("img_threadpage.html", err)
	}

	post_edit_tmpl, err = loadTemplate("post_edit.html", "img_header.html", "global_footer.html")
	if err != nil {
		return templateError("img_threadpage.html", err)
	}

	manage_bans_tmpl, err = loadTemplate("manage_bans.html")
	if err != nil {
		return templateError("manage_bans.html", err)
	}

	manage_boards_tmpl, err = loadTemplate("manage_boards.html")
	if err != nil {
		return templateError("manage_boards.html", err)
	}

	manage_config_tmpl, err = loadTemplate("manage_config.html")
	if err != nil {
		return templateError("manage_config.html", err)
	}

	manage_header_tmpl, err = loadTemplate("manage_header.html")
	if err != nil {
		return templateError("manage_header.html", err)
	}

	front_page_tmpl, err = loadTemplate("front.html", "global_footer.html")
	if err != nil {
		return templateError("front.html", err)
	}
	return nil
}

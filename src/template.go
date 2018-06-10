package main

import (
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"
	"text/template"
)

var funcMap = template.FuncMap{
	"add": func(a, b int) int {
		return a + b
	},
	"subtract": func(a, b int) int {
		return a - b
	},
	"len": func(arr []interface{}) int {
		return len(arr)
	},
	"getSlice": func(arr []interface{}, start, end int) []interface{} {
		slice := arr[start:end]
		defer func() {
			if r := recover(); r != nil {
				slice = make([]interface{}, 1)
			}
		}()
		return slice
	},
	"gt": func(a int, b int) bool {
		return a > b
	},
	"ge": func(a int, b int) bool {
		return a >= b
	},
	"lt": func(a int, b int) bool {
		return a < b
	},
	"le": func(a int, b int) bool {
		return a <= b
	},
	"makeLoop": func(n int, offset int) []int {
		loopArr := make([]int, n)
		for i := range loopArr {
			loopArr[i] = i + offset
		}
		return loopArr
	},
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
	"escapeString": func(a string) string {
		return html.EscapeString(a)
	},
	"isNil": func(i interface{}) bool {
		return i == nil
	},
	"intEq": func(a, b int) bool {
		return a == b
	},
	"intToString": strconv.Itoa,
	"isStyleDefault_img": func(style string) bool {
		return style == config.DefaultStyle_img
	},
	"formatTimestamp": humanReadableTime,
	"getThreadID": func(post_i interface{}) (thread int) {
		post := post_i.(PostTable)
		if post.ParentID == 0 {
			thread = post.ID
		} else {
			thread = post.ParentID
		}
		return
	},
	"getThumbnailFilename": func(name string) string {
		if name == "" || name == "deleted" {
			return ""
		}

		if name[len(name)-3:] == "gif" {
			name = name[:len(name)-3] + "jpg"
		} else if name[len(name)-4:] == "webm" {
			name = name[:len(name)-4] + "jpg"
		}
		extBegin := strings.LastIndex(name, ".")
		newName := name[:extBegin] + "t." + getFileExtension(name)
		return newName
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
	"formatFilesize": func(size_int int) string {
		size := float32(size_int)
		if size < 1000 {
			return fmt.Sprintf("%fB", size)
		} else if size <= 100000 {
			//size = size * 0.2
			return fmt.Sprintf("%0.1f KB", size/1024)
		} else if size <= 100000000 {
			//size = size * 0.2
			return fmt.Sprintf("%0.2f MB", size/1024/1024)
		}
		return fmt.Sprintf("%0.2f GB", size/1024/1024/1024)
	},
	"imageToThumbnailPath": func(img string) string {
		filetype := strings.ToLower(img[strings.LastIndex(img, ".")+1:])
		if filetype == "gif" || filetype == "webm" {
			filetype = "jpg"
		}
		index := strings.LastIndex(img, ".")
		if index < 0 || index > len(img) {
			return ""
		}
		return img[0:index] + "t." + filetype
	},
}

var (
	banpage_tmpl        *template.Template
	errorpage_tmpl      *template.Template
	front_page_tmpl     *template.Template
	global_header_tmpl  *template.Template
	img_boardpage_tmpl  *template.Template
	img_threadpage_tmpl *template.Template
	img_post_form_tmpl  *template.Template
	// manage_bans_tmpl    *template.Template
	manage_boards_tmpl *template.Template
	manage_config_tmpl *template.Template
	manage_header_tmpl *template.Template
	post_edit_tmpl     *template.Template
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
	banpage_tmpl, err = loadTemplate("banpage.html")
	if err != nil {
		return templateError("banpage.html", err)
	}

	errorpage_tmpl, err = loadTemplate("error.html")
	if err != nil {
		return templateError("error.html", err)
	}

	global_header_tmpl, err = loadTemplate("global_header.html")
	if err != nil {
		return templateError("global_header.html", err)
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

	/* manage_bans_tmpl, err = loadTemplate("manage_bans.html")
	if err != nil {
		return templateError("manage_bans.html", err)
	} */

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

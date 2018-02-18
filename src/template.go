package main

import (
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type FooterData struct {
	Version       string
	GeneratedTime float32
}

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
	"gte": func(a int, b int) bool {
		return a >= b
	},
	"lt": func(a int, b int) bool {
		return a < b
	},
	"lte": func(a int, b int) bool {
		return a <= b
	},
	"makeLoop": func(n int) []struct{} {
		return make([]struct{}, n)
	},
	"stringAppend": func(a, b string) string {
		return a + b
	},
	"stringEq": func(a, b string) bool {
		return a == b
	},
	"stringNeq": func(a, b string) bool {
		return a != b
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
			} else {
				return msg[:limit]
			}
		} else {
			return msg
		}
	},
	"escapeString": func(a string) string {
		return html.EscapeString(a)
	},
	"intEq": func(a, b int) bool {
		return a == b
	},
	"intToString": func(a int) string {
		return strconv.Itoa(a)
	},
	"isStyleDefault_img": func(style string) bool {
		return style == config.DefaultStyle_img
	},
	"isStyleNotDefault_img": func(style string) bool {
		return style != config.DefaultStyle_img
	},
	"getElement": func(in []interface{}, element int) interface{} {
		if len(in) > element {
			return in[element]
		}
		return nil
	},
	"getInterface": func(in []interface{}, index int) interface{} {
		var nope interface{}
		if len(in) == 0 {
			return nope
		} else if len(in) < index+1 {
			return nope
		}
		return in[index]
	},
	"formatTimestamp": func(timestamp time.Time) string {
		return humanReadableTime(timestamp)
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
	"getThumbnailFilename": func(name string) string {
		if name == "" || name == "deleted" {
			return ""
		}

		if name[len(name)-3:] == "gif" {
			name = name[:len(name)-3] + "jpg"
		} else if name[len(name)-4:] == "webm" {
			name = name[:len(name)-4] + "jpg"
		}
		ext_begin := strings.LastIndex(name, ".")
		new_name := name[:ext_begin] + "t." + getFileExtension(name)
		return new_name
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
	footer_data = FooterData{version, float32(0)}

	banpage_tmpl        *template.Template
	global_footer_tmpl  *template.Template
	global_header_tmpl  *template.Template
	img_header_tmpl     *template.Template
	img_boardpage_tmpl  *template.Template
	img_threadpage_tmpl *template.Template
	img_post_form_tmpl  *template.Template
	manage_header_tmpl  *template.Template
	manage_boards_tmpl  *template.Template
	manage_config_tmpl  *template.Template
	front_page_tmpl     *template.Template
)

func loadTemplate(name string, filename string, before string) (*template.Template, error) {
	tmplBytes, err := ioutil.ReadFile(config.TemplateDir + "/" + filename)
	if err != nil {
		return nil, err
	}
	tmplStr := before + string(tmplBytes)
	return template.New(name).Funcs(funcMap).Parse(tmplStr)
}

func initTemplates() {
	var err error
	resetBoardSectionArrays()

	banpage_tmpl, err = loadTemplate("banpage_tmpl", "banpage.html",
		"{{$config := getInterface .Data 0}}"+
			"{{$ban := getInterface .Data 1}}")
	if err != nil {
		println(0, "Failed loading template \""+config.TemplateDir+"/banpage.html: \""+err.Error())
		os.Exit(2)
	}

	global_footer_tmpl, err = loadTemplate("global_footer_tmpl", "global_footer.html", "{{$config := getInterface .Data 0}}")
	if err != nil {
		println(0, "Failed loading template \""+config.TemplateDir+"/global_footer.html: \""+err.Error())
		os.Exit(2)
	}

	global_header_tmpl, err = loadTemplate("global_header_tmpl", "global_header.html", "")
	if err != nil {
		println(0, "Failed loading template \""+config.TemplateDir+"/global_header.html: \""+err.Error())
		os.Exit(2)
	}

	img_header_tmpl, err = loadTemplate("img_header_tmpl", "img_header.html",
		"{{$config := getInterface .Data 0}}"+
			"{{$board_arr := (getInterface .Data 1).Data}}"+
			"{{$section_arr := (getInterface .Data 2).Data}}"+
			"{{$post_arr := (getInterface .Data 3).Data}}"+
			"{{$op := getElement $post_arr 0}}"+
			"{{$board := getElement $board_arr (subtract $op.BoardID 1)}}")
	if err != nil {
		println(0, "Failed loading template \""+config.TemplateDir+"/img_header.html: \""+err.Error())
		os.Exit(2)
	}

	img_boardpage_tmpl, err = loadTemplate("img_boardpage_tmpl", "img_boardpage.html",
		"{{$config := getInterface .Data 0}}"+
			"{{$board_arr := (getInterface .Data 1).Data}}"+
			"{{$section_arr := (getInterface .Data 2).Data}}"+
			"{{$thread_arr := (getInterface .Data 3).Data}}"+
			"{{$board_info := (getInterface .Data 4).Data}}"+
			"{{$board := getInterface $board_info 0}}")
	if err != nil {
		println(0, "Failed loading template \""+config.TemplateDir+"/img_boardpage.html: \""+err.Error())
		os.Exit(2)
	}

	img_threadpage_tmpl, err = loadTemplate("img_threadpage_tmpl", "img_threadpage.html",
		"{{$config := getInterface .Data 0}}"+
			"{{$board_arr := (getInterface .Data 1).Data}}"+
			"{{$section_arr := (getInterface .Data 2).Data}}"+
			"{{$post_arr := (getInterface .Data 3).Data}}"+
			"{{$op := getElement $post_arr 0}}"+
			"{{$board := getElement $board_arr (subtract $op.BoardID 1)}}")
	if err != nil {
		println(0, "Failed loading template \""+config.TemplateDir+"/img_threadpage.html: \""+err.Error())
		os.Exit(2)
	}

	manage_header_tmpl, err = loadTemplate("manage_header_tmpl", "manage_header.html", "")
	if err != nil {
		println(0, "Failed loading template \""+config.TemplateDir+"/manage_header.html: \""+err.Error())
		os.Exit(2)
	}

	manage_boards_tmpl, err = loadTemplate("manage_boards_tmpl", "manage_boards.html",
		"{{$config := getInterface .Data 0}}"+
			"{{$board := getInterface (getInterface .Data 1).Data 0}}"+
			"{{$section_arr := (getInterface .Data 2).Data}}")
	if err != nil {
		println(0, "Failed loading template \""+config.TemplateDir+"/manage_boards.html: \""+err.Error())
		os.Exit(2)
	}

	manage_config_tmpl, err = loadTemplate("manage_config_tmpl", "manage_config.html", "{{$config := getInterface .Data 0}}")
	if err != nil {
		println(0, "Failed loading template \""+config.TemplateDir+"/manage_config.html: \""+err.Error())
		os.Exit(2)
	}

	front_page_tmpl, err = loadTemplate("front_page_tmpl", "front.html",
		"{{$config := getInterface .Data 0}}"+
			"{{$page_arr := getInterface .Data 1}}"+
			"{{$board_arr := getInterface .Data 2}}"+
			"{{$section_arr := getInterface .Data 3}}"+
			"{{$recent_posts_arr := getInterface .Data 4}}")
	if err != nil {
		println(0, "Failed loading template \""+config.TemplateDir+"/front.html\": "+err.Error())
		os.Exit(2)
	}
}

func getStyleLinks(w http.ResponseWriter, stylesheet string) {
	styles_map := make(map[int]string)
	for i := 0; i < len(config.Styles_img); i++ {
		styles_map[i] = config.Styles_img[i]
	}

	if err := manage_header_tmpl.Execute(w, config); err != nil {
		println(0, err.Error())
		os.Exit(2)
	}
}

func renderTemplate(tmpl *template.Template, name string, output io.Writer, wrappers ...*Wrapper) error {
	var interfaces []interface{}
	interfaces = append(interfaces, config)

	for _, wrapper := range wrappers {
		interfaces = append(interfaces, wrapper)
	}
	wrapped := &Wrapper{IName: name, Data: interfaces}
	return tmpl.Execute(output, wrapped)
}

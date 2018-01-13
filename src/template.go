package main

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
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
	"unescapeString": func(a string) string {
		return html.UnescapeString(a)
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
		return in[element]
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
		if name[len(name)-3:] == "gif" || name[len(name)-3:] == "gif" {
			name = name[:len(name)-3] + "jpg"
		}
		ext_begin := strings.LastIndex(name, ".")
		new_name := name[:ext_begin] + "t." + getFileExtension(name)
		return new_name
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
		filetype := img[strings.LastIndex(img, ".")+1:]
		if filetype == "gif" || filetype == "GIF" {
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

	banpage_tmpl_str string
	banpage_tmpl     *template.Template

	global_footer_tmpl_str string
	global_footer_tmpl     *template.Template

	global_header_tmpl_str string
	global_header_tmpl     *template.Template

	img_boardpage_tmpl_str string
	img_boardpage_tmpl     *template.Template

	img_threadpage_tmpl_str string
	img_threadpage_tmpl     *template.Template

	manage_header_tmpl_str string
	manage_header_tmpl     *template.Template

	manage_boards_tmpl_str string
	manage_boards_tmpl     *template.Template

	front_page_tmpl_str string
	front_page_tmpl     *template.Template

	template_buffer bytes.Buffer
	starting_time   int
)

func initTemplates() {
	resetBoardSectionArrays()
	banpage_tmpl_bytes, tmpl_err := ioutil.ReadFile(config.TemplateDir + "/banpage.html")
	if tmpl_err != nil {
		fmt.Println("Failed loading template \"" + config.TemplateDir + "/banpage.html\": " + tmpl_err.Error())
		os.Exit(2)
	}
	banpage_tmpl_str = "{{$config := getInterface .Data 0}}" +
		"{{$ban := getInterface .Data 1}}" +
		string(banpage_tmpl_bytes)
	banpage_tmpl, tmpl_err = template.New("banpage_tmpl").Funcs(funcMap).Parse(string(banpage_tmpl_str))
	if tmpl_err != nil {
		fmt.Println("Failed loading template \"" + config.TemplateDir + "/banpage.html\": " + tmpl_err.Error())
		os.Exit(2)
	}

	global_footer_tmpl_bytes, tmpl_err := ioutil.ReadFile(config.TemplateDir + "/global_footer.html")
	if tmpl_err != nil {
		fmt.Println("Failed loading template \"" + config.TemplateDir + "/global_footer.html\": " + tmpl_err.Error())
		os.Exit(2)
	}
	global_footer_tmpl_str = string(global_footer_tmpl_bytes)
	global_footer_tmpl, tmpl_err = template.New("global_footer_tmpl").Funcs(funcMap).Parse(string(global_footer_tmpl_str))
	if tmpl_err != nil {
		fmt.Println("Failed loading template \"" + config.TemplateDir + "/global_footer.html\": " + tmpl_err.Error())
		os.Exit(2)
	}

	global_header_tmpl_bytes, tmpl_err := ioutil.ReadFile(config.TemplateDir + "/global_header.html")
	if tmpl_err != nil {
		fmt.Println("Failed loading template \"" + config.TemplateDir + "/global_header.html\": " + tmpl_err.Error())
		os.Exit(2)
	}
	global_header_tmpl_str = string(global_header_tmpl_bytes)
	global_header_tmpl, tmpl_err = template.New("global_header_tmpl").Funcs(funcMap).Parse(string(global_header_tmpl_str))
	if tmpl_err != nil {
		fmt.Println("Failed loading template \"" + config.TemplateDir + "/global_header.html\": " + tmpl_err.Error())
		os.Exit(2)
	}

	img_boardpage_tmpl_bytes, _ := ioutil.ReadFile(path.Join(config.TemplateDir, "img_boardpage.html"))
	if tmpl_err != nil {
		fmt.Println("Failed loading template \"" + config.TemplateDir + "/img_boardpage.html\": " + tmpl_err.Error())
		os.Exit(2)
	}
	img_boardpage_tmpl_str = "{{$config := getInterface .Data 0}}" +
		"{{$board_arr := (getInterface .Data 1).Data}}" +
		"{{$section_arr := (getInterface .Data 2).Data}}" +
		"{{$thread_arr := (getInterface .Data 3).Data}}" +
		"{{$board_info := (getInterface .Data 4).Data}}" +
		"{{$board := getInterface $board_info 0}}" +
		string(img_boardpage_tmpl_bytes)
	img_boardpage_tmpl, tmpl_err = template.New("img_boardpage_tmpl").Funcs(funcMap).Parse(img_boardpage_tmpl_str)
	if tmpl_err != nil {
		fmt.Println("Failed loading template \"" + config.TemplateDir + "/img_boardpage.html: \"" + tmpl_err.Error())
		os.Exit(2)
	}

	img_threadpage_tmpl_bytes, _ := ioutil.ReadFile(path.Join(config.TemplateDir, "img_threadpage.html"))
	if tmpl_err != nil {
		fmt.Println("Failed loading template \"" + config.TemplateDir + "/img_threadpage.html\": " + tmpl_err.Error())
		os.Exit(2)
	}
	img_threadpage_tmpl_str = "{{$config := getInterface .Data 0}}" +
		"{{$board_arr := (getInterface .Data 1).Data}}" +
		"{{$section_arr := (getInterface .Data 2).Data}}" +
		"{{$post_arr := (getInterface .Data 3).Data}}" +
		"{{$op := getElement $post_arr 0}}" +
		"{{$board := getElement $board_arr (subtract $op.BoardID 1)}}" +
		string(img_threadpage_tmpl_bytes)
	img_threadpage_tmpl, tmpl_err = template.New("img_threadpage_tmpl").Funcs(funcMap).Parse(img_threadpage_tmpl_str)
	if tmpl_err != nil {
		fmt.Println("Failed loading template \"" + config.TemplateDir + "/img_threadpage.html: \"" + tmpl_err.Error())
		os.Exit(2)
	}

	manage_header_tmpl_bytes, err := ioutil.ReadFile(config.TemplateDir + "/manage_header.html")
	if err != nil {
		fmt.Println(err.Error())
	}
	manage_header_tmpl_str = string(manage_header_tmpl_bytes)
	manage_header_tmpl, tmpl_err = template.New("manage_header_tmpl").Funcs(funcMap).Parse(manage_header_tmpl_str)
	if tmpl_err != nil {
		fmt.Println("Failed loading template \"" + config.TemplateDir + "/manage_header.html\": " + tmpl_err.Error())
		os.Exit(2)
	}

	manage_boards_tmpl_bytes, err := ioutil.ReadFile(config.TemplateDir + "/manage_boards.html")
	if err != nil {
		fmt.Println(err.Error())
	}
	manage_boards_tmpl_str = "{{$config := getInterface .Data 0}}" +
		"{{$board := getInterface (getInterface .Data 1).Data 0}}" +
		"{{$section_arr := (getInterface .Data 2).Data}}" +
		string(manage_boards_tmpl_bytes)

	manage_boards_tmpl, tmpl_err = template.New("manage_boards_tmpl").Funcs(funcMap).Parse(manage_boards_tmpl_str)
	if tmpl_err != nil {
		fmt.Println("Failed loading template \"" + config.TemplateDir + "/manage_boards.html\": " + tmpl_err.Error())
		os.Exit(2)
	}

	front_page_tmpl_bytes, err := ioutil.ReadFile(config.TemplateDir + "/front.html")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
	front_page_tmpl_str = "{{$config := getInterface .Data 0}}\n" +
		"{{$page_arr := getInterface .Data 1}}\n" +
		"{{$board_arr := getInterface .Data 2}}\n" +
		"{{$section_arr := getInterface .Data 3}}\n" +
		"{{$recent_posts_arr := getInterface .Data 4}}\n" +
		string(front_page_tmpl_bytes)
	front_page_tmpl, tmpl_err = template.New("front_page_tmpl").Funcs(funcMap).Parse(front_page_tmpl_str)
	if tmpl_err != nil {
		fmt.Println("Failed loading template \"" + config.TemplateDir + "/front.html\": " + tmpl_err.Error())
		os.Exit(2)
	}
}

func getTemplateAsString(templ template.Template) (string, error) {
	var buf bytes.Buffer
	err := templ.Execute(&buf, config)
	if err == nil {
		return buf.String(), nil
	}
	return "", err
}

func getStyleLinks(w http.ResponseWriter, stylesheet string) {
	styles_map := make(map[int]string)
	for i := 0; i < len(config.Styles_img); i++ {
		styles_map[i] = config.Styles_img[i]
	}

	err := manage_header_tmpl.Execute(w, config)
	if err != nil {
		fmt.Println(err.Error())
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

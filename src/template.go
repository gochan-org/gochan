package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"text/template"
	"time"
)



type FooterData struct {
	Version float32
	GeneratedTime float32
}



var funcMap = template.FuncMap{
	"add": func(a,b int) int {
		return a + b
	},
	"subtract": func(a,b int) int {
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
	"lt": func(a int, b int) bool {
		return a < b
	},
	"stringEq": func(a, b string) bool {
		return a == b
	},
	"stringNeq": func(a, b string) bool {
		return a != b
	},
	"intEq": func(a, b int) bool {
		return a == b
	},
	"isStyleDefault_img": func(style string) bool {
		return style == config.DefaultStyle_img
	},
	"isStyleNotDefault_img": func(style string) bool {
		return style != config.DefaultStyle_img
	},
	"getInterface":func(in []interface{}, index int) interface{} {
		return in[index]
	},
	"formatTimestamp": func(timestamp time.Time) string {
		return humanReadableTime(timestamp)
	},
	"getThumbnailFilename": func(name string) string {
		filetype := name[len(name)-4:]
		if filetype == ".gif" || filetype == ".GIF" {
			return name[0:len(name)-3]+"jpg"
		}
		return name
	},
	"formatFilesize": func(size_int int) string {
		size := float32(size_int)
		if(size < 1000) {
			return fmt.Sprintf("%fB", size)
		} else if(size <= 100000) {
			//size = size * 0.2
			return fmt.Sprintf("%0.1f KB", size/1024)
		} else if(size <= 100000000) {
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
		return img[0:index]+"t."+filetype

	},
}

var (
	footer_data = FooterData{version, float32(0)}
	global_footer_tmpl_str string
	global_footer_tmpl *template.Template
	
	global_header_tmpl_str string
	global_header_tmpl *template.Template

	img_boardpage_tmpl_str string
	img_boardpage_tmpl *template.Template
	
	img_thread_tmpl_str string
	img_thread_tmpl *template.Template
	
	manage_header_tmpl_str string
	manage_header_tmpl *template.Template

	front_page_tmpl_str string
	front_page_tmpl *template.Template

	template_buffer bytes.Buffer
	starting_time int
)

func initTemplates() {
	global_footer_tmpl_bytes,tmpl_err := ioutil.ReadFile(config.TemplateDir+"/global_footer.html")
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/global_footer.html\": " + tmpl_err.Error())
		os.Exit(2)
	}
	global_footer_tmpl_str = string(global_footer_tmpl_bytes)
	global_footer_tmpl,tmpl_err = template.New("global_footer_tmpl").Funcs(funcMap).Parse(string(global_footer_tmpl_str))
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/global_footer.html\": " + tmpl_err.Error())
		os.Exit(2)
	}
	
	global_header_tmpl_bytes,tmpl_err := ioutil.ReadFile(config.TemplateDir+"/global_header.html")
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/global_header.html\": " + tmpl_err.Error())
		os.Exit(2)
	}
	global_header_tmpl_str = string(global_header_tmpl_bytes)
	global_header_tmpl,tmpl_err = template.New("global_header_tmpl").Funcs(funcMap).Parse(string(global_header_tmpl_str))
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/global_header.html\": " + tmpl_err.Error())
		os.Exit(2)
	}

	img_boardpage_tmpl_bytes,_ := ioutil.ReadFile(path.Join(config.TemplateDir,"img_boardpage.html"))
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/img_boardpage.html\": " + tmpl_err.Error())
		os.Exit(2)
	}
	img_boardpage_tmpl_str = "{{$config := getInterface .Data 0}}{{$thread_arr := getInterface .Data 1}}{{$post_arr := getInterface .Data 2}}{{$board_arr := getInterface .Data 3}}{{$section_arr := getInterface .Data 4}}{{$op := getInterface $post_arr 0}}{{$boardid := subtract $op.BoardID 1}}{{$board := getInterface $board_arr.Data $boardid}}" + string(img_boardpage_tmpl_bytes)
	img_boardpage_tmpl,tmpl_err = template.New("img_boardpage_tmpl").Funcs(funcMap).Parse(img_boardpage_tmpl_str)
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/img_boardpage.html: \"" + tmpl_err.Error())
		os.Exit(2)
	}

	img_thread_tmpl_bytes,_ := ioutil.ReadFile(path.Join(config.TemplateDir,"img_thread.html"))
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/img_thread.html\": " + tmpl_err.Error())
		os.Exit(2)
	}
	img_thread_tmpl_str = "{{$config := getInterface .Data 0}}{{$post_arr := getInterface .Data 1}}{{$board_arr := getInterface .Data 2}}{{$section_arr := getInterface .Data 3}}{{$op := getInterface $post_arr 0}}{{$boardid := subtract $op.BoardID 1}}{{$board := getInterface $board_arr.Data $boardid}}" + string(img_thread_tmpl_bytes)
	img_thread_tmpl,tmpl_err = template.New("img_thread_tmpl").Funcs(funcMap).Parse(img_thread_tmpl_str)
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/img_thread.html: \"" + tmpl_err.Error())
		os.Exit(2)
	}

	manage_header_tmpl_bytes,err := ioutil.ReadFile(config.TemplateDir+"/manage_header.html")
	if err != nil {
		fmt.Println(err.Error())
	}
	manage_header_tmpl_str = string(manage_header_tmpl_bytes)
	manage_header_tmpl,tmpl_err = template.New("manage_header_tmpl").Funcs(funcMap).Parse(manage_header_tmpl_str)
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/manage_header.html\": "+tmpl_err.Error())
		os.Exit(2)
	}

	front_page_tmpl_bytes,err := ioutil.ReadFile(config.TemplateDir+"/front.html")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
	front_page_tmpl_str = "{{$config := getInterface .Data 0}}{{$page_arr := getInterface .Data 1}}{{$board_arr := getInterface .Data 2}}{{$section_arr := getInterface .Data 3}}" + string(front_page_tmpl_bytes)
	front_page_tmpl,tmpl_err = template.New("front_page_tmpl").Funcs(funcMap).Parse(front_page_tmpl_str)
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/front.html\": "+tmpl_err.Error())
		os.Exit(2)
	}
}

func getTemplateAsString(templ template.Template) (string,error) {
	var buf bytes.Buffer
	err := templ.Execute(&buf,config)
	if err == nil {
		return buf.String(),nil
	}
	return "",err
}

func getStyleLinks(w http.ResponseWriter, stylesheet string) {
	styles_map := make(map[int]string)
	for i := 0; i < len(config.Styles_img); i++ {
		styles_map[i] = config.Styles_img[i]
	}

	err := manage_header_tmpl.Execute(w,config)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
}

func buildAll() error {
	buildFrontPage()
	/*
  	results,err := db.Query("SELECT `dir` FROM `"+config.DBprefix+"boards")
	var entry BoardTable
	for results.Next() {
		err = results.Scan(&entry.dir)
		buildBoard(entry.dir)
	}
	*/
	return nil
}

func buildFrontPage() error {
	return nil
}

func buildBoard(dir string) error {
	//build board pages
	//build board thread pages
	return nil
}
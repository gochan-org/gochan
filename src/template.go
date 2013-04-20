package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"text/template"
)



type FooterData struct {
	Version float32
	GeneratedTime float32
}

var funcMap = template.FuncMap{
	"isStyleDefault_img": func(style string) bool {
		return style == config.DefaultStyle_img
	},
	"isStyleNotDefault_img": func(style string) bool {
		return style != config.DefaultStyle_img
	},
}

var (
	footer_data = FooterData{version, float32(0)}
	global_footer_tmpl_str string
	global_footer_tmpl *template.Template
	
	global_header_tmpl_str string
	global_header_tmpl *template.Template
	
	img_header_tmpl_str string
	img_header_tmpl *template.Template
	
	manage_header_tmpl_str string
	manage_header_tmpl *template.Template
	
	template_buffer bytes.Buffer
)

func initTemplates() {
	global_footer_tmpl_bytes,tmpl_err := ioutil.ReadFile(config.TemplateDir+"/global_footer.html")
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/global_footer.html\"")
		os.Exit(2)
	}
	global_footer_tmpl_str = string(global_footer_tmpl_bytes)
	global_footer_tmpl,tmpl_err = template.New("global_footer_tmpl").Funcs(funcMap).Parse(string(global_footer_tmpl_str))
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/global_footer.html\"")
		os.Exit(2)
	}
	
	global_header_tmpl_bytes,tmpl_err := ioutil.ReadFile(config.TemplateDir+"/global_header.html")
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/global_header.html\"")
		os.Exit(2)
	}
	global_header_tmpl_str = string(global_header_tmpl_bytes)
	global_header_tmpl,tmpl_err = template.New("global_header_tmpl").Funcs(funcMap).Parse(string(global_header_tmpl_str))
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/global_header.html\"")
		os.Exit(2)
	}

	img_header_tmpl_bytes,_ := ioutil.ReadFile(config.TemplateDir+"/img_header.html")
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/img_header.html\"")
		os.Exit(2)
	}
	img_header_tmpl_str = string(img_header_tmpl_bytes)
	img_header_tmpl,_ = template.New("img_header_tmpl").Funcs(funcMap).Parse(string(img_header_tmpl_str))
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/img_header.html\"")
		os.Exit(2)
	}

	manage_header_tmpl_bytes,_ := ioutil.ReadFile(config.TemplateDir+"/manage_header.html")
	manage_header_tmpl_str = string(manage_header_tmpl_bytes)
	manage_header_tmpl,_ = template.New("manage_header_tmpl").Funcs(funcMap).Parse(manage_header_tmpl_str)
	if tmpl_err != nil {
		fmt.Println("Failed loading template \""+config.TemplateDir+"/manage_header.html\"")
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
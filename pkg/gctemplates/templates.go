package gctemplates

import (
	"fmt"
	"html/template"
	"os"
	"path"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

var (
	Banpage      *template.Template
	Captcha      *template.Template
	Catalog      *template.Template
	ErrorPage    *template.Template
	FrontPage    *template.Template
	BoardPage    *template.Template
	JsConsts     *template.Template
	ManageBans   *template.Template
	ManageBoards *template.Template
	ManageConfig *template.Template
	ManageHeader *template.Template
	PostEdit     *template.Template
	ThreadPage   *template.Template
)

func loadTemplate(files ...string) (*template.Template, error) {
	var templates []string
	for i, file := range files {
		templates = append(templates, file)
		tmplPath := path.Join(config.Config.TemplateDir, "override", file)

		if _, err := os.Stat(tmplPath); !os.IsNotExist(err) {
			files[i] = tmplPath
		} else {
			files[i] = path.Join(config.Config.TemplateDir, file)
		}
	}

	return template.New(templates[0]).Funcs(funcMap).ParseFiles(files...)
}

func templateError(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("failed loading template '%s/%s': %s",
		config.Config.TemplateDir, name, err.Error())
}

// InitTemplates loads the given templates by name. If no parameters are given,
// or the first one is "all", all templates are (re)loaded
func InitTemplates(which ...string) error {
	gcsql.ResetBoardSectionArrays()
	if len(which) == 0 || which[0] == "all" {
		return templateLoading("", true)
	}
	for _, t := range which {
		err := templateLoading(t, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func templateLoading(t string, buildAll bool) error {
	var err error
	if buildAll || t == "banpage" {
		Banpage, err = loadTemplate("banpage.html", "page_footer.html")
		if err != nil {
			return templateError("banpage.html", err)
		}
	}
	if buildAll || t == "captcha" {
		Captcha, err = loadTemplate("captcha.html")
		if err != nil {
			return templateError("captcha.html", err)
		}
	}
	if buildAll || t == "catalog" {
		Catalog, err = loadTemplate("catalog.html", "page_header.html", "page_footer.html")
		if err != nil {
			return templateError("catalog.html", err)
		}
	}
	if buildAll || t == "error" {
		ErrorPage, err = loadTemplate("error.html")
		if err != nil {
			return templateError("error.html", err)
		}
	}
	if buildAll || t == "front" {
		FrontPage, err = loadTemplate("front.html", "front_intro.html", "page_header.html", "page_footer.html")
		if err != nil {
			return templateError("front.html", err)
		}
	}
	if buildAll || t == "boardpage" {
		BoardPage, err = loadTemplate("boardpage.html", "page_header.html", "postbox.html", "page_footer.html")
		if err != nil {
			return templateError("boardpage.html", err)
		}
	}
	if buildAll || t == "threadpage" {
		ThreadPage, err = loadTemplate("threadpage.html", "page_header.html", "postbox.html", "page_footer.html")
		if err != nil {
			return templateError("threadpage.html", err)
		}
	}
	if buildAll || t == "postedit" {
		PostEdit, err = loadTemplate("post_edit.html", "page_header.html", "page_footer.html")
		if err != nil {
			return templateError("threadpage.html", err)
		}
	}
	if buildAll || t == "managebans" {
		ManageBans, err = loadTemplate("manage_bans.html")
		if err != nil {
			return templateError("manage_bans.html", err)
		}
	}
	if buildAll || t == "manageboards" {
		ManageBoards, err = loadTemplate("manage_boards.html")
		if err != nil {
			return templateError("manage_boards.html", err)
		}
	}
	if buildAll || t == "manageconfig" {
		ManageConfig, err = loadTemplate("manage_config.html")
		if err != nil {
			return templateError("manage_config.html", err)
		}
	}
	if buildAll || t == "manageheader" {
		ManageHeader, err = loadTemplate("manage_header.html")
		if err != nil {
			return templateError("manage_header.html", err)
		}
	}
	if buildAll || t == "js" {
		JsConsts, err = loadTemplate("consts.js")
		if err != nil {
			return templateError("consts.js", err)
		}
	}
	return nil
}

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
	Banpage           *template.Template
	Captcha           *template.Template
	Catalog           *template.Template
	ErrorPage         *template.Template
	FrontPage         *template.Template
	BoardPage         *template.Template
	JsConsts          *template.Template
	ManageBans        *template.Template
	ManageBoards      *template.Template
	ManageSections    *template.Template
	ManageConfig      *template.Template
	ManageDashboard   *template.Template
	ManageIPSearch    *template.Template
	ManageRecentPosts *template.Template
	ManageWordfilters *template.Template
	ManageLogin       *template.Template
	ManageReports     *template.Template
	ManageStaff       *template.Template
	PageHeader        *template.Template
	PageFooter        *template.Template
	PostEdit          *template.Template
	ThreadPage        *template.Template
)

func loadTemplate(files ...string) (*template.Template, error) {
	var templates []string
	templateDir := config.GetSystemCriticalConfig().TemplateDir
	for i, file := range files {
		templates = append(templates, file)
		tmplPath := path.Join(templateDir, "override", file)

		if _, err := os.Stat(tmplPath); !os.IsNotExist(err) {
			files[i] = tmplPath
		} else {
			files[i] = path.Join(templateDir, file)
		}
	}

	return template.New(templates[0]).Funcs(funcMap).ParseFiles(files...)
}

func templateError(name string, err error) error {
	if err == nil {
		return nil
	}
	templateDir := config.GetSystemCriticalConfig().TemplateDir

	return fmt.Errorf("failed loading template '%s/%s': %s",
		templateDir, name, err.Error())
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
		BoardPage, err = loadTemplate("boardpage.html", "post.html", "page_header.html", "postbox.html", "page_footer.html")
		if err != nil {
			return templateError("boardpage.html", err)
		}
	}
	if buildAll || t == "threadpage" {
		ThreadPage, err = loadTemplate("threadpage.html", "post.html", "page_header.html", "postbox.html", "page_footer.html")
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
	if buildAll || t == "managesections" {
		ManageSections, err = loadTemplate("manage_sections.html")
		if err != nil {
			return templateError("manage_sections.html", err)
		}
	}
	if buildAll || t == "manageconfig" {
		ManageConfig, err = loadTemplate("manage_config.html")
		if err != nil {
			return templateError("manage_config.html", err)
		}
	}
	if buildAll || t == "managedashboard" {
		ManageDashboard, err = loadTemplate("manage_dashboard.html")
		if err != nil {
			return templateError("manage_dashboard.html", err)
		}
	}
	if buildAll || t == "managelogin" {
		ManageLogin, err = loadTemplate("manage_login.html")
		if err != nil {
			return templateError("manage_login.html", err)
		}
	}
	if buildAll || t == "managereports" {
		ManageReports, err = loadTemplate("manage_reports.html")
		if err != nil {
			return templateError("manage_reports.html", err)
		}
	}
	if buildAll || t == "manageipsearch" {
		ManageIPSearch, err = loadTemplate("manage_ipsearch.html")
		if err != nil {
			return templateError("manage_ipsearch.html", err)
		}
	}
	if buildAll || t == "managerecents" {
		ManageRecentPosts, err = loadTemplate("manage_recentposts.html")
		if err != nil {
			return templateError("manage_recentposts.html", err)
		}
	}
	if buildAll || t == "managewordfilters" {
		ManageWordfilters, err = loadTemplate("manage_wordfilters.html")
		if err != nil {
			return templateError("manage_wordfilters.html", err)
		}
	}
	if buildAll || t == "managestaff" {
		ManageStaff, err = loadTemplate("manage_staff.html")
		if err != nil {
			return templateError("manage_staff.html", err)
		}
	}
	if buildAll || t == "pageheader" {
		PageHeader, err = loadTemplate("page_header.html")
		if err != nil {
			return templateError("page_header.html", err)
		}
	}
	if buildAll || t == "pagefooter" {
		PageFooter, err = loadTemplate("page_footer.html")
		if err != nil {
			return templateError("page_footer.html", err)
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

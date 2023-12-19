package gctemplates

import (
	"errors"
	"fmt"
	"html/template"
	"os"
	"path"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

const (
	BanPage             = "banpage.html"
	BoardPage           = "boardpage.html"
	Captcha             = "captcha.html"
	Catalog             = "catalog.html"
	JsConsts            = "consts.js"
	ErrorPage           = "error.html"
	FrontPage           = "front.html"
	ManageAnnouncements = "manage_announcements.html"
	ManageAppeals       = "manage_appeals.html"
	ManageBans          = "manage_bans.html"
	ManageBoards        = "manage_boards.html"
	ManageDashboard     = "manage_dashboard.html"
	ManageFileBans      = "manage_filebans.html"
	ManageFixThumbnails = "manage_fixthumbnails.html"
	ManageIPSearch      = "manage_ipsearch.html"
	ManageLogin         = "manage_login.html"
	ManageNameBans      = "manage_namebans.html"
	ManageRecentPosts   = "manage_recentposts.html"
	ManageReports       = "manage_reports.html"
	ManageSections      = "manage_sections.html"
	ManageStaff         = "manage_staff.html"
	ManageTemplates     = "manage_templateoverride.html"
	ManageThreadAttrs   = "manage_threadattrs.html"
	ManageViewLog       = "manage_viewlog.html"
	ManageWordfilters   = "manage_wordfilters.html"
	MoveThreadPage      = "movethreadpage.html"
	PageFooter          = "page_footer.html"
	PageHeader          = "page_header.html"
	PostEdit            = "post_edit.html"
	ThreadPage          = "threadpage.html"
)

var (
	ErrUnrecognizedTemplate = errors.New("unrecognized template")

	templateMap = map[string]*gochanTemplate{
		BanPage: {
			files: []string{"banpage.html", "page_footer.html"},
		},
		BoardPage: {
			files: []string{"boardpage.html", "topbar.html", "post.html", "page_header.html", "postbox.html", "page_footer.html"},
		},
		Captcha: {
			files: []string{"captcha.html"},
		},
		Catalog: {
			files: []string{"catalog.html", "topbar.html", "page_header.html", "page_footer.html"},
		},
		JsConsts: {
			files: []string{"consts.js"},
		},
		ErrorPage: {
			files: []string{"error.html"},
		},
		FrontPage: {
			files: []string{"front.html", "topbar.html", "front_intro.html", "page_header.html", "page_footer.html"},
		},
		ManageAnnouncements: {
			files: []string{"manage_announcements.html", "page_header.html", "topbar.html", "page_footer.html"},
		},
		ManageAppeals: {
			files: []string{"manage_appeals.html"},
		},
		ManageBans: {
			files: []string{"manage_bans.html"},
		},
		ManageBoards: {
			files: []string{"manage_boards.html"},
		},
		ManageDashboard: {
			files: []string{"manage_dashboard.html"},
		},
		ManageFileBans: {
			files: []string{"manage_filebans.html"},
		},
		ManageFixThumbnails: {
			files: []string{"manage_fixthumbnails.html"},
		},
		ManageIPSearch: {
			files: []string{"manage_ipsearch.html"},
		},
		ManageLogin: {
			files: []string{"manage_login.html"},
		},
		ManageNameBans: {
			files: []string{"manage_namebans.html"},
		},
		ManageRecentPosts: {
			files: []string{"manage_recentposts.html"},
		},
		ManageReports: {
			files: []string{"manage_reports.html"},
		},
		ManageSections: {
			files: []string{"manage_sections.html"},
		},
		ManageStaff: {
			files: []string{"manage_staff.html"},
		},
		ManageTemplates: {
			files: []string{"manage_templateoverride.html"},
		},
		ManageThreadAttrs: {
			files: []string{"manage_threadattrs.html"},
		},
		ManageViewLog: {
			files: []string{"manage_viewlog.html"},
		},
		ManageWordfilters: {
			files: []string{"manage_wordfilters.html"},
		},
		MoveThreadPage: {
			files: []string{"movethreadpage.html", "page_header.html", "topbar.html", "page_footer.html"},
		},
		PageFooter: {
			files: []string{"page_footer.html"},
		},
		PageHeader: {
			files: []string{"page_header.html", "topbar.html"},
		},
		PostEdit: {
			files: []string{"post_edit.html", "page_header.html", "topbar.html", "page_footer.html"},
		},
		ThreadPage: {
			files: []string{"threadpage.html", "topbar.html", "post.html", "page_header.html", "postbox.html", "page_footer.html"},
		},
	}
)

type gochanTemplate struct {
	files []string
	tmpl  *template.Template
}

func (gt *gochanTemplate) Load() (err error) {
	gt.tmpl, err = loadTemplate(gt.files...)
	return err
}

func (gt *gochanTemplate) Template() *template.Template {
	return gt.tmpl
}

func GetTemplate(name string) (*template.Template, error) {
	gctmpl, ok := templateMap[name]
	if !ok {
		fmt.Printf("Unrecognized template %q\n", name)
		return nil, ErrUnrecognizedTemplate
	}
	if gctmpl.tmpl != nil {
		return gctmpl.tmpl, nil
	}
	var err error
	gctmpl.tmpl, err = loadTemplate(gctmpl.files...)
	return gctmpl.tmpl, err
}

func loadTemplate(files ...string) (*template.Template, error) {
	var templates []string
	templateDir := config.GetSystemCriticalConfig().TemplateDir
	var foundFiles []string
	for i, file := range files {
		foundFiles = append(foundFiles, file)
		templates = append(templates, file)
		tmplPath := path.Join(templateDir, "override", file)

		if _, err := os.Stat(tmplPath); err == nil {
			foundFiles[i] = tmplPath
		} else if os.IsNotExist(err) {
			foundFiles[i] = path.Join(templateDir, file)
		} else {
			return nil, err
		}
	}

	tmpl, err := template.New(templates[0]).Funcs(funcMap).ParseFiles(foundFiles...)
	return tmpl, templateError(templates[0], err)
}

func ParseTemplate(name, tmplStr string) (*template.Template, error) {
	return template.New(name).Funcs(funcMap).Parse(tmplStr)
}

func templateError(name string, err error) error {
	if err == nil {
		return nil
	}
	templateDir := config.GetSystemCriticalConfig().TemplateDir

	return fmt.Errorf("failed loading template '%s: %s': %s",
		templateDir, name, err.Error())
}

// InitTemplates loads the given templates by name. If no parameters are given,
// all templates are (re)loaded
func InitTemplates(which ...string) error {
	err := gcsql.ResetBoardSectionArrays()
	if err != nil {
		return err
	}

	if which == nil {
		// no templates specified
		for t := range templateMap {
			if err = templateMap[t].Load(); err != nil {
				return err
			}
		}
	}

	for _, t := range which {
		if _, err = GetTemplate(t); err != nil {
			return err
		}
	}
	return nil
}

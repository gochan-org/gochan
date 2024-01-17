package gctemplates

import (
	"errors"
	"html/template"
	"os"
	"path"
	"sort"

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
	FrontIntro          = "front_intro.html"
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
	PostFlag            = "flag.html"
	ThreadPage          = "threadpage.html"
)

var (
	ErrUnrecognizedTemplate = errors.New("unrecognized template")

	templateMap = map[string]*gochanTemplate{
		BanPage: {
			files: []string{"banpage.html", "page_footer.html"},
		},
		BoardPage: {
			files: []string{"boardpage.html", "topbar.html", "post_flag.html", "post.html", "page_header.html", "postbox.html", "page_footer.html"},
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
		FrontIntro: {
			files: []string{"front_intro.html"},
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
		PostFlag: {
			files: []string{"post_flag.html"},
		},
		ThreadPage: {
			files: []string{"threadpage.html", "topbar.html", "post_flag.html", "post.html", "page_header.html", "postbox.html", "page_footer.html"},
		},
	}
)

type gochanTemplate struct {
	files      []string
	tmpl       *template.Template
	isOverride bool
	filePath   string
}

func (gt *gochanTemplate) Load() (err error) {
	templateDir := config.GetSystemCriticalConfig().TemplateDir

	var filePaths []string
	for _, file := range gt.files {

		if _, err = os.Stat(path.Join(templateDir, "override", file)); os.IsNotExist(err) {
			filePaths = append(filePaths, path.Join(templateDir, file))
		} else if err == nil {
			filePaths = append(filePaths, path.Join(templateDir, "override", file))
		} else {
			return err
		}
	}
	gt.filePath = filePaths[0]
	gt.tmpl, err = template.New(gt.files[0]).Funcs(funcMap).ParseFiles(filePaths...)
	return err
}

func (gt *gochanTemplate) Template() *template.Template {
	return gt.tmpl
}

// IsOverride returns true if the base file is overriden (i.e., it exist in the overrides subdirectory
// of the templates directory)
func (gt *gochanTemplate) IsOverride() bool {
	return gt.isOverride
}

// TemplatePath returns the path to the base template file
func (gt *gochanTemplate) TemplatePath() string {
	return gt.filePath
}

// GetTemplate takes the filename of the template and returns the template if it exists and
// is already loaded, and attempts to load and then return it if it exists but isn't loaded
func GetTemplate(name string) (*template.Template, error) {
	gctmpl, ok := templateMap[name]
	if !ok {
		return nil, ErrUnrecognizedTemplate
	}
	if gctmpl.tmpl != nil {
		return gctmpl.tmpl, nil
	}
	err := gctmpl.Load()
	return gctmpl.tmpl, err
}

func GetTemplatePath(name string) (string, error) {
	gctmpl, ok := templateMap[name]
	if !ok {
		return "", ErrUnrecognizedTemplate
	}
	return gctmpl.filePath, nil
}

// GetTemplateList returns a string array of all valid template filenames
func GetTemplateList() []string {
	var templateList []string
	for t := range templateMap {
		templateList = append(templateList, t)
	}
	sort.Strings(templateList)
	return templateList
}

func ParseTemplate(name, tmplStr string) (*template.Template, error) {
	return template.New(name).Funcs(funcMap).Parse(tmplStr)
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

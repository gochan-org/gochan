package templatetests_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/stretchr/testify/assert"
)

var (
	testingSiteConfig = &config.SiteConfig{
		SiteName:   "Gochan",
		SiteSlogan: "Gochan test",
	}
	simpleBoardConfig = &config.BoardConfig{
		DefaultStyle: "pipes.css",
		Styles: []config.Style{
			{Name: "Pipes", Filename: "pipes.css"},
			{Name: "Yotsuba A", Filename: "yotsuba.css"},
			{Name: "Yotsuba B", Filename: "yotsubab.css"},
		},
		Banners: []config.PageBanner{
			{Filename: "banner1.png", Width: 300, Height: 100},
			{Filename: "banner2.png", Width: 300, Height: 100},
			{Filename: "banner3.png", Width: 300, Height: 100},
		},
		EnableSpoileredImages:  true,
		EnableSpoileredThreads: true,
	}

	simpleBoard1 = &gcsql.Board{
		ID:            1,
		SectionID:     1,
		URI:           "test",
		Dir:           "test",
		Title:         "Testing board",
		Subtitle:      "Board for testing",
		Description:   "Board for testing",
		DefaultStyle:  "pipes.css",
		AnonymousName: "Anonymous Coward",
	}

	banPageCases = []templateTestCase{
		{
			desc: "appealable permaban",
			data: map[string]any{
				"ban": &gcsql.IPBan{
					RangeStart: "192.168.56.0",
					RangeEnd:   "192.168.56.255",
					IPBanBase: gcsql.IPBanBase{
						Permanent: true,
						CanAppeal: true,
						StaffID:   1,
						Message:   "ban message goes here",
					},
				},
				"ip":         "192.168.56.1",
				"siteConfig": testingSiteConfig,
				"systemCritical": config.SystemCriticalConfig{
					WebRoot: "/",
				},
				"boardConfig": config.BoardConfig{
					DefaultStyle: "pipes.css",
				},
			},
			validationFunc: func(t *testing.T, reader io.Reader) {
				doc, err := goquery.NewDocumentFromReader(reader)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, "YOU ARE BANNED:(", doc.Find("title").Text())
				assert.Equal(t, "all boards", doc.Find(".ban-boards").Text())
				assert.Equal(t, "ban message goes here", doc.Find(".reason").Text())
				banTime := doc.Find(".ban-timestamp").First()
				assert.Equal(t, "0001-01-01T00:00:00Z", banTime.AttrOr("datetime", ""))
				assert.Equal(t, "Mon,January 01,0001 12:00:00 AM", banTime.Text())
				assert.Equal(t, "not expire", banTime.Next().Text())
				assert.Equal(t, "192.168.56.1", doc.Find(".ban-ip").Text())
				assert.Equal(t, "You may appeal this ban:", doc.Find("#appeal-form").Prev().Nodes[0].PrevSibling.Data)
			},
		},
		{
			desc: "unappealable permaban (banned forever)",
			data: map[string]any{
				"ban": &gcsql.IPBan{
					RangeStart: "192.168.56.0",
					RangeEnd:   "192.168.56.255",
					IPBanBase: gcsql.IPBanBase{
						IsActive:  true,
						Permanent: true,
						StaffID:   1,
						Message:   "ban message goes here",
					},
				},
				"ip":         "192.168.56.1",
				"siteConfig": testingSiteConfig,
				"systemCritical": config.SystemCriticalConfig{
					WebRoot: "/",
				},
				"boardConfig": config.BoardConfig{
					DefaultStyle: "pipes.css",
				},
			},
			validationFunc: func(t *testing.T, reader io.Reader) {
				doc, err := goquery.NewDocumentFromReader(reader)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, "YOU'RE PERMABANNED,\u00A0IDIOT!", doc.Find("title").Text())
				assert.Equal(t, "all boards", doc.Find(".ban-boards").Text())
				assert.Equal(t, "ban message goes here", doc.Find(".reason").Text())
				banTime := doc.Find(".ban-timestamp").First()
				assert.Equal(t, "0001-01-01T00:00:00Z", banTime.AttrOr("datetime", ""))
				assert.Equal(t, "Mon,January 01,0001 12:00:00 AM", banTime.Text())
				assert.Equal(t, "not expire", banTime.Next().Text())
				assert.Equal(t, "192.168.56.1", doc.Find(".ban-ip").Text())
				assert.Equal(t, "/static/permabanned.jpg", doc.Find("img#banpage-image").AttrOr("src", ""))
				assert.Equal(t, 1, doc.Find("audio#jack").Length())
			},
		},
		{
			desc: "appealable temporary ban",
			data: map[string]any{
				"ban": &gcsql.IPBan{
					RangeStart: "192.168.56.0",
					RangeEnd:   "192.168.56.255",
					IPBanBase: gcsql.IPBanBase{
						CanAppeal: true,
						StaffID:   1,
						Message:   "ban message goes here",
					},
				},
				"ip":         "192.168.56.1",
				"siteConfig": testingSiteConfig,
				"systemCritical": config.SystemCriticalConfig{
					WebRoot: "/",
				},
				"boardConfig": config.BoardConfig{
					DefaultStyle: "pipes.css",
				},
			},
			validationFunc: func(t *testing.T, reader io.Reader) {
				doc, err := goquery.NewDocumentFromReader(reader)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, "YOU ARE BANNED:(", doc.Find("title").Text())
				assert.Equal(t, "all boards", doc.Find(".ban-boards").Text())
				assert.Equal(t, "ban message goes here", doc.Find(".reason").Text())
				banTime := doc.Find(".ban-timestamp").First()
				assert.Equal(t, "0001-01-01T00:00:00Z", banTime.AttrOr("datetime", ""))
				assert.Equal(t, "Mon,January 01,0001 12:00:00 AM", banTime.Text())
				assert.Equal(t, "Mon, January 01, 0001 12:00:00 AM", banTime.Next().Text())
				assert.Equal(t, "192.168.56.1", doc.Find(".ban-ip").Text())
				assert.Equal(t, "You may appeal this ban:", doc.Find("#appeal-form").Prev().Nodes[0].PrevSibling.Data)
			},
		},
		{
			desc: "unappealable temporary ban",
			data: map[string]any{
				"ban": &gcsql.IPBan{
					RangeStart: "192.168.56.0",
					RangeEnd:   "192.168.56.255",
					IPBanBase: gcsql.IPBanBase{
						StaffID: 1,
						Message: "ban message goes here",
					},
				},
				"ip":         "192.168.56.1",
				"siteConfig": testingSiteConfig,
				"systemCritical": config.SystemCriticalConfig{
					WebRoot: "/",
				},
				"boardConfig": config.BoardConfig{
					DefaultStyle: "pipes.css",
				},
			},
			validationFunc: func(t *testing.T, reader io.Reader) {
				doc, err := goquery.NewDocumentFromReader(reader)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, "YOU ARE BANNED:(", doc.Find("title").Text())
				assert.Equal(t, "You are banned from posting onall boardsfor the following reason:ban message goes hereYour ban was placed onMon,January 01,0001 12:00:00 AM and will expire on Mon, January 01, 0001 12:00:00 AM.Your IP address is192.168.56.1.You maynot appeal this ban.", doc.Find("#ban-info").Text())
			},
		},
	}

	boardPageTestCases = []templateTestCase{
		{
			desc: "base case, no threads",
			data: map[string]any{
				"boardConfig": simpleBoardConfig,
				"board":       simpleBoard1,
				"numPages":    1,
				"sections": []gcsql.Section{
					{ID: 1},
				},
			},
			getDefaultStyle: true,
			validationFunc: func(t *testing.T, reader io.Reader) {
				doc, err := goquery.NewDocumentFromReader(reader)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, "/test/-Testing board", doc.Find("title").Text())
				assert.Equal(t, "/test/-Testing board", doc.Find("#board-title").Text())
				assert.Equal(t, "Board for testingCatalog | Bottom", doc.Find("#board-subtitle").Text())
				assert.Equal(t, 1, doc.Find("#postbox-area").Length())
				assert.Equal(t, 1, doc.Find("#main-form").Length())
				assert.Equal(t, 0, doc.Find("#main-form .thread").Length())
			},
		},
		{
			desc: "base case, multi threads and pages",
			data: map[string]any{
				"boardConfig": simpleBoardConfig,
				"board":       simpleBoard1,
				"numPages":    2,
				"sections": []gcsql.Section{
					{ID: 1},
				},
				"threads": []map[string]any{
					{
						"Posts": []*building.Post{
							{
								ParentID: 1,
								Post: gcsql.Post{
									ID:        1,
									IsTopPost: true,
									Name:      "Test name",
									Tripcode:  "Tripcode",
									Subject:   "Test subject",
									Message:   "Test message",
									CreatedOn: time.Now(),
								},
							},
							{
								ParentID: 1,
								Post: gcsql.Post{
									ID:        2,
									Name:      "Test name 2",
									Tripcode:  "Tripcode",
									Message:   "Test message 2",
									CreatedOn: time.Now(),
								},
							},
						},
						"OmittedPosts": 0,
					},
					{
						"Posts": []*building.Post{
							{
								ParentID: 2,
								Post: gcsql.Post{
									ID:               3,
									IsTopPost:        true,
									IsSecureTripcode: true,
									Name:             "Test name 3",
									Tripcode:         "Secure",
									Subject:          "Test subject 3",
									Message:          "Test message 3",
									CreatedOn:        time.Now(),
								},
							},
						},
						"OmittedPosts": 0,
					},
				},
			},
			getDefaultStyle: true,
			validationFunc: func(t *testing.T, reader io.Reader) {
				doc, err := goquery.NewDocumentFromReader(reader)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, "/test/-Testing board", doc.Find("title").Text())
				assert.Equal(t, "/test/-Testing board", doc.Find("#board-title").Text())
				assert.Equal(t, "Board for testingCatalog | Bottom", doc.Find("#board-subtitle").Text())
				assert.Equal(t, 1, doc.Find("#postbox-area").Length())
				assert.Equal(t, 1, doc.Find("#main-form").Length())

				threads := doc.Find("#main-form .thread")
				thread1 := doc.Find("#main-form .thread").Eq(0)
				assert.Equal(t, 2, threads.Length())
				assert.Equal(t, 1, thread1.Find(".op-post").Length())
				assert.Equal(t, 1, thread1.Find(".reply-container").Length())
				assert.Equal(t, "Test name", thread1.Find(".op-post .postername").Text())
				assert.Equal(t, "!Tripcode", thread1.Find(".op-post .tripcode").Text())

				thread2 := doc.Find("#main-form .thread").Eq(1)
				assert.Equal(t, 1, thread2.Find(".op-post").Length())
				assert.Equal(t, 0, thread2.Find(".reply-container").Length())
				assert.Equal(t, "Test name 3", thread2.Find(".op-post .postername").Text())
				assert.Equal(t, "!!Secure", thread2.Find(".op-post .tripcode").Text())

				assert.Equal(t, 2, doc.Find("#left-bottom-content #pages a").Length())
			},
		},
	}

	jsConstsCases = []templateTestCase{
		{
			desc: "base test",
			data: map[string]any{
				"styles": []config.Style{
					{Name: "Pipes", Filename: "pipes.css"},
					{Name: "Yotsuba A", Filename: "yotsuba.css"},
				},
				"defaultStyle": "pipes.css",
				"webroot":      "/",
				"timezone":     -1,
				"fileTypes": map[string]string{
					".ext": "thumb.png",
				},
			},
			validationFunc: func(t *testing.T, reader io.Reader) {
				ba, err := io.ReadAll(reader)
				if assert.NoError(t, err) {
					assert.Equal(t,
						`const styles=[{Name:"Pipes",Filename:"pipes.css"},{Name:"Yotsuba A",Filename:"yotsuba.css"}];const defaultStyle="pipes.css";const webroot="/";const serverTZ=-1;const fileTypes=[".ext",];`,
						string(ba))
				}
			},
		},
		{
			desc: "empty values",
			data: map[string]any{
				"defaultStyle": "",
				"webroot":      "",
				"timezone":     0,
			},
			validationFunc: func(t *testing.T, reader io.Reader) {
				ba, err := io.ReadAll(reader)
				if assert.NoError(t, err) {
					assert.Equal(t,
						`const styles=[];const defaultStyle="";const webroot="";const serverTZ=0;const fileTypes=[];`,
						string(ba))
				}
			},
		},
		{
			desc: "escaped string",
			data: map[string]any{
				"defaultStyle": `"a\a"`,
				"webroot":      "",
				"timezone":     0,
			},
			validationFunc: func(t *testing.T, reader io.Reader) {
				ba, err := io.ReadAll(reader)
				if assert.NoError(t, err) {
					assert.Equal(t,
						`const styles=[];const defaultStyle="\&#34;a\\a\&#34;";const webroot="";const serverTZ=0;const fileTypes=[];`,
						string(ba))
				}
			},
		},
	}

	baseFooterCases = []templateTestCase{
		{
			desc: "base footer test",
			data: map[string]any{
				"boardConfig": simpleBoardConfig,
				"board":       simpleBoard1,
				"numPages":    1,
				"sections": []gcsql.Section{
					{ID: 1},
				},
			},
			validationFunc: func(t *testing.T, reader io.Reader) {
				ba, err := io.ReadAll(reader)
				if assert.NoError(t, err) {
					assert.Equal(t, `<footer>Powered by<a href="http://github.com/gochan-org/gochan/">Gochan 4.0</a><br /></footer></div></body></html>`, string(ba))
				}
			},
		},
	}

	baseHeaderCases = []templateTestCase{
		{
			desc: "Header Test test board",
			data: map[string]any{
				"boardConfig": simpleBoardConfig,
				"board":       simpleBoard1,
				"numPages":    1,
				"sections": []gcsql.Section{
					{ID: 1},
				},
			},
			getDefaultStyle: true,
			validationFunc: func(t *testing.T, reader io.Reader) {
				doc, err := goquery.NewDocumentFromReader(reader)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, "/test/-Testing board", doc.Find("title").Text())
				assert.Equal(t, "/css/global.css", doc.Find("link[rel='stylesheet']").AttrOr("href", ""))
				assert.Equal(t, "/css/pipes.css", doc.Find("link#theme").AttrOr("href", ""))
				assert.Equal(t, "/favicon.png", doc.Find("link[rel='shortcut icon']").AttrOr("href", ""))
				assert.Equal(t, "/js/consts.js", doc.Find("script[src='/js/consts.js']").AttrOr("src", ""))
				assert.Equal(t, "/js/gochan.js", doc.Find("script[src='/js/gochan.js']").AttrOr("src", ""))
				assert.Equal(t, "home", doc.Find("#topbar a.topbar-item").First().Text())
				assert.Equal(t, "/test/", doc.Find("#topbar a.topbar-item").Eq(1).Text())
				assert.Equal(t, "/test2/", doc.Find("#topbar a.topbar-item").Eq(2).Text())
			},
		},
		{
			desc: "Perma Ban Header Test",
			data: map[string]any{
				"ban": &gcsql.IPBan{
					RangeStart: "192.168.56.0",
					RangeEnd:   "192.168.56.255",
					IPBanBase: gcsql.IPBanBase{
						IsActive:  true,
						Permanent: true,
						StaffID:   1,
						Message:   "ban message goes here",
					},
				},
				"ip":         "192.168.56.1",
				"siteConfig": testingSiteConfig,
				"systemCritical": config.SystemCriticalConfig{
					WebRoot: "/",
				},
				"boardConfig": config.BoardConfig{
					DefaultStyle: "pipes.css",
				},
			},
			validationFunc: func(t *testing.T, reader io.Reader) {
				doc, err := goquery.NewDocumentFromReader(reader)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, "YOU'RE PERMABANNED,\u00a0IDIOT!", doc.Find("title").Text())
				assert.Equal(t, "/css/global.css", doc.Find("link[rel='stylesheet']").AttrOr("href", ""))
				assert.Equal(t, "/css/pipes.css", doc.Find("link#theme").AttrOr("href", ""))
				assert.Equal(t, "/favicon.png", doc.Find("link[rel='shortcut icon']").AttrOr("href", ""))
				assert.Equal(t, "/js/consts.js", doc.Find("script[src='/js/consts.js']").AttrOr("src", ""))
				assert.Equal(t, "/js/gochan.js", doc.Find("script[src='/js/gochan.js']").AttrOr("src", ""))
			},
		},
	}
)

type templateTestCase struct {
	desc            string
	data            any
	expectsError    bool
	getDefaultStyle bool

	validationFunc func(t *testing.T, reader io.Reader)
}

func (tC *templateTestCase) Run(t *testing.T, templateName string) {
	var buf bytes.Buffer

	err := serverutil.MinifyTemplate(templateName, tC.data, &buf, "text/javascript")
	if tC.expectsError {
		assert.Error(t, err)
	} else {
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		if assert.NotNilf(t, tC.validationFunc, "Validation function for %q is not implemented", tC.desc) {
			tC.validationFunc(t, &buf)
		}
	}
}

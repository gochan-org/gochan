package config

var (
	defaultGochanConfig = &GochanConfig{
		SystemCriticalConfig: SystemCriticalConfig{
			WebRoot: "/",
			SQLConfig: SQLConfig{
				DBTimeoutSeconds:     DefaultSQLTimeout,
				DBMaxOpenConnections: DefaultSQLMaxConns,
				DBMaxIdleConnections: DefaultSQLMaxConns,
				DBConnMaxLifetimeMin: DefaultSQLConnMaxLifetimeMin,
			},
			CheckRequestReferer: true,
		},
		SiteConfig: SiteConfig{
			FirstPage:             []string{"index.html", "firstrun.html", "1.html"},
			CookieMaxAge:          "1y",
			StaffSessionDuration:  "3mo",
			SiteName:              "Gochan",
			MinifyHTML:            true,
			MinifyJS:              true,
			MaxRecentPosts:        15,
			EnableAppeals:         true,
			FingerprintHashLength: 16,
		},
		BoardConfig: BoardConfig{
			isGlobal:            true,
			InheritGlobalStyles: true,
			DateTimeFormat:      "Mon, January 02, 2006 3:04:05 PM",
			Banners: []PageBanner{
				{Filename: "gochan_go-parody.png", Width: 300, Height: 100},
			},
			Styles: []Style{
				{Name: "Pipes", Filename: "pipes.css"},
				{Name: "BunkerChan", Filename: "bunkerchan.css"},
				{Name: "Burichan", Filename: "burichan.css"},
				{Name: "Clear", Filename: "clear.css"},
				{Name: "Dark", Filename: "dark.css"},
				{Name: "Photon", Filename: "photon.css"},
				{Name: "Yotsuba", Filename: "yotsuba.css"},
				{Name: "Yotsuba B", Filename: "yotsubab.css"},
				{Name: "Windows 9x", Filename: "win9x.css"},
			},
			DefaultStyle:    "pipes.css",
			LockdownMessage: "This imageboard has temporarily disabled posting. We apologize for the inconvenience",

			EnableSpoileredImages:  true,
			EnableSpoileredThreads: true,

			PostConfig: PostConfig{
				ThreadsPerPage:           20,
				RepliesOnBoardPage:       3,
				StickyRepliesOnBoardPage: 1,
				EnableCyclicThreads:      true,
				CyclicThreadNumPosts:     500,
				BanMessage:               "USER WAS BANNED FOR THIS POST",
				EmbedWidth:               400,
				EmbedHeight:              300,
				ImagesOpenNewTab:         true,
				NewTabOnExternalLinks:    true,
			},
			UploadConfig: UploadConfig{
				ThumbWidth:         200,
				ThumbHeight:        200,
				ThumbWidthReply:    125,
				ThumbHeightReply:   125,
				ThumbWidthCatalog:  50,
				ThumbHeightCatalog: 50,
			},
			Worksafe: true,
			Cooldowns: BoardCooldowns{
				NewThread:  30,
				Reply:      7,
				ImageReply: 7,
			},
			RenderURLsAsLinks: true,
		},
	}
)

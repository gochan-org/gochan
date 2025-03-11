package posting

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type embedTestCase struct {
	desc           string
	url            string
	expectError    bool
	expectFilename string
	boardHasEmbeds bool
}

var (
	embedTestCases = []embedTestCase{
		{
			desc:           "youtube",
			url:            "https://www.youtube.com/watch?v=123456",
			expectError:    false,
			expectFilename: "embed:youtube:123456",
			boardHasEmbeds: true,
		},
		{
			desc:           "youtube short",
			url:            "https://youtu.be/123456",
			expectError:    false,
			expectFilename: "embed:youtube:123456",
			boardHasEmbeds: true,
		},
		{
			desc:           "vimeo",
			url:            "https://vimeo.com/123456",
			expectError:    false,
			expectFilename: "embed:vimeo:123456",
			boardHasEmbeds: true,
		},
		{
			desc:           "unrecognized",
			url:            "https://example.com/123456",
			expectError:    true,
			boardHasEmbeds: true,
		},
		{
			desc:           "no URL",
			boardHasEmbeds: true,
		},
		{
			desc:        "no embeds",
			url:         "https://www.youtube.com/watch?v=123456",
			expectError: true,
		},
	}

	testEmbedMatchers = map[string]config.EmbedMatcher{
		"youtube": {
			URLRegex:             "^https?://(?:(?:(?:www\\.)?youtube\\.com/watch\\?v=)|(?:youtu\\.be/))([^&]+)",
			EmbedTemplate:        "<iframe class=\"embed\" width={{.ThumbWidth}} height={{.ThumbHeight}} src=\"https://www.youtube-nocookie.com/embed/{{.VideoID}}\" allow=\"accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share\" referrerpolicy=\"strict-origin-when-cross-origin\" allowfullscreen></iframe>",
			ThumbnailURLTemplate: "https://img.youtube.com/vi/{{.VideoID}}/0.jpg",
		},
		"vimeo": {
			URLRegex:             "^https?://(?:\\w+\\.)?vimeo\\.com/(\\d{2,10})",
			EmbedTemplate:        "<iframe src=\"https://player.vimeo.com/video/{{.VideoID}}\" class=\"embed\" width=\"{{.ThumbWidth}}\" height=\"{{.ThumbHeight}}\" allow=\"autoplay; fullscreen; picture-in-picture; clipboard-write\" referrerpolicy=\"strict-origin-when-cross-origin\" allowfullscreen></iframe>",
			ThumbnailURLTemplate: "https://vumbnail.com/{{.VideoID}}.jpg",
		},
	}
)

func generateEmbedRequest(embedURL string) *http.Request {
	req, _ := http.NewRequest("POST", "http://example.com", http.NoBody)
	req.PostForm = url.Values{}
	if embedURL != "" {
		req.PostForm.Add("embed", embedURL)
	}
	return req
}

func embedTestRunner(t *testing.T, tc *embedTestCase, boardCfg *config.BoardConfig, warnEv, errEv *zerolog.Event) {
	if tc.boardHasEmbeds {
		boardCfg.EmbedMatchers = testEmbedMatchers
	} else {
		boardCfg.EmbedMatchers = nil
	}
	config.SetBoardConfig("test", boardCfg)
	embedUpload, err := CheckEmbed(generateEmbedRequest(tc.url), &gcsql.Post{ID: 1}, boardCfg, warnEv, errEv)
	if tc.expectError {
		assert.Error(t, err)
		return
	} else {
		assert.NoError(t, err)
		filename := ""
		if embedUpload != nil {
			filename = embedUpload.Filename
		}
		assert.Equal(t, tc.expectFilename, filename)
	}
	if tc.url == "" {
		assert.Nil(t, embedUpload)
	} else {
		if !tc.boardHasEmbeds {
			assert.ErrorIs(t, err, ErrNoEmbedding)
			t.FailNow()
		}
		assert.Equal(t, tc.expectFilename, embedUpload.Filename)
	}
}

func TestCheckEmbed(t *testing.T) {
	config.InitConfig("4.1.0")
	// Set up a board config with some embed matchers
	boardCfg := &config.BoardConfig{
		PostConfig: config.PostConfig{
			EmbedWidth:  400,
			EmbedHeight: 300,
		},
	}
	config.SetBoardConfig("test", boardCfg)
	_, warnEv, errEv := testutil.GetTestLogs(t)
	for _, tc := range embedTestCases {
		t.Run(tc.desc, func(t *testing.T) {
			embedTestRunner(t, &tc, boardCfg, warnEv, errEv)
		})
	}
}

package config

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

var embedMatcherTestCases = []testCaseEmbedMatchers{
	{
		description:                 "Youtube",
		url:                         "https://www.youtube.com/watch?v=mdkKLlUEAow",
		expectedSiteID:              "youtube",
		expectedMediaID:             "mdkKLlUEAow",
		expectedEmbedTemplateOutput: `<iframe class="embed" width=200 height=200 src="https://www.youtube-nocookie.com/embed/mdkKLlUEAow" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>`,
		expectedThumbnailURL:        "https://img.youtube.com/vi/mdkKLlUEAow/0.jpg",
	},
	{
		description:                 "Vimeo",
		url:                         "https://vimeo.com/55073825",
		expectedSiteID:              "vimeo",
		expectedMediaID:             "55073825",
		expectedEmbedTemplateOutput: `<iframe src="https://player.vimeo.com/video/55073825" class="embed" width=200 height=200 allow="autoplay; fullscreen; picture-in-picture; clipboard-write" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>`,
		expectedThumbnailURL:        "https://vumbnail.com/55073825.jpg",
	},
	{
		description:                 "Raw video URL",
		url:                         "http://example.com/videos/video.mp4",
		expectedSiteID:              "rawvideo",
		expectedMediaID:             "http://example.com/videos/video.mp4",
		expectedEmbedTemplateOutput: `<video class="embed embed-rawvideo" src="http://example.com/videos/video.mp4" style="max-width:200px; max-height:200px"></video>`,
		expectedMediaURL:            "http://example.com/videos/video.mp4",
	},
	{
		description:        "No match",
		url:                "http://example.com/notavideo.txt",
		expectMediaIDError: true,
	},
	{
		description:          "Embed template error",
		url:                  "http://embedtemplateerror.com/video.ogv",
		expectedSiteID:       "embedtemplateerror",
		expectTemplatesError: true,
		expectedMediaID:      "video.ogv",
	},
}

type testCaseEmbedMatchers struct {
	description string
	url         string

	expectedSiteID              string
	expectedMediaID             string
	expectedEmbedTemplateOutput string
	expectedThumbnailURL        string
	expectedMediaURL            string

	expectMediaIDError   bool
	expectTemplatesError bool
}

func TestEmbedMatchers(t *testing.T) {
	basePath := t.TempDir()
	defer resetTestConfig(t)
	for _, tc := range embedMatcherTestCases {
		t.Run(tc.description, func(t *testing.T) {
			err := initializeExampleConfig(t, basePath, func(cfg *GochanConfig) {
				cfg.DBtype = "sqlite3"
				cfg.DBhost = ":memory:"
				if tc.expectTemplatesError {
					cfg.EmbedMatchers["embedtemplateerror"] = EmbedMatcher{
						URLRegex:      "http://embedtemplateerror.com/(video.ogv)",
						EmbedTemplate: "{{",
					}
				}
			})
			if tc.expectTemplatesError {
				assert.Error(t, err)
				return
			}
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			assert.NotNil(t, cfg.EmbedMatchers)
			assert.NotEmpty(t, cfg.EmbedMatchers)
			assert.True(t, cfg.HasEmbedMatchers())

			siteID, mediaID, err := cfg.GetEmbedMediaID(tc.url)
			if tc.expectMediaIDError {
				assert.Error(t, err)
				return
			}
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			assert.Equal(t, tc.expectedSiteID, siteID)
			assert.Equal(t, tc.expectedMediaID, mediaID)

			embedTemplate, thumbURLTemplate, err := cfg.GetEmbedTemplates(siteID)
			if tc.expectTemplatesError {
				assert.Error(t, err)
				return
			}
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			var buf bytes.Buffer
			err = embedTemplate.Execute(&buf, map[string]any{
				"HandlerID":   siteID,
				"MediaID":     mediaID,
				"ThumbWidth":  200,
				"ThumbHeight": 200,
			})
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			assert.Equal(t, tc.expectedEmbedTemplateOutput, buf.String())

			buf.Reset()
			if tc.expectedThumbnailURL == "" {
				assert.Nil(t, thumbURLTemplate)
			} else {
				if !assert.NotNil(t, thumbURLTemplate) {
					t.FailNow()
				}
				err = thumbURLTemplate.Execute(&buf, map[string]string{
					"MediaID": mediaID,
				})
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, tc.expectedThumbnailURL, buf.String())
			}
			if tc.expectedMediaURL != "" {
				buf.Reset()
				linkTemplate, err := cfg.GetLinkTemplate(siteID)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				err = linkTemplate.Execute(&buf, map[string]string{
					"MediaID": mediaID,
				})
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, tc.expectedMediaURL, buf.String())
			}
		})
	}
}

package posting

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type embedTestCase struct {
	desc           string
	url            string
	expectError    bool
	expectUpload   *gcsql.Upload
	boardHasEmbeds bool
}

var (
	embedTestCases = []embedTestCase{
		{
			desc:        "youtube",
			url:         "https://www.youtube.com/watch?v=123456",
			expectError: false,
			expectUpload: &gcsql.Upload{
				Filename:         "embed:youtube",
				OriginalFilename: "123456",
			},
			boardHasEmbeds: true,
		},
		{
			desc:        "youtube short",
			url:         "https://youtu.be/123456",
			expectError: false,
			expectUpload: &gcsql.Upload{
				Filename:         "embed:youtube",
				OriginalFilename: "123456",
			},
			boardHasEmbeds: true,
		},
		{
			desc:        "vimeo",
			url:         "https://vimeo.com/123456",
			expectError: false,
			expectUpload: &gcsql.Upload{
				Filename:         "embed:vimeo",
				OriginalFilename: "123456",
			},
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
		{
			desc:        "raw video",
			url:         "https://example.com/blah/video.mp4",
			expectError: false,
			expectUpload: &gcsql.Upload{
				Filename:         "embed:rawvideo",
				OriginalFilename: "https://example.com/blah/video.mp4",
			},
			boardHasEmbeds: true,
		},
	}

	testEmbedMatchers = map[string]config.EmbedMatcher{
		"youtube": {
			URLRegex:             `^https?://(?:(?:(?:www\.)?youtube\.com/watch\?v=)|(?:youtu\.be/))([^&]+)`,
			EmbedTemplate:        `<iframe class="embed" width={{.ThumbWidth}} height={{.ThumbHeight}} src="https://www.youtube-nocookie.com/embed/{{.VideoID}}" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>`,
			ThumbnailURLTemplate: "https://img.youtube.com/vi/{{.VideoID}}/0.jpg",
		},
		"vimeo": {
			URLRegex:             `^https?://(?:\w+\.)?vimeo\.com/(\d{2,10})`,
			EmbedTemplate:        `<iframe src="https://player.vimeo.com/video/{{.VideoID}}" class="embed" width="{{.ThumbWidth}}" height="{{.ThumbHeight}}" allow="autoplay; fullscreen; picture-in-picture; clipboard-write" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>`,
			VideoIDSubmatchIndex: intPointer(1),
			ThumbnailURLTemplate: "https://vumbnail.com/{{.VideoID}}.jpg",
		},
		"rawvideo": {
			URLRegex:             `^https?://\S+\.\S+/\S+/(\S+\.(?:mp4|webm))`,
			EmbedTemplate:        `<video class="embed" controls><source src="{{.VideoID}}" type="video/mp4"></video>`,
			VideoIDSubmatchIndex: intPointer(0),
		},
	}
)

func intPointer(i int) *int {
	return &i
}

func generateEmbedRequest(embedURL string) *http.Request {
	req, _ := http.NewRequest("POST", "http://example.com", http.NoBody)
	req.PostForm = url.Values{}
	if embedURL != "" {
		req.PostForm.Add("embed", embedURL)
	}
	return req
}

func embedTestRunner(t *testing.T, tc *embedTestCase, boardCfg *config.BoardConfig, warnEv, errEv *zerolog.Event) {
	db, mock, err := sqlmock.New()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	err = gcsql.SetTestingDB("mysql", "gochan", "", db)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer db.Close()

	if tc.boardHasEmbeds {
		boardCfg.EmbedMatchers = testEmbedMatchers
		mock.ExpectBegin()
		mock.ExpectPrepare(`SELECT filename, dir FROM files\s+` +
			`JOIN posts ON post_id = posts.id\s+` +
			`JOIN threads ON thread_id = threads.id\s+` +
			`JOIN boards ON boards.id = board_id\s+` +
			`WHERE posts.id = ?`).
			ExpectQuery().WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"filename", "dir"}).AddRow("", ""))
		mock.ExpectPrepare(`SELECT COALESCE\(MAX\(file_order\) \+ 1, 0\) FROM files WHERE post_id = \?`).
			ExpectQuery().WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"COALESCE(MAX(file_order) + 1, 0)"}).AddRow(1))
		mock.ExpectPrepare(`INSERT INTO files\s+` +
			`\(\s*post_id, file_order, original_filename, filename, checksum, file_size, is_spoilered, thumbnail_width, thumbnail_height, width, height\)\s*` +
			`VALUES\(\?,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?\)`).ExpectExec().WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectPrepare(`SELECT MAX\(id\) FROM files`).
			ExpectQuery().WillReturnRows(sqlmock.NewRows([]string{"MAX(id)"}).AddRow(99))
		mock.ExpectCommit()
	} else {
		boardCfg.EmbedMatchers = nil
	}
	if !assert.NoError(t, config.SetBoardConfig("test", boardCfg)) {
		t.FailNow()
	}

	embedUpload, err := AttachEmbed(generateEmbedRequest(tc.url), &gcsql.Post{ID: 1}, boardCfg, warnEv, errEv)
	if tc.expectError {
		assert.Error(t, err)
		return
	}
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if tc.url == "" {
		assert.Nil(t, embedUpload)
		return
	}
	if !assert.NotNil(t, embedUpload) {
		t.FailNow()
	}
	assert.Equal(t, tc.expectUpload.Filename, embedUpload.Filename)
	assert.Equal(t, tc.expectUpload.OriginalFilename, embedUpload.OriginalFilename)
	assert.Equal(t, embedUpload.ID, 99)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAttachEmbed(t *testing.T) {
	config.InitConfig("4.1.0")
	config.SetTestDBConfig("mysql", "localhost", "gochan", "gochan", "", "")

	// Set up a board config with some embed matchers
	boardCfg := &config.BoardConfig{
		PostConfig: config.PostConfig{
			EmbedWidth:  400,
			EmbedHeight: 300,
		},
	}
	config.SetBoardConfig("test", boardCfg)
	_, warnEv, errEv := testutil.GetTestLogs(t)
	defer gcutil.LogDiscard(warnEv, errEv)
	for _, tc := range embedTestCases {
		t.Run(tc.desc, func(t *testing.T) {
			embedTestRunner(t, &tc, boardCfg, warnEv, errEv)
		})
	}
}

func TestOnlyAllowOneEmbed(t *testing.T) {
	// verify that only one embed or file upload is allowed. Multiple files/uploading is on my to-do list,
	// but for now, single upload/embedding is enforced.
	config.InitConfig("4.1.0")
	config.SetTestDBConfig("mysql", "localhost", "gochan", "gochan", "", "")
	boardCfg := &config.BoardConfig{
		PostConfig: config.PostConfig{
			EmbedWidth:    400,
			EmbedHeight:   300,
			EmbedMatchers: testEmbedMatchers,
		},
	}
	config.SetBoardConfig("test", boardCfg)
	_, warnEv, errEv := testutil.GetTestLogs(t)
	defer gcutil.LogDiscard(warnEv, errEv)
	db, mock, err := sqlmock.New()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	err = gcsql.SetTestingDB("mysql", "gochan", "", db)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer db.Close()
	const prepStr = `SELECT filename, dir FROM files\s+` +
		`JOIN posts ON post_id = posts.id\s+` +
		`JOIN threads ON thread_id = threads.id\s+` +
		`JOIN boards ON boards.id = board_id\s+` +
		`WHERE posts.id = ?`
	mock.ExpectBegin()
	mock.ExpectPrepare(prepStr).
		ExpectQuery().WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"filename", "dir"}).AddRow("file.png", "test"))
	_, err = AttachEmbed(generateEmbedRequest("https://www.youtube.com/watch?v=123456"),
		&gcsql.Post{ID: 1}, boardCfg, warnEv, errEv)
	assert.ErrorIs(t, err, gcsql.ErrUploadAlreadyAttached)
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}

	mock.ExpectBegin()
	mock.ExpectPrepare(prepStr).
		ExpectQuery().WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"filename", "dir"}).AddRow("embed:youtube", "test"))
	_, err = AttachEmbed(generateEmbedRequest("https://www.youtube.com/watch?v=123456"),
		&gcsql.Post{ID: 1}, boardCfg, warnEv, errEv)
	assert.ErrorIs(t, err, gcsql.ErrEmbedAlreadyAttached)
	assert.NoError(t, mock.ExpectationsWereMet())
}

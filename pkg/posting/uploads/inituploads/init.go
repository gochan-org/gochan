package uploads

import (
	"path"
	"text/template"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
)

func init() {
	gctemplates.AddTemplateFuncs(template.FuncMap{
		"getCatalogThumbnail": func(img string) string {
			_, catalogThumb := uploads.GetThumbnailFilenames(img)
			return catalogThumb
		},
		"getThreadThumbnail": func(img string) string {
			thumb, _ := uploads.GetThumbnailFilenames(img)
			return thumb
		},
		"getUploadType": func(name string) string {
			return uploads.GetThumbnailExtension(path.Ext(name))
		},
		"getThumbnailWebPath": func(postID int) string {
			filename, board, err := gcsql.GetUploadFilenameAndBoard(postID)
			if err != nil {
				gcutil.LogError(err).Caller().Int("postID", postID).Send()
				return ""
			}
			filename, _ = uploads.GetThumbnailFilenames(filename)
			return config.WebPath(board, "thumb", filename)
		},
	})
}

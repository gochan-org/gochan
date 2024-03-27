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

func getCatalogThumbnailTmplFunc(img string) string {
	_, catalogThumb := uploads.GetThumbnailFilenames(img)
	return catalogThumb
}

func getThreadThumbnailTmplFunc(img string) string {
	thumb, _ := uploads.GetThumbnailFilenames(img)
	return thumb
}

func getUploadTypeTmplFunc(name string) string {
	return uploads.GetThumbnailExtension(path.Ext(name))
}

func getThumbnailWebPathTmplFunc(postID int) string {
	filename, board, err := gcsql.GetUploadFilenameAndBoard(postID)
	if err != nil {
		gcutil.LogError(err).Caller().Int("postID", postID).Send()
		return ""
	}
	filename, _ = uploads.GetThumbnailFilenames(filename)
	return config.WebPath(board, "thumb", filename)
}

func init() {
	gctemplates.AddTemplateFuncs(template.FuncMap{
		"getCatalogThumbnail": getCatalogThumbnailTmplFunc,
		"getThreadThumbnail":  getThreadThumbnailTmplFunc,
		"getUploadType":       getUploadTypeTmplFunc,
		"getThumbnailWebPath": getThumbnailWebPathTmplFunc,
	})
}

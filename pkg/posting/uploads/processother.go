package uploads

import (
	"errors"
	"os"
	"path"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/rs/zerolog"
)

var (
	ErrUnsupportedFileExt = errors.New("unsupported file extension")
)

func processOther(upload *gcsql.Upload, post *gcsql.Post, board string, filePath string, thumbPath string, catalogThumbPath string, infoEv *zerolog.Event, errEv *zerolog.Event) error {
	boardConfig := config.GetBoardConfig(board)
	ext := path.Ext(filePath)
	cfgThumb, ok := boardConfig.AllowOtherExtensions[ext]
	if !ok {
		errEv.Err(ErrUnsupportedFileExt).Str("ext", ext).Caller().Send()
		return ErrUnsupportedFileExt
	}
	infoEv.Str("post", "withOther")

	stat, err := os.Stat(filePath)
	if err != nil {
		errEv.Err(err).Caller().
			Str("filePath", filePath).Send()
		return err
	}

	upload.FileSize = int(stat.Size())
	if post.ThreadID == 0 {
		// OP
		upload.ThumbnailWidth = boardConfig.ThumbWidth
		upload.ThumbnailHeight = boardConfig.ThumbHeight
	} else {
		// reply
		upload.ThumbnailWidth = boardConfig.ThumbWidthReply
		upload.ThumbnailHeight = boardConfig.ThumbHeightReply
	}
	staticThumbPath := path.Join("static/", cfgThumb)
	originalThumbPath := path.Join(config.GetSystemCriticalConfig().DocumentRoot, staticThumbPath)
	if _, err = os.Stat(originalThumbPath); err != nil {
		errEv.Err(err).Str("originalThumbPath", originalThumbPath).Send()
		return err
	}

	if err = os.Symlink(originalThumbPath, thumbPath); err != nil {
		os.Remove(filePath)
		errEv.Err(err).Caller().
			Str("filePath", filePath).Send()
		return err
	}
	if post.ThreadID == 0 {
		if err = os.Symlink(originalThumbPath, catalogThumbPath); err != nil {
			os.Remove(filePath)
			errEv.Err(err).Caller().
				Str("filePath", filePath).Send()
			return err
		}
	}
	return nil
}

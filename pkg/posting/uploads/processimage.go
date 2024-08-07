package uploads

import (
	"errors"
	"image"
	"image/gif"
	"os"
	"os/exec"
	"path"

	"github.com/disintegration/imaging"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

var (
	ErrStripMetadata = errors.New("unable to strip image metadata")
)

func stripImageMetadata(filePath string, boardConfig *config.BoardConfig) (err error) {
	var stripFlag string
	switch boardConfig.StripImageMetadata {
	case "exif":
		stripFlag = "-EXIF="
	case "all":
		stripFlag = "-all="
	case "none":
		fallthrough
	case "":
		return nil
	}
	err = exec.Command(boardConfig.ExiftoolPath, "-overwrite_original_in_place", stripFlag, filePath).Run()
	return
}

func numImageFrames(imgPath string) (int, error) {
	if path.Ext(imgPath) != ".gif" {
		return 1, nil
	}
	fi, err := os.Open(imgPath)
	if err != nil {
		return 0, err
	}
	g, err := gif.DecodeAll(fi)
	if err != nil {
		return 0, err
	}
	return len(g.Image), fi.Close()
}

func processImage(upload *gcsql.Upload, post *gcsql.Post, board string, filePath string, thumbPath string, catalogThumbPath string, infoEv *zerolog.Event, accessEv *zerolog.Event, errEv *zerolog.Event) error {
	boardConfig := config.GetBoardConfig(board)
	gcutil.LogStr("stripImageMetadata", boardConfig.StripImageMetadata, errEv, infoEv)

	err := stripImageMetadata(filePath, boardConfig)
	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to strip metadata")
		return ErrStripMetadata
	}
	img, err := imaging.Open(filePath)
	if err != nil {
		os.Remove(filePath)
		errEv.Err(err).Caller().
			Str("filePath", filePath).Send()
		return err
	}
	accessEv.Str("handler", "image")

	// Get image width and height, as well as thumbnail width and height
	upload.Width = img.Bounds().Max.X
	upload.Height = img.Bounds().Max.Y
	thumbType := ThumbnailReply
	if post.ThreadID == 0 {
		thumbType = ThumbnailOP
	}
	upload.ThumbnailWidth, upload.ThumbnailHeight = getThumbnailSize(upload.Width, upload.Height, board, thumbType)

	if upload.IsSpoilered {
		if err = createSpoilerThumbnail(upload, board, post.IsTopPost, thumbPath); err != nil {
			errEv.Err(err).Caller().Msg("Unable to create spoiler thumbnail")
			return ErrUnableToCreateSpoiler
		}
		return nil
	}

	shouldThumb := ShouldCreateThumbnail(filePath,
		upload.Width, upload.Height, upload.ThumbnailWidth, upload.ThumbnailHeight)

	if shouldThumb {
		var thumbnail image.Image
		var catalogThumbnail image.Image
		if post.ThreadID == 0 {
			// If this is a new thread, generate thumbnail and catalog thumbnail
			thumbnail = createImageThumbnail(img, board, ThumbnailOP)
			catalogThumbnail = createImageThumbnail(img, board, ThumbnailCatalog)
			if err = imaging.Save(catalogThumbnail, catalogThumbPath); err != nil {
				errEv.Err(err).Caller().
					Str("thumbPath", catalogThumbPath).
					Msg("Couldn't generate catalog thumbnail")
				return err
			}
		} else {
			thumbnail = createImageThumbnail(img, board, ThumbnailReply)
		}
		if err = imaging.Save(thumbnail, thumbPath); err != nil {
			errEv.Err(err).Caller().
				Str("thumbPath", thumbPath).
				Msg("Couldn't generate catalog thumbnail")
			return err
		}
	} else {
		// If image fits in thumbnail size, symlink thumbnail to original
		upload.ThumbnailWidth = img.Bounds().Max.X
		upload.ThumbnailHeight = img.Bounds().Max.Y
		if err := os.Symlink(filePath, thumbPath); err != nil {
			errEv.Err(err).Caller().
				Str("thumbPath", thumbPath).
				Msg("Couldn't generate catalog thumbnail")
			return err
		}
		if post.ThreadID == 0 {
			// Generate catalog thumbnail
			catalogThumbnail := createImageThumbnail(img, board, ThumbnailCatalog)
			if err = imaging.Save(catalogThumbnail, catalogThumbPath); err != nil {
				errEv.Err(err).Caller().
					Str("thumbPath", catalogThumbPath).
					Msg("Couldn't generate catalog thumbnail")
				return err
			}
		}
	}
	return nil
}

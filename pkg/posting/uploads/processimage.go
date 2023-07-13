package uploads

import (
	"image"
	"image/gif"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/disintegration/imaging"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/rs/zerolog"
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
	defer fi.Close()
	g, err := gif.DecodeAll(fi)
	if err != nil {
		return 0, err
	}
	return len(g.Image), nil
}

func processImage(upload *gcsql.Upload, post *gcsql.Post, board string, filePath string, thumbPath string, catalogThumbPath string, infoEv *zerolog.Event, accessEv *zerolog.Event, errEv *zerolog.Event) error {
	img, err := imaging.Open(filePath)
	if err != nil {
		os.Remove(filePath)
		errEv.Err(err).Caller().
			Str("filePath", filePath).Send()
		return err
	}
	// Get image filesize
	stat, err := os.Stat(filePath)
	if err != nil {
		errEv.Err(err).Caller().
			Str("filePath", filePath).Send()
		return err
	}
	upload.FileSize = int(stat.Size())

	// Get image width and height, as well as thumbnail width and height
	upload.Width = img.Bounds().Max.X
	upload.Height = img.Bounds().Max.Y
	thumbType := ThumbnailReply
	if post.ThreadID == 0 {
		thumbType = ThumbnailOP
	}
	upload.ThumbnailWidth, upload.ThumbnailHeight = getThumbnailSize(upload.Width, upload.Height, board, thumbType)

	documentRoot := config.GetSystemCriticalConfig().DocumentRoot
	if upload.IsSpoilered {
		// If spoiler is enabled, symlink thumbnail to spoiler image
		if _, err := os.Stat(path.Join(documentRoot, "spoiler.png")); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
		if err = syscall.Symlink(path.Join(documentRoot, "spoiler.png"), thumbPath); err != nil {
			errEv.Err(err).
				Str("thumbPath", thumbPath).
				Msg("Error creating symbolic link to thumbnail path")
			return err
		}
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
		if err := syscall.Symlink(filePath, thumbPath); err != nil {
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
	accessEv.Str("handler", "image")
	return nil
}

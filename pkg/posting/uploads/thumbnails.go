package uploads

import (
	"errors"
	"image"
	"os/exec"
	"path"
	"strconv"
	"strings"

	_ "golang.org/x/image/webp"

	"github.com/disintegration/imaging"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

type ThumbnailCategory int

const (
	ThumbnailOP ThumbnailCategory = iota
	ThumbnailReply
	ThumbnailCatalog
)

var (
	thumbnailExtensions = map[string]string{
		".gif":  ".png",
		".mp4":  ".png",
		".webm": ".png",
		".webp": ".png",
		".jfif": ".jpg",
		".jpeg": ".jpg",
	}
)

func GetThumbnailExtension(fileExt string) string {
	thumbExt, ok := thumbnailExtensions[fileExt]
	if !ok {
		return fileExt
	}
	return thumbExt
}

func SetThumbnailExtension(fileExt, thumbExt string) {
	thumbnailExtensions[fileExt] = thumbExt
}

// GetThumbnailFilenames returns the regular thumbnail and the catalog thumbnail filenames of the given upload
// filename. It does not check if the catalog actually exists (for example, if it's a reply)
func GetThumbnailFilenames(img string) (string, string) {
	ext := GetThumbnailExtension(path.Ext(img))
	index := strings.LastIndex(img, ".")
	if index < 0 || index > len(img) {
		return "", ""
	}
	return img[:index] + "t" + ext, img[:index] + "c" + ext
}

func createImageThumbnail(imageObj image.Image, boardDir string, thumbType ThumbnailCategory) image.Image {
	thumbWidth, thumbHeight := getBoardThumbnailSize(boardDir, thumbType)

	oldRect := imageObj.Bounds()
	if thumbWidth >= oldRect.Max.X && thumbHeight >= oldRect.Max.Y {
		return imageObj
	}

	thumbW, thumbH := getThumbnailSize(oldRect.Max.X, oldRect.Max.Y, boardDir, thumbType)
	imageObj = imaging.Resize(imageObj, thumbW, thumbH, imaging.CatmullRom) // resize to 600x400 px using CatmullRom cubic filter
	return imageObj
}

func createVideoThumbnail(video, thumb string, size int) error {
	sizeStr := strconv.Itoa(size)
	outputBytes, err := exec.Command("ffmpeg", "-y" /* "-itsoffset", "-1", */, "-i", video, "-vframes", "1", "-filter:v", "scale='min("+sizeStr+"\\, "+sizeStr+"):-1'", thumb).CombinedOutput()
	if err != nil {
		outputStringArr := strings.Split(string(outputBytes), "\n")
		if len(outputStringArr) > 1 {
			outputString := outputStringArr[len(outputStringArr)-2]
			err = errors.New(outputString)
		}
	}
	return err
}

func getBoardThumbnailSize(boardDir string, thumbType ThumbnailCategory) (int, int) {
	boardCfg := config.GetBoardConfig(boardDir)
	switch thumbType {
	case ThumbnailOP:
		return boardCfg.ThumbWidth, boardCfg.ThumbHeight
	case ThumbnailReply:
		return boardCfg.ThumbWidthReply, boardCfg.ThumbHeightReply
	case ThumbnailCatalog:
		return boardCfg.ThumbWidth, boardCfg.ThumbHeight
	}
	// todo: use reflect package to print location to error log, because this shouldn't happen
	return -1, -1
}

// find out what out thumbnail's width and height should be, partially ripped from Kusaba X
func getThumbnailSize(uploadWidth, uploadHeight int, boardDir string, thumbType ThumbnailCategory) (newWidth, newHeight int) {
	thumbWidth, thumbHeight := getBoardThumbnailSize(boardDir, thumbType)
	if uploadWidth < thumbWidth && uploadHeight < thumbHeight {
		newWidth = uploadWidth
		newHeight = uploadHeight
	} else if uploadWidth == uploadHeight {
		newWidth = thumbWidth
		newHeight = thumbHeight
	} else {
		var percent float32
		if uploadWidth > uploadHeight {
			percent = float32(thumbWidth) / float32(uploadWidth)
		} else {
			percent = float32(thumbHeight) / float32(uploadHeight)
		}
		newWidth = int(float32(uploadWidth) * percent)
		newHeight = int(float32(uploadHeight) * percent)
	}
	return
}
func ShouldCreateThumbnail(imgPath string, imgWidth int, imgHeight int, thumbWidth int, thumbHeight int) bool {
	ext := strings.ToLower(path.Ext(imgPath))
	if ext == ".gif" {
		numFrames, err := numImageFrames(imgPath)
		if err != nil {
			gcutil.LogError(err).
				Str("imgPath", imgPath).Send()
			return true
		}
		if numFrames > 1 {
			return true
		}
	}

	return imgWidth > thumbWidth || imgHeight > thumbHeight
}

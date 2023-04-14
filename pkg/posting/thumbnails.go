package posting

import (
	"errors"
	"image"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

func createImageThumbnail(imageObj image.Image, boardDir string, thumbType string) image.Image {
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
	outputBytes, err := exec.Command("ffmpeg", "-y", "-itsoffset", "-1", "-i", video, "-vframes", "1", "-filter:v", "scale='min("+sizeStr+"\\, "+sizeStr+"):-1'", thumb).CombinedOutput()
	if err != nil {
		outputStringArr := strings.Split(string(outputBytes), "\n")
		if len(outputStringArr) > 1 {
			outputString := outputStringArr[len(outputStringArr)-2]
			err = errors.New(outputString)
		}
	}
	return err
}

func getBoardThumbnailSize(boardDir string, thumbType string) (int, int) {
	boardCfg := config.GetBoardConfig(boardDir)
	switch thumbType {
	case "op":
		return boardCfg.ThumbWidth, boardCfg.ThumbHeight
	case "reply":
		return boardCfg.ThumbWidthReply, boardCfg.ThumbHeightReply
	case "catalog":
		return boardCfg.ThumbWidth, boardCfg.ThumbHeight
	}
	// todo: use reflect package to print location to error log, because this shouldn't happen
	return -1, -1
}

// find out what out thumbnail's width and height should be, partially ripped from Kusaba X
func getThumbnailSize(uploadWidth, uploadHeight int, boardDir string, thumbType string) (newWidth, newHeight int) {
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
func shouldCreateThumbnail(imgPath string, imgWidth int, imgHeight int, thumbWidth int, thumbHeight int) bool {
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

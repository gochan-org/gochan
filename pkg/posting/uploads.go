package posting

import (
	"errors"
	"image"
	"image/gif"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

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

func shouldCreateThumbnail(imgPath string, imgWidth int, imgHeight int, thumbWidth int, thumbHeight int) bool {
	ext := strings.ToLower(path.Ext(imgPath))
	if ext == ".gif" {
		numFrames, err := numImageFrames(imgPath)
		if err != nil {
			gclog.Printf(gclog.LErrorLog, "Error processing %q: %s", imgPath, err.Error())
			return true
		}
		if numFrames > 1 {
			return true
		}
	}

	return imgWidth > thumbWidth || imgHeight > thumbHeight
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

func getVideoInfo(path string) (map[string]int, error) {
	vidInfo := make(map[string]int)

	outputBytes, err := exec.Command("ffprobe", "-v quiet", "-show_format", "-show_streams", path).CombinedOutput()
	if err == nil && outputBytes != nil {
		outputStringArr := strings.Split(string(outputBytes), "\n")
		for _, line := range outputStringArr {
			lineArr := strings.Split(line, "=")
			if len(lineArr) < 2 {
				continue
			}

			if lineArr[0] == "width" || lineArr[0] == "height" || lineArr[0] == "size" {
				value, _ := strconv.Atoi(lineArr[1])
				vidInfo[lineArr[0]] = value
			}
		}
	}
	return vidInfo, err
}

func getNewFilename() string {
	now := time.Now().Unix()
	rand.Seed(now)
	return strconv.Itoa(int(now)) + strconv.Itoa(rand.Intn(98)+1)
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

func numImageFrames(imgPath string) (int, error) {
	ext := strings.ToLower(gcutil.GetFileExtension(imgPath))
	if ext != "gif" {
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

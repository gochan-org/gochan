package posting

import (
	"errors"
	"image"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gochan-org/gochan/pkg/config"
)

func createImageThumbnail(imageObj image.Image, size string) image.Image {
	var thumbWidth int
	var thumbHeight int
	boardCfg := config.GetBoardConfig("")

	switch size {
	case "op":
		thumbWidth = boardCfg.ThumbWidth
		thumbHeight = boardCfg.ThumbHeight
	case "reply":
		thumbWidth = boardCfg.ThumbWidthReply
		thumbHeight = boardCfg.ThumbHeightReply
	case "catalog":
		thumbWidth = boardCfg.ThumbWidthCatalog
		thumbHeight = boardCfg.ThumbHeightCatalog
	}
	oldRect := imageObj.Bounds()
	if thumbWidth >= oldRect.Max.X && thumbHeight >= oldRect.Max.Y {
		return imageObj
	}

	thumbW, thumbH := getThumbnailSize(oldRect.Max.X, oldRect.Max.Y, size)
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
func getThumbnailSize(w, h int, size string) (newWidth, newHeight int) {
	var thumbWidth int
	var thumbHeight int
	boardCfg := config.GetBoardConfig("")
	switch {
	case size == "op":
		thumbWidth = boardCfg.ThumbWidth
		thumbHeight = boardCfg.ThumbHeight
	case size == "reply":
		thumbWidth = boardCfg.ThumbWidthReply
		thumbHeight = boardCfg.ThumbHeightReply
	case size == "catalog":
		thumbWidth = boardCfg.ThumbWidthCatalog
		thumbHeight = boardCfg.ThumbHeightCatalog
	}
	if w == h {
		newWidth = thumbWidth
		newHeight = thumbHeight
	} else {
		var percent float32
		if w > h {
			percent = float32(thumbWidth) / float32(w)
		} else {
			percent = float32(thumbHeight) / float32(h)
		}
		newWidth = int(float32(w) * percent)
		newHeight = int(float32(h) * percent)
	}
	return
}

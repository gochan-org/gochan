package uploads

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

func processVideo(upload *gcsql.Upload, post *gcsql.Post, board string, filePath string, thumbPath string, catalogThumbPath string, infoEv *zerolog.Event, errEv *zerolog.Event) error {
	boardConfig := config.GetBoardConfig(board)
	infoEv.Str("post", "withVideo")
	var err error
	if post.ThreadID == 0 {
		if err = createVideoThumbnail(filePath, thumbPath, boardConfig.ThumbWidth); err != nil {
			errEv.Err(err).Caller().
				Int("thumbWidth", boardConfig.ThumbWidth).
				Msg("Error creating video thumbnail")
			return err
		}
	} else {
		if err = createVideoThumbnail(filePath, thumbPath, boardConfig.ThumbWidthReply); err != nil {
			errEv.Err(err).Caller().
				Str("thumbPath", thumbPath).
				Int("thumbWidth", boardConfig.ThumbWidthReply).
				Msg("Error creating video thumbnail for reply")
			return err
		}
	}

	if err = createVideoThumbnail(filePath, catalogThumbPath, boardConfig.ThumbWidthCatalog); err != nil {
		errEv.Err(err).Caller().
			Str("thumbPath", thumbPath).
			Int("thumbWidth", boardConfig.ThumbWidthCatalog).
			Msg("Error creating video thumbnail for catalog")
		return err
	}

	outputBytes, err := exec.Command("ffprobe", "-v", "quiet", "-show_format", "-show_streams", filePath).CombinedOutput()
	if err != nil {
		gcutil.LogError(err).Msg("Error getting video info")
		return err
	}
	if outputBytes != nil {
		outputStringArr := strings.Split(string(outputBytes), "\n")
		for _, line := range outputStringArr {
			lineArr := strings.Split(line, "=")
			if len(lineArr) < 2 {
				continue
			}
			value, _ := strconv.Atoi(lineArr[1])
			switch lineArr[0] {
			case "width":
				upload.Width = value
			case "height":
				upload.Height = value
			case "size":
				upload.FileSize = value
			}
		}
		thumbType := ThumbnailReply
		if post.ThreadID == 0 {
			thumbType = ThumbnailOP
		}
		upload.ThumbnailWidth, upload.ThumbnailHeight = getThumbnailSize(
			upload.Width, upload.Height, board, thumbType)
	}
	return nil
}

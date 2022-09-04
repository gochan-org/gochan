package posting

import (
	"os"
	"path"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

var tempCleanerTicker *time.Ticker

func tempCleaner() {
	for {
		select {
		case <-tempCleanerTicker.C:
			for p := range gcsql.TempPosts {
				post := &gcsql.TempPosts[p]
				if !time.Now().After(post.Timestamp.Add(time.Minute * 5)) {
					continue
				}
				// temporary post is >= 5 minutes, time to prune it
				gcsql.TempPosts[p] = gcsql.TempPosts[len(gcsql.TempPosts)-1]
				gcsql.TempPosts = gcsql.TempPosts[:len(gcsql.TempPosts)-1]
				if post.FilenameOriginal == "" {
					continue
				}
				var board gcsql.Board
				err := board.PopulateData(post.BoardID)
				if err != nil {
					continue
				}

				systemCritical := config.GetSystemCriticalConfig()
				fileSrc := path.Join(systemCritical.DocumentRoot, board.Dir, "src", post.FilenameOriginal)
				if err = os.Remove(fileSrc); err != nil {
					gcutil.LogError(err).
						Str("subject", "tempUpload").
						Str("filePath", fileSrc).Send()
				}

				thumbSrc := gcutil.GetThumbnailPath("thread", fileSrc)
				if err = os.Remove(thumbSrc); err != nil {
					gcutil.LogError(err).
						Str("subject", "tempUpload").
						Str("filePath", thumbSrc).Send()
				}

				if post.ParentID == 0 {
					catalogSrc := gcutil.GetThumbnailPath("catalog", fileSrc)
					if err = os.Remove(catalogSrc); err != nil {
						gcutil.LogError(err).
							Str("subject", "tempUpload").
							Str("filePath", catalogSrc).Send()
					}
				}
			}
		}
	}
}

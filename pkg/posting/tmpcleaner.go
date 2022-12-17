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
				if !time.Now().After(post.CreatedOn.Add(time.Minute * 5)) {
					continue
				}
				// temporary post is >= 5 minutes, time to prune it
				gcsql.TempPosts[p] = gcsql.TempPosts[len(gcsql.TempPosts)-1]
				gcsql.TempPosts = gcsql.TempPosts[:len(gcsql.TempPosts)-1]
				upload, err := post.GetUpload()
				if err != nil {
					continue
				}
				if upload.OriginalFilename == "" {
					continue
				}
				board, err := post.GetBoard()
				if err != nil {
					continue
				}

				systemCritical := config.GetSystemCriticalConfig()
				fileSrc := path.Join(systemCritical.DocumentRoot, board.Dir, "src", upload.OriginalFilename)
				if err = os.Remove(fileSrc); err != nil {
					gcutil.LogError(err).
						Str("subject", "tempUpload").
						Str("filePath", fileSrc).Send()
				}

				thumbSrc := upload.ThumbnailPath("thread")
				if err = os.Remove(thumbSrc); err != nil {
					gcutil.LogError(err).
						Str("subject", "tempUpload").
						Str("filePath", thumbSrc).Send()
				}

				if post.IsTopPost {
					catalogSrc := upload.ThumbnailPath("catalog")
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

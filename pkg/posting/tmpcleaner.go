package posting

import (
	"os"
	"path"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

var tempCleanerTicker *time.Ticker

func tempCleaner() {
	for {
		select {
		case <-tempCleanerTicker.C:
			for p, post := range gcsql.TempPosts {
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

				fileSrc := path.Join(config.Config.DocumentRoot, board.Dir, "src", post.FilenameOriginal)
				var gErr error
				if gErr = os.Remove(fileSrc); gErr != nil {
					gclog.Printf(errStdLogs,
						"Error pruning temporary upload for %q: %s", fileSrc, gErr.Error())
				}

				thumbSrc := gcutil.GetThumbnailPath("thread", fileSrc)
				if gErr = os.Remove(thumbSrc); gErr != nil {
					gclog.Printf(errStdLogs,
						"Error pruning temporary upload for %q: %s", thumbSrc, gErr.Error())
				}

				if post.ParentID == 0 {
					catalogSrc := gcutil.GetThumbnailPath("catalog", fileSrc)
					if gErr = os.Remove(catalogSrc); gErr != nil {
						gclog.Printf(errStdLogs,
							"Error pruning temporary upload for %s: %s", catalogSrc, gErr.Error())
					}
				}
			}
		}
	}
}

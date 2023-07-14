package posting

import (
	"os"
	"path"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
)

func tempCleaner() {
	ticker := time.Tick(time.Minute * 5)
	for range ticker {
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

			thumbnail, catalogThumbnail := uploads.GetThumbnailFilenames(
				path.Join(systemCritical.DocumentRoot, board.Dir, "thumb", upload.Filename))
			if err = os.Remove(thumbnail); err != nil {
				gcutil.LogError(err).
					Str("subject", "tempUpload").
					Str("filePath", thumbnail).Send()
			}

			if post.IsTopPost {
				if err = os.Remove(catalogThumbnail); err != nil {
					gcutil.LogError(err).
						Str("subject", "tempUpload").
						Str("filePath", catalogThumbnail).Send()
				}
			}
		}
	}
}

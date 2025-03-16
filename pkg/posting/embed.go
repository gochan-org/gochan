package posting

import (
	"errors"
	"net/http"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/rs/zerolog"
)

var (
	ErrNoEmbedding       = errors.New("embedding is disabled on this board")
	ErrUnrecognizedEmbed = errors.New("unrecognized embed URL")
)

type EmbedVideo struct {
	VideoID     string
	Handler     string
	ThumbWidth  int
	ThumbHeight int
}

// AttachEmbed checks if the post contains an embedded media URL from the form (if applicable) and if it is valid.
// It returns true if the post contains an embedded media URL, an error if the URL is invalid or some other error occurred.
// It attaches the embed as a pseudo-upload in the database if the URL is valid.
func AttachEmbed(request *http.Request, post *gcsql.Post, boardCfg *config.BoardConfig, warnEv, errEv *zerolog.Event, requestOpts ...*gcsql.RequestOptions) (*gcsql.Upload, error) {
	url := request.PostFormValue("embed")
	if url == "" {
		return nil, nil
	}

	canEmbed := len(boardCfg.EmbedMatchers) > 0
	if !canEmbed {
		warnEv.Msg("Rejected a post with an embed URL on a board that doesn't allow it")
		return nil, ErrNoEmbedding
	}
	handlerID, videoID, err := boardCfg.GetEmbedVideoID(url)
	if err != nil {
		return nil, err
	}

	upload := &gcsql.Upload{
		Filename:         "embed:" + handlerID,
		OriginalFilename: videoID,
		PostID:           post.ID,
		ThumbnailWidth:   boardCfg.ThumbWidth,
		ThumbnailHeight:  boardCfg.ThumbHeight,
	}
	if err = post.AddAttachment(upload, requestOpts...); err != nil {
		errEv.Err(err).Caller().Msg("Failed to attach embed")
		return nil, err
	}

	return upload, nil
}

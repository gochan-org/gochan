package posting

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/rs/zerolog"
)

var (
	ErrNoMatchingEmbedHandler = errors.New("no matching handler for the embed URL")
	ErrNoEmbedding            = errors.New("embedding is disabled on this board")
	ErrUnrecognizedEmbed      = errors.New("unrecognized embed URL")
)

type EmbedVideo struct {
	VideoID     string
	Handler     string
	ThumbWidth  int
	ThumbHeight int
}

// CheckEmbed checks if the post contains an embedded media URL from the form (if applicable) and if it is valid.
// It returns true if the post contains an embedded media URL, an error if the URL is invalid or some other error occurred.
// It attaches the embed as a pseudo-upload in the database if the URL is valid.
func CheckEmbed(request *http.Request, post *gcsql.Post, boardCfg *config.BoardConfig, warnEv, errEv *zerolog.Event) (*gcsql.Upload, error) {
	url := request.PostFormValue("embed")
	if url == "" {
		return nil, nil
	}

	canEmbed := len(boardCfg.EmbedMatchers) > 0
	if !canEmbed {
		warnEv.Msg("Rejected a post with an embed URL on a board that doesn't allow it")
		return nil, ErrNoEmbedding
	}
	var re *regexp.Regexp
	submatchIndex := 1
	var filename string
	for m, matcher := range boardCfg.EmbedMatchers {
		re = regexp.MustCompile(matcher.URLRegex) // already checked in config
		matches := re.FindAllStringSubmatch(url, -1)
		if len(matches) == 1 {
			if matcher.VideoIDSubmatchIndex != nil {
				submatchIndex = *matcher.VideoIDSubmatchIndex
			}
			filename = "embed:" + m + ":" + matches[0][submatchIndex]
			return &gcsql.Upload{
				Filename: filename,
				PostID:   post.ID,
			}, nil
		}
	}

	return nil, ErrUnrecognizedEmbed
}

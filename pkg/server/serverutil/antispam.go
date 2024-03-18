package serverutil

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

// ValidReferer checks to make sure that the incoming request is from the same domain (or if debug mode is enabled)
func ValidReferer(request *http.Request, errEv ...*zerolog.Event) bool {
	referer := request.Referer()
	rURL, err := url.ParseRequestURI(referer)
	if err != nil {
		var ev *zerolog.Event
		if len(errEv) == 1 {
			ev = gcutil.LogError(err).Caller()
		} else {
			ev = errEv[0].Err(err).Caller()
		}
		ev.Str("referer", referer).Msg("Error parsing referer URL")
		return false
	}
	return strings.Index(rURL.Path, config.GetSystemCriticalConfig().WebRoot) == 0
}

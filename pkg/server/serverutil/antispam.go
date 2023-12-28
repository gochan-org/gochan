package serverutil

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

// ValidReferer checks to make sure that the incoming request is from the same domain (or if debug mode is enabled)
func ValidReferer(request *http.Request) bool {
	if config.VerboseMode() {
		return true
	}
	referer := request.Referer()
	rURL, err := url.ParseRequestURI(referer)
	if err != nil {
		gcutil.Logger().Err(err).
			Str("referer", referer).
			Msg("Error parsing referer URL")
		return false
	}
	return strings.Index(rURL.Path, config.GetSystemCriticalConfig().WebRoot) == 0
}

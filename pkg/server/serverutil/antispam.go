package serverutil

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/gochan-org/gochan/pkg/config"
)

var (
	ErrSpambot = errors.New("request looks like spam")
)

const (
	// NoReferer is returned when the request has no referer
	NoReferer RefererResult = iota
	// InvalidReferer is returned when the referer not a valid URL
	InvalidReferer
	// InternalReferer is returned when the request came from the same site as the server
	InternalReferer
	// ExternalReferer is returned when the request came from another site. It may or may not be be spam, depending on the context
	ExternalReferer
)

type RefererResult int

// CheckReferer checks to make sure that the incoming request is from the same domain
func CheckReferer(request *http.Request) (RefererResult, error) {
	referer := request.Referer()
	if referer == "" {
		return NoReferer, nil
	}

	rURL, err := url.ParseRequestURI(referer)
	if err != nil {
		return InvalidReferer, err
	}
	systemCriticalConfig := config.GetSystemCriticalConfig()
	siteURLBase := url.URL{
		Host: systemCriticalConfig.SiteHost,
	}
	var result RefererResult = ExternalReferer
	if rURL.Host == siteURLBase.Host {
		result = InternalReferer
	}
	return result, nil
}

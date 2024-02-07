package geoip

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

var (
	geoipHandlers = make(map[string]GeoIPHandler)
	activeHandler GeoIPHandler

	ErrInvalidIP     = errors.New("invalid IP address")
	ErrNotConfigured = errors.New("geoip is not configured")
	ErrUnrecognized  = errors.New("unrecognized GeoIP handler ID")
)

// Country represents the country data (or custom flag data) used by gochan.
// For posts set to use the poster's country, `Flag` is the country's
// abbreviation, and `Name` is the country name. If a custom flag is selected,
// Flag is the filename accessible in /static/flags/{flag}, and Name is the
// configured flag name. This package does not handle custom flag validation.
type Country struct {
	Flag string
	Name string
}

// IsGeoIP is true of the country has a recognized abbreviation set as its
// flag. Otherwise it is assumed to be either a custom flag, or no flag if
// the string is blank
func (c Country) IsGeoIP() bool {
	if c.Flag == "" || c.Name == "" {
		return false
	}
	if _, err := GetCountryName(c.Flag); err == nil {
		return true
	}
	return false
}

type GeoIPHandler interface {
	Init(options map[string]any) error
	GetCountry(request *http.Request, board string, errEv *zerolog.Event) (*Country, error)
	Close() error
}

// RegisterGeoIPHandler registers an object that can handle incoming posts, name allows it
// to be used as config.SiteConfig.GeoIP.Type
func RegisterGeoIPHandler(id string, cb GeoIPHandler) error {
	_, ok := geoipHandlers[id]
	if ok {
		return fmt.Errorf("a geoip handler has already been registered to the ID %q", id)
	}
	geoipHandlers[id] = cb
	return nil
}

// SetupGeoIP sets the handler to be used for GeoIP requests. If the ID has not been registered
// by a handler, it will return ErrUnrecognized
func SetupGeoIP(id string, options map[string]any) (err error) {
	if id == "" {
		// not using GeoIP
		return nil
	}
	var ok bool
	activeHandler, ok = geoipHandlers[id]
	if !ok {
		return ErrUnrecognized
	}

	return activeHandler.Init(options)
}

// GetCountry looks up the country the request comes from using the active handler.
// It throws ErrNotConfigured if one has not been configured
func GetCountry(request *http.Request, board string, errEv ...*zerolog.Event) (*Country, error) {
	if activeHandler == nil {
		return nil, ErrNotConfigured
	}
	var ev *zerolog.Event
	if errEv != nil {
		ev = errEv[0]
	} else {
		ev = gcutil.LogError(nil).
			Str("ip", gcutil.GetRealIP(request))
		defer ev.Discard()
	}
	return activeHandler.GetCountry(request, board, ev)
}

func Close() error {
	if activeHandler != nil {
		return activeHandler.Close()
	}
	return nil
}

func init() {
	mmdb = &mmdbHandler{}
	RegisterGeoIPHandler("mmdb", mmdb)
	RegisterGeoIPHandler("geoip2", mmdb)
	RegisterGeoIPHandler("geolite2", mmdb)
}

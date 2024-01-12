package geoip

import (
	"errors"
	"fmt"
	"net/http"
)

const (
	CustomFlag FlagType = iota
	GeoIPFlag
)

var (
	geoipHandlers = make(map[string]GeoIPHandler)
	activeHandler GeoIPHandler

	ErrInvalidIP     = errors.New("invalid IP address")
	ErrNotConfigured = errors.New("geoip is not configured")
	ErrUnrecognized  = errors.New("unrecognized GeoIP handler ID")
)

type FlagType int

type Country struct {
	Name string
	Flag string
}

func (c *Country) Type() FlagType {
	if _, ok := abbrMap[c.Flag]; ok {
		return GeoIPFlag
	}
	return CustomFlag
}

type GeoIPHandler interface {
	Init(options map[string]any) error
	GetCountry(request *http.Request, board string) (*Country, error)
	Close() error
}

// RegisterGeoIPHandler registers an object that can handle incoming posts, name allows it
// to be used as config.SiteConfig.GeoIP.Type
func RegisterGeoIPHandler(id string, cb GeoIPHandler) error {
	if id == "" {
		// not using GeoIP
		return nil
	}
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
	var ok bool
	activeHandler, ok = geoipHandlers[id]
	if !ok {
		return ErrUnrecognized
	}

	return activeHandler.Init(options)
}

func LookupCountry(request *http.Request, board string) (*Country, error) {
	if activeHandler == nil {
		return nil, ErrNotConfigured
	}
	return activeHandler.GetCountry(request, board)
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

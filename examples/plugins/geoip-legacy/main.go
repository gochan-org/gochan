package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	legacy "github.com/Eggbertx/geoip-legacy"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/geoip"
	"github.com/rs/zerolog"
)

const invalidValueFmt = "invalid %q value type %T, expected string"

var handler = &legacyHandler{}

type legacyHandler struct {
	db legacy.GeoIPDB
}

// Close implements geoip.GeoIPHandler.
func (l *legacyHandler) Close() error {
	if l.db != nil {
		return l.db.Close()
	}
	return nil
}

// GetCountry implements geoip.GeoIPHandler.
func (l *legacyHandler) GetCountry(request *http.Request, board string, errEv *zerolog.Event) (*geoip.Country, error) {
	var err error
	if l.db == nil {
		if err = l.Init(config.GetSiteConfig().GeoIPOptions); err != nil {
			return nil, err
		}
	}

	ip := gcutil.GetRealIP(request)
	result, err := l.db.GetCountryByAddr(ip)
	if err != nil {
		return nil, err
	}
	return &geoip.Country{
		Flag: result.Code,
		Name: result.NameUTF8,
	}, nil
}

// Init implements geoip.GeoIPHandler.
func (l *legacyHandler) Init(options map[string]any) error {
	var err error
	var v4db, v6db string
	var ok bool
	for key, val := range options {
		keyLower := strings.ToLower(key)
		switch keyLower {
		case "ipv4db":
			fallthrough
		case "ipv4dblocation":
			v4db, ok = val.(string)
			if !ok {
				return fmt.Errorf(invalidValueFmt, key, key)
			}
		case "ipv6db":
			fallthrough
		case "ipv6dblocation":
			v6db, ok = val.(string)
			if !ok {
				return fmt.Errorf(invalidValueFmt, key, key)
			}
		}
	}
	if v4db != "" && v6db != "" {
		// both databases provided, use CombinedDB
		l.db, err = legacy.OpenCombinedDB(v4db, v6db)
		return err
	} else if v4db != "" {
		// IPv4 db provided
		l.db, err = legacy.OpenDB(v4db, nil)
		return err
	} else if v6db != "" {
		// IPv6 db provided
		l.db, err = legacy.OpenDB(v4db, &legacy.GeoIPOptions{IsIPv6: true})
		return err
	}
	return errors.New("no legacy geoip databases provided, must provide 'ipv4DB' or 'ipv6DB' (or both) JSON keys in gochan.json")
}

func InitPlugin() error {
	return geoip.RegisterGeoIPHandler("legacy", handler)
}

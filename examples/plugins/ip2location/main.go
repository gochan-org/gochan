package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/geoip"
	"github.com/ip2location/ip2location-go/v9"
	"github.com/rs/zerolog"
)

var (
	i2ldb = &ip2locationDB{}
)

type ip2locationDB struct {
	db *ip2location.DB
}

// Close implements geoip.GeoIPHandler.
func (i *ip2locationDB) Close() error {
	if i.db == nil {
		return nil
	}
	i.db.Close()
	return nil
}

// GetCountry implements geoip.GeoIPHandler.
func (i *ip2locationDB) GetCountry(request *http.Request, board string, errEv *zerolog.Event) (*geoip.Country, error) {
	var err error
	if i.db == nil {
		if err = i.Init(config.GetSiteConfig().GeoIPOptions); err != nil {
			return nil, err
		}
	}

	country := &geoip.Country{}
	ip := gcutil.GetRealIP(request)

	record, err := i.db.Get_country_long(ip)
	if err != nil {
		return nil, err
	}
	country.Name = record.Country_long

	if record, err = i.db.Get_country_short(ip); err != nil {
		return nil, err
	}
	country.Flag = record.Country_short
	return country, nil
}

// Init implements geoip.GeoIPHandler.
func (i *ip2locationDB) Init(options map[string]any) (err error) {
	for key, val := range options {
		keyLower := strings.ToLower(key)
		switch keyLower {
		case "database":
			fallthrough
		case "dblocation":
			dbLocation, ok := val.(string)
			if !ok {
				return fmt.Errorf("invalid %q value type %T, expected string", key, key)
			}
			i.db, err = ip2location.OpenDB(dbLocation)
			return err
		}
	}
	return errors.New("missing 'database' JSON key in gochan.json")
}

func InitPlugin() error {
	return geoip.RegisterGeoIPHandler("ip2location", i2ldb)
}

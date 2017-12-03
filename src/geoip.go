package main

import (
	"github.com/nranchev/go-libGeoIP"
)

func getCountryCode(ip string) (string, error) {
	if config.EnableGeoIP && config.GeoIPDBlocation != "" {
		gi, err := libgeo.Load(config.GeoIPDBlocation)
		if err != nil {
			return "", err
		}
		return gi.GetLocationByIP(ip).CountryCode, nil
	}
	return "", nil
}

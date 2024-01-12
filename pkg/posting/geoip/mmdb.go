package geoip

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/oschwald/maxminddb-golang"
)

var (
	ErrMissingDBArg = errors.New("missing database argument")
	mmdb            *mmdbHandler
)

type mmdbRecord struct {
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
}

type mmdbHandler struct {
	db *maxminddb.Reader
}

func (mh *mmdbHandler) Init(options map[string]any) error {
	infoEv := gcutil.LogInfo()
	errEv := gcutil.LogError(nil)
	defer gcutil.LogDiscard(infoEv, errEv)
	gcutil.LogStr("geoipType", "mmdb", infoEv, errEv)
	if options == nil {
		errEv.Err(ErrMissingDBArg).Caller().Send()
		return ErrMissingDBArg
	}

	var dbLocation string
	var ok bool
	var err error
	for k, v := range options {
		key := strings.ToLower(k)
		if key == "dbLocation" || key == "database" || key == "mmdb" {
			dbLocation, ok = v.(string)
			if !ok {
				err = fmt.Errorf("invalid database argument (expected string, got %T)", v)
				errEv.Err(err).Caller().
					Interface("dbLocation", options["dbLocation"]).Send()
				return err
			}
			break
		}
	}
	if dbLocation == "" {
		errEv.Err(ErrMissingDBArg).Caller().Send()
		return ErrMissingDBArg
	}
	gcutil.LogStr("dbLocation", dbLocation, infoEv, errEv)

	if mh.db, err = maxminddb.Open(dbLocation); err != nil {
		errEv.Err(err).Caller().
			Str("dbLocation", dbLocation).Send()
		return err
	}
	infoEv.Msg("GeoIP initialized")
	return nil
}

func (mh *mmdbHandler) GetCountry(request *http.Request, board string) (*Country, error) {
	var err error
	if mh.db == nil {
		return nil, err
	}
	ip := net.ParseIP(gcutil.GetRealIP(request))
	if ip == nil {
		return nil, ErrInvalidIP
	}
	var record mmdbRecord
	err = mh.db.Lookup(ip, &record)
	if err != nil {
		return nil, err
	}
	return &Country{
		Name: record.Country.ISOCode,
		Flag: record.Country.Names["en"],
	}, nil
}

func (mh *mmdbHandler) Close() error {
	if mh.db != nil {
		return mh.db.Close()
	}
	return nil
}

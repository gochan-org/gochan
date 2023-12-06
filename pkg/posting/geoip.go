package posting

import (
	"errors"
	"net"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	maxminddb "github.com/oschwald/maxminddb-golang"
)

var (
	mmdb         *maxminddb.Reader
	ErrInvalidIP = errors.New("invalid IP address")
)

type mmdbRecord struct {
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
}

func InitGeoIP() {
	dbLocation := config.GetSiteConfig().GeoIPDBlocation
	if dbLocation == "" {
		return
	}
	var err error
	mmdb, err = maxminddb.Open(dbLocation)
	if err != nil {
		gcutil.LogFatal().Err(err).Send()
	}
}

func LookupCountry(post *gcsql.Post, board string) (abbr string, name string, err error) {
	boardCfg := config.GetBoardConfig(board)
	if !boardCfg.EnableGeoIP || mmdb == nil {
		return "", "", nil
	}
	ip := net.ParseIP(post.IP)
	if ip == nil {
		return "", "", ErrInvalidIP
	}
	var record mmdbRecord
	err = mmdb.Lookup(ip, &record)
	return record.Country.ISOCode, record.Country.Names["en"], err
}

func CloseGeoipDB() error {
	if mmdb == nil {
		return nil
	}
	return mmdb.Close()
}

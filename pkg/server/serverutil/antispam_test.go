package serverutil

import (
	"net/http"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/stretchr/testify/assert"
)

var (
	checkRefererTestCases = []checkRefererTestCase{
		{
			desc:           "Internal referer",
			referer:        "http://gochan.org",
			siteDomain:     "gochan.org",
			expectedResult: InternalReferer,
		},
		{
			desc:           "External referer",
			referer:        "http://somesketchysite.com",
			siteDomain:     "gochan.com",
			expectedResult: ExternalReferer,
		},
		{
			desc:           "No referer",
			siteDomain:     "gochan.org",
			expectedResult: NoReferer,
		},
		{
			desc:           "Internal referer with port",
			referer:        "http://127.0.0.1:8080",
			siteDomain:     "127.0.0.1:8080",
			expectedResult: InternalReferer,
		},
		{
			desc:           "Internal referer with port, IPv6",
			referer:        "http://[::1]:8080",
			siteDomain:     "[::1]:8080",
			expectedResult: InternalReferer,
		},
	}
)

type checkRefererTestCase struct {
	desc           string
	referer        string
	siteDomain     string
	expectedResult RefererResult
}

func TestCheckReferer(t *testing.T) {
	config.SetVersion("4.0.0")
	systemCriticalConfig := config.GetSystemCriticalConfig()
	req, err := http.NewRequest("GET", "http://gochan.org", nil)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	for _, tC := range checkRefererTestCases {
		t.Run(tC.desc, func(t *testing.T) {
			systemCriticalConfig.SiteDomain = tC.siteDomain
			config.SetSystemCriticalConfig(systemCriticalConfig)
			req.Header.Set("Referer", tC.referer)
			result, err := CheckReferer(req)
			assert.NoError(t, err)
			assert.Equal(t, tC.expectedResult, result)
		})
	}
}

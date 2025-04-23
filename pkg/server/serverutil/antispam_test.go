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
			siteHost:       "gochan.org",
			expectedResult: InternalReferer,
		},
		{
			desc:           "External referer",
			referer:        "http://somesketchysite.com",
			siteHost:       "gochan.com",
			expectedResult: ExternalReferer,
		},
		{
			desc:           "No referer",
			siteHost:       "gochan.org",
			expectedResult: NoReferer,
		},
		{
			desc:           "Internal referer with port",
			referer:        "http://127.0.0.1:8080",
			siteHost:       "127.0.0.1:8080",
			expectedResult: InternalReferer,
		},
		{
			desc:           "Internal referer with port, IPv6",
			referer:        "http://[::1]:8080",
			siteHost:       "[::1]:8080",
			expectedResult: InternalReferer,
		},
	}
)

type checkRefererTestCase struct {
	desc           string
	referer        string
	siteHost       string
	expectedResult RefererResult
}

func TestCheckReferer(t *testing.T) {
	config.InitTestConfig()
	systemCriticalConfig := config.GetSystemCriticalConfig()
	req, err := http.NewRequest("GET", "https://gochan.org", nil)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	for _, tC := range checkRefererTestCases {
		t.Run(tC.desc, func(t *testing.T) {
			systemCriticalConfig.SiteHost = tC.siteHost
			config.SetSystemCriticalConfig(systemCriticalConfig)
			req.Header.Set("Referer", tC.referer)
			result, err := CheckReferer(req)
			assert.NoError(t, err)
			assert.Equal(t, tC.expectedResult, result)
		})
	}
}

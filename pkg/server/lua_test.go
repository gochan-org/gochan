package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
)

var (
	staticFileHeadersTests = []staticFileHeadersTestCase{
		{
			desc:      "valid .pdf set",
			expectExt: ".pdf",
			luaScript: `local server = require("server")
server.register_ext_headers(".pdf", {
  ["Content-Type"] = "application/pdf",
  ["Cache-Control"] = "max-age=3600",
  ["X-Test"] = "test-value"
})`,
			expectedHeaders: StaticFileHeaders{
				ContentType:  "application/pdf",
				CacheControl: "max-age=3600",
				Other: map[string]string{
					"X-Test": "test-value",
				},
			},
		},
		{
			desc: "missing content-type",
			luaScript: `local server = require("server")
server.register_ext_headers(".pdf", {
  ["Cache-Control"] = "max-age=3600",
  ["X-Test"] = "test-value"
})`,
			expectError: true,
		},
	}
)

type staticFileHeadersTestCase struct {
	desc            string
	luaScript       string
	expectExt       string
	expectedHeaders StaticFileHeaders
	expectError     bool
}

func TestPreloadModule(t *testing.T) {
	for _, tc := range staticFileHeadersTests {
		t.Run(tc.desc, func(t *testing.T) {
			l := lua.NewState()
			defer l.Close()
			l.PreloadModule("server", PreloadModule)
			err := l.DoString(tc.luaScript)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				headers, exists := knownFileHeaders[tc.expectExt]
				if !assert.True(t, exists) {
					t.FailNow()
				}
				assert.Equal(t, tc.expectedHeaders, headers)
			}
		})
	}
}

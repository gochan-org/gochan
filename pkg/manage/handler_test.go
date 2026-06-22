package manage

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	_ "github.com/gochan-org/gochan/pkg/gcsql/initsql"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bunrouter"
)

var (
	panicked              bool = false
	testCustomTitleAction      = &Action{
		ID:    "test",
		Title: "Test Action",
		Callback: func(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, _ zerolog.Logger) (any, error) {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
				}
			}()
			SetCustomPageTitle(request, "Custom Title Set")
			return nil, nil
		},
	}
)

func TestSetCustomPageTitleHandleImpropertCaller(t *testing.T) {
	// when called outside of a manage action callback wrapped with setupManageFunction, SetCustomPageTitle should do nothing
	setupManageTestSuite(t, config.SQLConfig{})
	panicked = false
	req := bunrouter.NewRequest(httptest.NewRequest(http.MethodGet, "http://example.com", nil))

	responseWriter := httptest.NewRecorder()
	testCustomTitleAction.Callback(responseWriter, req.Request, &gcsql.Staff{}, false, zerolog.Nop())
	assert.False(t, panicked, "SetCustomPageTitle should not panic, even when improperly called")
	_, ok := req.Context().Value(customTitleContextKey{}).(*string)
	assert.False(t, ok, "custom title pointer should not be set yet")
}

func TestSetCustomPageTitle(t *testing.T) {
	setupManageTestSuite(t, config.SQLConfig{})
	panicked = false
	req := bunrouter.NewRequest(httptest.NewRequest(http.MethodGet, "http://example.com", nil))
	responseWriter := httptest.NewRecorder()
	setupActionCB := setupManageFunction(testCustomTitleAction)
	setupActionCB(responseWriter, req)
	assert.False(t, panicked)

	output := responseWriter.Body.String()
	assert.Contains(t, output, "<title>Custom Title Set - Gochan</title>", "page title should be set to custom title")
}

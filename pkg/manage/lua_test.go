package manage

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bunrouter"
	lua "github.com/yuin/gopher-lua"
)

var (
	luaRegisterActionTestCases = []luaRegisterActionTestCase{
		{
			desc:        "valid action registration via register_manage_page",
			luaScript:   `manage.register_manage_page("test_action", "Test Action", 1, 1, function() return "<h1>Test</h1>" end)`,
			expectError: false,
			expectAction: Action{
				ID:          "test_action",
				Title:       "Test Action",
				Permissions: 1,
				JSONoutput:  1,
			},
		},
		{
			desc: "valid action registration via register_staff_action",
			luaScript: `manage.register_staff_action({
				id = "test_action2",
				title = "Test Action",
				permissions = 1,
				json_output = 1,
				hidden = false,
				handler = function() return "<h1>Test</h1>" end
			})`,
			expectError: false,
			expectAction: Action{
				ID:          "test_action2",
				Title:       "Test Action",
				Permissions: 1,
				JSONoutput:  1,
			},
		},
		{
			desc:        "invalid action registration missing id",
			luaScript:   `manage.register_staff_action({ title = "Test Action", permissions = 1, json_output = 1 })`,
			expectError: true,
		},
		{
			desc:        "invalid action registration missing title",
			luaScript:   `manage.register_staff_action({ id = "test_action", permissions = 1, json_output = 1 })`,
			expectError: true,
		},
		{
			desc:        "invalid action registration invalid permissions type",
			luaScript:   `manage.register_staff_action({ id = "test_action", title = "Test Action", permissions = {}, json_output = 1 })`,
			expectError: true,
		},
		{
			desc:        "invalid action registration invalid json_output type",
			luaScript:   `manage.register_staff_action({ id = "test_action", title = "Test Action", permissions = 1, json_output = {} })`,
			expectError: true,
		},
	}
)

type luaRegisterActionTestCase struct {
	desc         string
	luaScript    string
	expectError  bool
	expectAction Action
}

func TestPreloadModule(t *testing.T) {
	config.InitTestConfig()
	for _, tc := range luaRegisterActionTestCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := lua.NewState()
			defer l.Close()
			l.PreloadModule("manage", PreloadModule)
			err := l.DoString("local manage = require('manage')\n" + tc.luaScript)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				var found bool
				for _, action := range actions {
					if action.ID == tc.expectAction.ID {
						found = true
						assert.Equal(t, tc.expectAction.Title, action.Title)
						assert.Equal(t, tc.expectAction.Permissions, action.Permissions)
						assert.Equal(t, tc.expectAction.JSONoutput, action.JSONoutput)
					}
				}
				assert.True(t, found)
			}
		})
	}
}

func TestSetCustomTitleLua(t *testing.T) {
	setupManageTestSuite(t, config.SQLConfig{})
	l := lua.NewState()
	defer l.Close()
	l.PreloadModule("manage", PreloadModule)
	err := l.DoString(`local manage = require("manage")
		manage.register_staff_action({
			id = "customtitle",
			title = "Test Custom Title",
			permissions = 0,
			json_output = 0,
			handler = function(_, request)
				manage.set_custom_page_title(request, "Title Set Via Lua")
				return "", nil
			end
		})
	`)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	var found bool
	for a, action := range actions {
		if action.ID == "customtitle" {
			actionCB := setupManageFunction(&actions[a])
			req := bunrouter.NewRequest(httptest.NewRequest(http.MethodGet, "http://example.com", nil))
			responseWriter := httptest.NewRecorder()
			actionCB(responseWriter, req)
			output := responseWriter.Body.String()
			assert.Contains(t, output, "<title>Title Set Via Lua - Gochan</title>", "page title should be set to custom title")
			found = true
			break
		}
	}
	assert.True(t, found)
}

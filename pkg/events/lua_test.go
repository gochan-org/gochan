package events

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func TestRegisterEventFromLua(t *testing.T) {
	tests := []struct {
		name             string
		luaStr           string
		expects          string
		expectsError     bool
		expectsRecovered bool
	}{
		{
			name: "register event",
			luaStr: `local events = require("events");
function event_handler(trigger, data)
	local _, err = buffer:WriteString("registered event from lua " .. data .. "\n");
	assert(err == nil);
end
events.register_event({"register_event_test","register_event_test2"}, event_handler);`,
			expects: "registered event from lua data\nregistered event from lua 2\n",
		},
		{
			name: "register event that returns error",
			luaStr: `local events = require("events");
events.register_event({"register_event_test","register_event_test2"}, function(trigger, data)
	return "uh oh";
end);`,
			expectsError: true,
		},
	}
	for _, tC := range tests {
		buf := new(bytes.Buffer)
		l := lua.NewState()
		l.SetGlobal("buffer", luar.New(l, buf))
		l.PreloadModule("events", PreloadModule)

		t.Run(tC.name, func(t *testing.T) {
			assert.NoError(t, l.DoString(tC.luaStr))

			handled, err, recovered := TriggerEvent("register_event_test", "data")
			assert.True(t, handled)
			assert.Equal(t, tC.expectsError, err != nil)
			assert.Equal(t, tC.expectsRecovered, recovered)

			handled, err, recovered = TriggerEvent("register_event_test2", 2)
			assert.True(t, handled)
			assert.Equal(t, tC.expectsError, err != nil)
			assert.Equal(t, tC.expectsRecovered, recovered)

			assert.Equal(t, tC.expects, buf.String())
		})
	}
}

func testEventHandler(trigger string, data ...any) error {
	if len(data) < 2 {
		panic("expected buffer and at least one other argument to event")
	}
	buf, ok := data[0].(*bytes.Buffer)
	if !ok || buf == nil {
		panic("expected first argument to be a non-nil *bytes.Buffer")
	}
	buf.WriteString(fmt.Sprintln(trigger, data[1:]))
	return nil
}

func TestTriggerEventFromLua(t *testing.T) {
	buf := new(bytes.Buffer)
	tests := []struct {
		name             string
		luaStr           string
		expects          string
		expectsRecovered bool
	}{
		{
			name: "register event",
			luaStr: `local events = require("events");
events.trigger_event("event1", buffer, 1);
events.trigger_event("event2", buffer, "a");`,
			expects: "event1 [1]\nevent2 [a]\n",
		},
	}
	for _, tC := range tests {
		buf.Reset()
		l := lua.NewState()
		l.SetGlobal("buffer", luar.New(l, buf))
		l.PreloadModule("events", PreloadModule)

		t.Run(tC.name, func(t *testing.T) {
			RegisterEvent([]string{"event1", "event2"}, testEventHandler)
			assert.NoError(t, l.DoString(tC.luaStr))
			assert.Equal(t, tC.expects, buf.String())
		})
	}
}

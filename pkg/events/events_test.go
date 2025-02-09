package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPanicRecover(t *testing.T) {
	RegisterEvent([]string{"TestPanicRecoverEvt"}, func(tr string, i ...any) error {
		t.Log("Testing panic recover")
		t.Log(i[0])
		return nil
	})
	handled, err, recovered := TriggerEvent("TestPanicRecoverEvt") // should panic

	assert.True(t, handled, "TriggerEvent for TestPanicRecoverEvt should be handled")
	assert.Nil(t, err)
	t.Log("TestPanicRecoverEvt recovered: ", recovered)
	assert.True(t, recovered, "TestPanicRecoverEvt should have caused a panic and recovered from it")
}

func TestEventEditValue(t *testing.T) {
	RegisterEvent([]string{"TestEventEditValue"}, func(tr string, i ...any) error {
		p := i[0].(*int)
		*p += 1
		return nil
	})
	var a int
	t.Logf("a before TestEventEditValue triggered: %d", a)
	TriggerEvent("TestEventEditValue", &a)
	assert.NotEqual(t, 0, a, "TestEventEditValue event should increment the pointer to an int passed to it when triggered")
	t.Logf("a after TestEventEditValue triggered: %d", a)
}

func TestMultipleEventTriggers(t *testing.T) {
	triggered := map[string]bool{}
	RegisterEvent([]string{"a", "b"}, func(tr string, i ...any) error {
		triggered[tr] = true
		return nil
	})
	TriggerEvent("a")
	TriggerEvent("b")
	aTriggered := triggered["a"]
	bTriggered := triggered["b"]
	assert.True(t, aTriggered, "'a' event should be triggered")
	assert.True(t, bTriggered, "'b' event should be triggered")
}

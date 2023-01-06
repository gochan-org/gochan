package events

import "testing"

func TestPanicRecover(t *testing.T) {
	RegisterEvent([]string{"TestPanicRecoverEvt"}, func(tr string, i ...interface{}) {
		t.Log("Testing panic recover")
		t.Log(i[0])
	})
	handled, recovered := TriggerEvent("TestPanicRecoverEvt") // should panic
	if !handled {
		t.Fatal("TriggerEvent for TestPanicRecoverEvt wasn't handled")
	}
	t.Log("TestPanicRecoverEvt recovered: ", recovered)
	if !recovered {
		t.Fatal("TestPanicRecoverEvt should have caused a panic and recovered from it")
	}
}

func TestEventEditValue(t *testing.T) {
	RegisterEvent([]string{"TestEventEditValue"}, func(tr string, i ...interface{}) {
		p := i[0].(*int)
		*p += 1
	})
	var a int
	t.Logf("a before TestEventEditValue triggered: %d", a)
	TriggerEvent("TestEventEditValue", &a)
	if a == 0 {
		t.Fatal("TestEventEditValue event didn't properly increment the pointer to a passed to it when triggered")
	}
	t.Logf("a after TestEventEditValue triggered: %d", a)
}

func TestMultipleEventTriggers(t *testing.T) {
	triggered := map[string]bool{}
	RegisterEvent([]string{"a", "b"}, func(tr string, i ...interface{}) {
		triggered[tr] = true
	})
	TriggerEvent("a")
	TriggerEvent("b")
	aTriggered := triggered["a"]
	bTriggered := triggered["b"]
	if !aTriggered {
		t.Fatal("a event not triggered")
	}
	if !bTriggered {
		t.Fatal("b event not triggered")
	}
}

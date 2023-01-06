package events

import (
	"fmt"
	"os"
	"strings"

	"github.com/gochan-org/gochan/pkg/gcutil"
)

var (
	registeredEvents map[string][]EventHandler
	testingMode      bool
)

type EventHandler func(string, ...interface{})

// RegisterEvent registers a new event handler to be called when any of the elements of triggers are passed
// to TriggerEvent
func RegisterEvent(triggers []string, handler func(trigger string, i ...interface{})) {
	for _, t := range triggers {
		registeredEvents[t] = append(registeredEvents[t], handler)
	}
}

// TriggerEvent triggers the event handler registered to trigger
func TriggerEvent(trigger string, data ...interface{}) (handled bool, recovered bool) {
	errEv := gcutil.LogError(nil).Caller(1)
	defer func() {
		if a := recover(); a != nil {
			if !testingMode {
				errEv.Err(fmt.Errorf("%s", a)).
					Str("event", trigger).
					Msg("Recovered from panic while handling event")
			}
			handled = true
			recovered = true
		}
	}()
	for _, handler := range registeredEvents[trigger] {
		handler(trigger, data...)
		handled = true
	}
	errEv.Discard()
	return
}

func init() {
	registeredEvents = map[string][]EventHandler{}
	testingMode = strings.HasSuffix(os.Args[0], ".test")
}

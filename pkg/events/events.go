package events

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gochan-org/gochan/pkg/gcutil"
)

var (
	registeredEvents        map[string][]EventHandler
	testingMode             bool
	ErrRecovered            = errors.New("recovered from a panic in event handler")
	InvalidArgumentErrorStr = "invalid argument(s) passed to event %q"
)

type EventHandler func(string, ...interface{}) error

// RegisterEvent registers a new event handler to be called when any of the elements of triggers are passed
// to TriggerEvent
func RegisterEvent(triggers []string, handler func(trigger string, i ...interface{}) error) {
	for _, t := range triggers {
		registeredEvents[t] = append(registeredEvents[t], handler)
	}
}

// TriggerEvent triggers the event handler registered to trigger
func TriggerEvent(trigger string, data ...interface{}) (handled bool, err error, recovered bool) {
	errEv := gcutil.LogError(nil).Caller(1)
	defer func() {
		if a := recover(); a != nil {
			errEv.Err(fmt.Errorf("%v", a)).
				Str("event", trigger).
				Msg("Recovered from panic while handling event")
			handled = true
			recovered = true
		}
		errEv.Discard()
	}()
	for _, handler := range registeredEvents[trigger] {
		if err = handler(trigger, data...); err != nil {
			handled = true
			break
		}
		handled = true
	}
	return
}

func init() {
	registeredEvents = map[string][]EventHandler{}
	testingMode = strings.HasSuffix(os.Args[0], ".test")
}

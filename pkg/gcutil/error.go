package gcutil

import (
	"encoding/json"
)

var (
	// ErrNotImplemented should be used for unimplemented functionality when necessary
	ErrNotImplemented = NewError("Not implemented", true)
)

// GcError is an error type that can be rendered as a regular string or a JSON string
// and can be treated as a regular error if needed
type GcError struct {
	Message   string     `json:"error"`
	UserError bool       `json:"userError"`
	SubErrors []*GcError `json:"childErrors"`
}

// NewError creates a new GcError from a string
func NewError(message string, userCaused bool) *GcError {
	if message == "" {
		return nil
	}
	return &GcError{message, userCaused, nil}
}

// FromError creates a new GcError from an error type
func FromError(err error, userCaused bool) *GcError {
	if err == nil {
		return nil
	}
	return &GcError{err.Error(), userCaused, nil}
}

// JoinErrors joins multiple GcErrors into a single GcError, with the first one as the parent
func JoinErrors(gce ...*GcError) *GcError {
	if len(gce) == 0 {
		return nil
	}
	parent := gce[0]
	for e := range gce {
		if gce[e] != nil && e > 0 {
			parent.SubErrors = append(parent.SubErrors, gce[e])
		}
	}
	return parent
}

func (gce *GcError) addError(user bool, err ...interface{}) {
	for _, eI := range err {
		if err == nil {
			continue
		}
		eStr, ok := eI.(string)
		if ok {
			gce.SubErrors = append(gce.SubErrors, &GcError{eStr, user, nil})
			continue
		}
		eErr, ok := eI.(error)
		if ok {
			gce.SubErrors = append(gce.SubErrors, &GcError{eErr.Error(), user, nil})
		}
	}
}

// AddChildError creates a new GcError object, adds it to the SubErrors array, and returns it
func (gce *GcError) AddChildError(err error, userCaused bool) *GcError {
	if err == nil {
		return nil
	}
	child := FromError(err, userCaused)
	gce.SubErrors = append(gce.SubErrors, child)
	return child
}

// AddSystemError adds adds the supplied errors (can be strings or errors) as child errors
func (gce *GcError) AddSystemError(err ...interface{}) {
	gce.addError(false, err...)
}

// AddUserError adds adds the supplied errors (can be strings or errors) as child errors
func (gce *GcError) AddUserError(err ...interface{}) {
	gce.addError(true, err...)
}

func (gce GcError) Error() string {
	return gce.Message
}

// JSON returns a JSON string representing the error
func (gce *GcError) JSON() string {
	ba, _ := json.Marshal(gce)
	return string(ba)
}

// CompareErrors returns true if the given errors have the same error message
func CompareErrors(err1 error, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}
	if err1 != nil {
		return err1.Error() == err2.Error()
	}
	return false
}

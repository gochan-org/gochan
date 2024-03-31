package testutil

import (
	"bytes"
	"net/http"
)

// MockResponseWriter can be used in place of a http.ResponseWriter interface for tests
type MockResponseWriter struct {
	StatusCode int
	Buffer     *bytes.Buffer
	header     http.Header
}

func (m MockResponseWriter) Header() http.Header {
	PanicIfNotTest()
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header

}

func (m MockResponseWriter) Write(ba []byte) (int, error) {
	PanicIfNotTest()
	if m.Buffer == nil {
		m.Buffer = new(bytes.Buffer)
	}
	return m.Buffer.Write(ba)
}

func (m MockResponseWriter) WriteHeader(statusCode int) {
	PanicIfNotTest()
	m.StatusCode = statusCode
}

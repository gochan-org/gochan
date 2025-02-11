package serverutil

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	isRequestingJSONTestCases = []isRequestingJSONTestCase{
		{
			val: "1",
			exp: true,
		},
		{
			val: "on",
			exp: false,
		},
		{
			val: "true",
			exp: true,
		},
		{
			val: "yes",
			exp: false,
		},
	}
)

type isRequestingJSONTestCase struct {
	val string
	exp bool
}

func TestIsRequestingJSON(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	assert.False(t, IsRequestingJSON(req))
	for _, tc := range isRequestingJSONTestCases {
		t.Run("GET "+tc.val, func(t *testing.T) {
			req.Form.Set("json", tc.val)
			assert.Equal(t, tc.exp, IsRequestingJSON(req))
			req.Form.Del("json")
		})
		req.Method = "POST"
		req.PostFormValue("_")
		t.Run("POST "+tc.val, func(t *testing.T) {
			req.PostForm.Set("json", tc.val)
			assert.Equal(t, tc.exp, IsRequestingJSON(req))
			req.PostForm.Del("json")
		})
	}
}

type testResponseWriter struct {
	header http.Header
	status int
}

func (w *testResponseWriter) Header() http.Header {
	return w.header
}
func (w *testResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}
func (w *testResponseWriter) WriteHeader(s int) {
	w.status = s
}

func TestDeleteCookie(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	writer := testResponseWriter{
		header: make(http.Header),
	}
	assert.False(t, DeleteCookie(&writer, req, "test"))
	cookie := &http.Cookie{
		Name:    "test",
		Value:   "test",
		MaxAge:  90,
		Expires: time.Now().Add(7 * 24 * time.Hour),
	}
	req.AddCookie(cookie)

	assert.True(t, DeleteCookie(&writer, req, "test"))
	cookieExpireStr := writer.header.Get("Set-Cookie")

	ct, err := time.ParseInLocation(time.RFC1123, cookieExpireStr[strings.Index(cookieExpireStr, "Expires=")+8:], time.Local)
	assert.NoError(t, err)
	assert.True(t, ct.Before(time.Now().Add(-7*time.Hour)))
}

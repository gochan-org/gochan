package gcutil

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	failedPostIndentJSON = `{
	"action": "post",
	"message": "Post failed",
	"success": false
}`
	failedPostMinifiedJSON = `{"action":"post","message":"Post failed","success":false}`
	madePostIndentJSON     = `{
	"action": "post",
	"board": "test",
	"post": "12345#12346",
	"success": true
}`
	madePostMinifiedJSON = `{"action":"post","board":"test","post":"12345#12346","success":true}`
)

func TestMarshalJSON(t *testing.T) {
	testCases := []struct {
		desc              string
		postData          any
		expectedNonIndent string
		expectedIndent    string
		err               bool
	}{
		{
			desc:     "unmarshallable returns error",
			err:      true,
			postData: func() {},
		},
		{
			desc: "failed post data",
			postData: map[string]any{
				"action":  "post",
				"success": false,
				"message": errors.New("Post failed").Error(),
			},
			expectedNonIndent: failedPostMinifiedJSON,
			expectedIndent:    failedPostIndentJSON,
		},
		{
			desc: "successful post data",
			postData: map[string]any{
				"action":  "post",
				"success": true,
				"board":   "test",
				"post":    "12345#12346", // JS converts this to /test/res/12345.html#123456
			},
			expectedNonIndent: madePostMinifiedJSON,
			expectedIndent:    madePostIndentJSON,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			out, err := MarshalJSON(tC.postData, false)
			if tC.err {
				assert.Error(t, err)
				_, err = MarshalJSON(tC.postData, true)
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tC.expectedNonIndent, out)
			out, err = MarshalJSON(tC.postData, true)
			assert.NoError(t, err)
			assert.Equal(t, tC.expectedIndent, out)
		})
	}
}

func TestGetRealIP(t *testing.T) {
	const remoteAddr = "192.168.56.1"
	const testIP = "192.168.56.2"
	const cfIP = "192.168.56.3"
	const forwardedIP = "192.168.56.4"
	req := &http.Request{
		RemoteAddr: remoteAddr,
		Header:     make(http.Header),
	}
	assert.Equal(t, remoteAddr, GetRealIP(req))

	req.Header.Set("X-Forwarded-For", forwardedIP)
	assert.Equal(t, forwardedIP, GetRealIP(req))

	req.Header.Set("HTTP_CF_CONNECTING_IP", cfIP)
	assert.Equal(t, cfIP, GetRealIP(req))

	t.Setenv(TestingIPEnvVar, testIP)
	assert.Equal(t, testIP, GetRealIP(req))
}

func TestHackyStringToInt(t *testing.T) {
	i := HackyStringToInt("not an int")
	assert.Zero(t, i)
	i = HackyStringToInt("32")
	assert.NotZero(t, i)
}

func TestRandomString(t *testing.T) {
	var str string
	for i := 0; i < 255; i++ {
		str = RandomString(i)
		assert.Equal(t, i, len(str))
	}
}

func TestStripHTML(t *testing.T) {
	testCases := []struct {
		desc     string
		htmlIn   string
		expected string
	}{
		{
			desc:     "properly escape and strip HTML",
			htmlIn:   `<a href="#">&gt;implying</a>`,
			expected: "&gt;implying",
		},
		{
			desc:     "prevent JavaScript injection",
			htmlIn:   `<script>alert("Hello")</script>`,
			expected: "alert(&#34;Hello&#34;)",
		},
		{
			desc:     "don't error on unclosed tag",
			htmlIn:   `<div>unclosed tag`,
			expected: "unclosed tag",
		},
		{
			desc:     "empty element in, empty string out",
			htmlIn:   "<div></div>",
			expected: "",
		},
		{
			desc: "empty string in, empty string out",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			assert.Equal(t, tC.expected, StripHTML(tC.htmlIn))
		})
	}
}

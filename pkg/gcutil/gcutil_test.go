package gcutil

import (
	"errors"
	"net/http"
	"os"
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
	_, err := MarshalJSON(func() {}, false)
	assert.Error(t, err)
	failedPostData := map[string]interface{}{
		"action":  "post",
		"success": false,
		"message": errors.New("Post failed").Error(),
	}
	failedPost, err := MarshalJSON(failedPostData, false)
	assert.NoError(t, err)
	assert.Equal(t, failedPostMinifiedJSON, failedPost)
	failedPost, err = MarshalJSON(failedPostData, true)
	assert.NoError(t, err)
	assert.Equal(t, failedPostIndentJSON, failedPost)

	madePostData := map[string]interface{}{
		"action":  "post",
		"success": true,
		"board":   "test",
		"post":    "12345#12346", // JS converts this to /test/res/12345.html#123456
	}
	madePost, err := MarshalJSON(madePostData, false)
	assert.NoError(t, err)
	assert.Equal(t, madePostMinifiedJSON, madePost)
	madePost, err = MarshalJSON(madePostData, true)
	assert.NoError(t, err)
	assert.Equal(t, madePostIndentJSON, madePost)
}

func TestNameParsing(t *testing.T) {
	name, trip := ParseName("Name#Trip")
	assert.Equal(t, "Name", name)
	assert.Equal(t, "piec1MorXg", trip)
	name, trip = ParseName("#Trip")
	assert.Equal(t, "", name)
	assert.Equal(t, "piec1MorXg", trip)
	name, trip = ParseName("Name")
	assert.Equal(t, "Name", name)
	assert.Equal(t, "", trip)
	name, trip = ParseName("Name#")
	assert.Equal(t, "Name", name)
	assert.Equal(t, "", trip)
	name, trip = ParseName("#")
	assert.Equal(t, "", name)
	assert.Equal(t, "", trip)
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

	os.Setenv("GC_TESTIP", testIP)
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
	htmlIn := `<a href="#">&gt;implying</a>`
	stripped := StripHTML(htmlIn)
	assert.Equal(t, "&gt;implying", stripped)

	htmlIn = `<script>alert("Hello")</script>`
	stripped = StripHTML(htmlIn)
	assert.Equal(t, "alert(&#34;Hello&#34;)", stripped)

	htmlIn = `<div>unclosed tag`
	stripped = StripHTML(htmlIn)
	assert.Equal(t, "unclosed tag", stripped)

	htmlIn = "<div></div>"
	stripped = StripHTML(htmlIn)
	assert.Equal(t, "", stripped)

	htmlIn = ""
	stripped = StripHTML(htmlIn)
	assert.Equal(t, "", stripped)
}

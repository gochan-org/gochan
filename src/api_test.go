package main

import (
	"errors"
	"testing"
)

func TestAPI(t *testing.T) {
	failedPost, _ := marshalJSON(map[string]interface{}{
		"action":  "post",
		"success": false,
		"message": errors.New("Post failed").Error(),
	}, true)

	madePost, _ := marshalJSON(map[string]interface{}{
		"action":  "post",
		"success": true,
		"board":   "test",
		"post":    "12345#12346", // JS converts this to /test/res/12345.html#123456
	}, true)

	t.Log(
		"failedPost:", failedPost,
		"\nmadePost:", madePost,
	)
}

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
)

func TestServeJSON(t *testing.T) {
	// Set up a mock configuration
	config.SetMockConfig()

	writer := httptest.NewRecorder()
	data := map[string]interface{}{
		"status":    "success",
		"postID":    "777",
		"srcBoard":  "srcBoard",
		"destBoard": "destBoard",
	}

	ServeJSON(writer, data)

	// Check the status code
	if writer.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, writer.Code)
	}

	// Check the content type
	if writer.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected content type application/json, got %s", writer.Header().Get("Content-Type"))
	}

	// Check the response body
	var response map[string]interface{}
	if err := json.Unmarshal(writer.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Debugging output
	for key, value := range response {
		t.Logf("Response key: %s, value: %v", key, value)
	}

	for key, expectedValue := range data {
		actualValue := response[key]
		if actualValue != expectedValue {
			t.Errorf("expected %s to be %v, got %v", key, expectedValue, actualValue)
		} else {
			t.Logf("Match for %s: expected %v, got %v", key, expectedValue, actualValue)
		}
	}
}

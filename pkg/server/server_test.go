package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	_ "github.com/gochan-org/gochan/pkg/gcsql/initsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	_ "github.com/gochan-org/gochan/pkg/posting/uploads/inituploads"

	"github.com/stretchr/testify/assert"
)

func TestServeJSON(t *testing.T) {
	_, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		t.Fatalf("Failed to get current working directory: %v", err)
		return
	}

	config.InitTestConfig()
	config.SetRandomSeed("test")

	writer := httptest.NewRecorder()
	data := map[string]any{
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
	var response map[string]any
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

func TestServeErrorPage(t *testing.T) {
	_, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err, "Unable to switch to gochan root directory") {
		return
	}
	config.InitTestConfig()
	config.SetTestTemplateDir("templates")
	if !assert.NoError(t, gctemplates.InitTemplates()) {
		return
	}

	// Set writer and error string message
	writer := httptest.NewRecorder()
	errorMsg := "Unexpected error has occurred."

	ServeErrorPage(writer, errorMsg)

	body := writer.Body.String()
	t.Log("=============")
	t.Log("Response Body:", body)
	t.Log("=============")

	// Check response code & content-type
	assert.Equal(t, http.StatusOK, writer.Code)
	assert.Equal(t, "text/html; charset=utf-8", writer.Header().Get("Content-Type"))

	// Check the response body for the error message
	assert.Contains(t, body, errorMsg)
	assert.Contains(t, body, "Error")
}

func TestServeError(t *testing.T) {
	config.InitTestConfig()

	tests := []struct {
		name      string
		err       string
		wantsJSON bool
		data      map[string]any
		expected  string
	}{
		{
			name:      "JSON response with error",
			err:       "some error occurred",
			wantsJSON: true,
			data:      nil,
			expected:  `some error occurred`,
		},
		{
			name:      "JSON response with existing data",
			err:       "another error occurred",
			wantsJSON: true,
			data:      map[string]any{"info": "some info"},
			expected:  `another error occurred`,
		},
		{
			name:      "Non-JSON response",
			err:       "page not found",
			wantsJSON: false,
			data:      nil,
			expected:  "<!doctype html><meta charset=utf-8><title>Error :c</title><h1>Error</h1><p>page not found<hr><address>Site powered by Gochan " + config.GochanVersion + "</address>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a response recorder to capture the response
			rr := httptest.NewRecorder()
			// Call the ServeError function
			ServeError(rr, tt.err, tt.wantsJSON, tt.data)

			// Check the response
			if tt.wantsJSON {
				// Check if the response is JSON
				var responseMap map[string]any
				if err := json.NewDecoder(rr.Body).Decode(&responseMap); err != nil {
					t.Fatalf("Failed to decode JSON response: %v", err)
				}

				// Check if the expected error is present
				if responseMap["error"] != tt.expected {
					t.Errorf("Expected error %v, got %v", tt.expected, responseMap["error"])
				}
			} else if rr.Body.String() != tt.expected {
				// Check if the response body matches the expected error message
				t.Errorf("Expected response %v, got %v", tt.expected, rr.Body.String())
			}
		})
	}
}

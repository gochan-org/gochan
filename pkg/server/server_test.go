package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/stretchr/testify/assert"
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

func TestServeErrorPage(t *testing.T) {
	// Set up a mock configuration
	config.SetMockConfig()

	// Set writer and error string message
	writer := httptest.NewRecorder()
	err := "Unexpected error has occurred."

	ServeErrorPage(writer, err)

	body := writer.Body.String()
	t.Log("=============")
	t.Log("Response Body:", body)
	t.Log("=============")

	// Check response code & content-type
	assert.Equal(t, http.StatusOK, writer.Code)
	assert.Equal(t, "text/html; charset=utf-8", writer.Header().Get("Content-Type"))

	// Check the response body for the error message
	//assert.Contains(t, body, err)       Check if the body contains the error message
	//assert.Contains(t, body, "Error")   Check if the body contains the error title or header
}

func TestServeError(t *testing.T) {
	// Set up a mock configuration
	config.SetMockConfig()

	tests := []struct {
		name      string
		err       string
		wantsJSON bool
		data      map[string]interface{}
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
			data:      map[string]interface{}{"info": "some info"},
			expected:  `another error occurred`,
		},
		{
			name:      "Non-JSON response",
			err:       "page not found",
			wantsJSON: false,
			data:      nil,
			expected:  "", // Should we expect something ?
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
				var responseMap map[string]interface{}
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

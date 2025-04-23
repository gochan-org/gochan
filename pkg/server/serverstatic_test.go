package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestServeFile(t *testing.T) {
	config.InitTestConfig()

	tempDir, err := os.MkdirTemp("", "testservefile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Set the DocumentRoot in the mock config
	sysConfig := config.GetSystemCriticalConfig()
	sysConfig.DocumentRoot = tempDir
	sysConfig.WebRoot = "/"

	// Create a test file
	testFileName := "testfile.txt"
	testFilePath := path.Join(tempDir, testFileName)
	err = os.WriteFile(testFilePath, []byte("Hello, World!"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/"+testFileName, nil)
	rr := httptest.NewRecorder()

	serveFile(rr, req)

	res := rr.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Hello, World!", string(body))
}

func TestServeFile_NotFound(t *testing.T) {
	config.InitTestConfig()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "testservefile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir) // Clean up after the test

	// Set the DocumentRoot in the mock config
	sysConfig := config.GetSystemCriticalConfig() // This should return a pointer now
	sysConfig.DocumentRoot = tempDir
	sysConfig.WebRoot = "/"

	// Create a request for a non-existent file
	req := httptest.NewRequest("GET", "/nonexistentfile.txt", nil)
	rr := httptest.NewRecorder()

	// Call the serveFile function
	serveFile(rr, req)

	// Check the response
	res := rr.Result()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestSetFileHeaders(t *testing.T) {
	tests := []struct {
		filename      string
		expectedType  string
		expectedCache string
	}{
		{"image.png", "image/png", "max-age=86400"},
		{"image.gif", "image/gif", "max-age=86400"},
		{"image.jpg", "image/jpeg", "max-age=86400"},
		{"image.jpeg", "image/jpeg", "max-age=86400"},
		{"style.css", "text/css", "max-age=43200"},
		{"script.js", "text/javascript", "max-age=43200"},
		{"data.json", "application/json", "max-age=5, must-revalidate"},
		{"video.webm", "video/webm", "max-age=86400"},
		{"index.html", "text/html", "max-age=5, must-revalidate"},
		{"index.htm", "text/html", "max-age=5, must-revalidate"},
		{"unknownfile.xyz", "application/octet-stream", "max-age=86400"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			rr := httptest.NewRecorder()

			setFileHeaders(tt.filename, rr)

			assert.Equal(t, tt.expectedType, rr.Header().Get("Content-Type"))
			assert.Equal(t, tt.expectedCache, rr.Header().Get("Cache-Control"))
		})
	}
}

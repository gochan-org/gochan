package gcutil

import (
	"crypto/md5"
	"crypto/sha1" // skipcq GSC-G505
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/aquilax/tripcode"
	"golang.org/x/crypto/bcrypt"
	x_html "golang.org/x/net/html"
)

const (
	// DefaultMaxAge is used for cookies that have an invalid or unset max age (default is 1 month)
	DefaultMaxAge = 60 * 60 * 24 * 31
)

var (
	// ErrNotImplemented should be used for unimplemented functionality when necessary, not for bugs
	ErrNotImplemented = errors.New("not implemented")
)

// BcryptSum generates and returns a checksum using the bcrypt hashing function
func BcryptSum(str string) string {
	digest, err := bcrypt.GenerateFromPassword([]byte(str), 10)
	if err == nil {
		return string(digest)
	}
	return ""
}

// Md5Sum generates and returns a checksum using the MD5 hashing function
func Md5Sum(str string) string {
	hash := md5.New() // skipcq: GSC-G401
	io.WriteString(hash, str)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// Sha1Sum generates and returns a checksum using the SHA-1 hashing function
func Sha1Sum(str string) string {
	hash := sha1.New() // skipcq: GSC-G401
	io.WriteString(hash, str)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// CloseHandle closes the given closer object only if it is non-nil
func CloseHandle(handle io.Closer) {
	if handle != nil {
		handle.Close()
	}
}

// DeleteMatchingFiles deletes files in a folder (root) that match a given regular expression.
// Returns the number of files that were deleted, and any error encountered.
func DeleteMatchingFiles(root, match string) (filesDeleted int, err error) {
	files, err := os.ReadDir(root)
	if err != nil {
		return 0, err
	}
	for _, f := range files {
		match, _ := regexp.MatchString(match, f.Name())
		if match {
			os.Remove(filepath.Join(root, f.Name()))
			filesDeleted++
		}
	}
	return filesDeleted, err
}

// FindResource searches for a file in the given paths and returns the first one it finds
// or a blank string if none of the paths exist
func FindResource(paths ...string) string {
	var err error
	for _, filepath := range paths {
		if _, err = os.Stat(filepath); err == nil {
			return filepath
		}
	}
	return ""
}

// GetFileParts returns the base filename, the filename sans-extension, and the extension
// sans-filename
func GetFileParts(filename string) (string, string, string) {
	base := path.Base(filename)
	var noExt string
	var ext string
	lastIndex := strings.LastIndex(base, ".")
	if lastIndex > -1 {
		noExt = base[:strings.LastIndex(base, ".")]
		ext = path.Ext(base)[1:]
	}
	return base, noExt, ext
}

// GetFormattedFilesize returns a human readable filesize
func GetFormattedFilesize(size float64) string {
	if size < 1000 {
		return fmt.Sprintf("%dB", int(size))
	} else if size <= 100000 {
		return fmt.Sprintf("%fKB", size/1024)
	} else if size <= 100000000 {
		return fmt.Sprintf("%fMB", size/1024.0/1024.0)
	}
	return fmt.Sprintf("%0.2fGB", size/1024.0/1024.0/1024.0)
}

// GetRealIP checks the HTTP_CF_CONNCTING_IP and X-Forwarded-For HTTP headers to get a
// potentially obfuscated IP address, before getting the requests reported remote address
// if neither header is set
func GetRealIP(request *http.Request) string {
	if request.Header.Get("HTTP_CF_CONNECTING_IP") != "" {
		return request.Header.Get("HTTP_CF_CONNECTING_IP")
	}
	if request.Header.Get("X-Forwarded-For") != "" {
		return request.Header.Get("X-Forwarded-For")
	}
	remoteHost, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return request.RemoteAddr
	}
	return remoteHost
}

// HackyStringToInt parses a string to an int, or 0 if error
func HackyStringToInt(text string) int {
	value, _ := strconv.Atoi(text)
	return value
}

// MarshalJSON creates a JSON string with the given data and returns the string and any errors
func MarshalJSON(data interface{}, indent bool) (string, error) {
	var jsonBytes []byte
	var err error

	if indent {
		jsonBytes, err = json.MarshalIndent(data, "", "	")
	} else {
		jsonBytes, err = json.Marshal(data)
	}

	if err != nil {
		jsonBytes, _ = json.Marshal(map[string]string{"error": err.Error()})
	}
	return string(jsonBytes), err
}

// ParseName takes a name string from a request object and returns the name and tripcode parts
func ParseName(name string) (string, string) {
	var namePart string
	var tripcodePart string
	if !strings.Contains(name, "#") {
		namePart = name
	} else if strings.Index(name, "#") == 0 {
		tripcodePart = tripcode.Tripcode(name[1:])
	} else if strings.Index(name, "#") > 0 {
		postNameArr := strings.SplitN(name, "#", 2)
		namePart = postNameArr[0]
		tripcodePart = tripcode.Tripcode(postNameArr[1])
	}
	return namePart, tripcodePart
}

// RandomString returns a randomly generated string of the given length
func RandomString(length int) string {
	var str string
	for i := 0; i < length; i++ {
		num := rand.Intn(127) // skipcq: GSC-G404
		if num < 32 {
			num += 32
		}
		str += fmt.Sprintf("%c", num)
	}
	return str
}

func StripHTML(htmlIn string) string {
	dom := x_html.NewTokenizer(strings.NewReader(htmlIn))
	for tokenType := dom.Next(); tokenType != x_html.ErrorToken; {
		if tokenType != x_html.TextToken {
			tokenType = dom.Next()
			continue
		}
		txtContent := strings.TrimSpace(x_html.UnescapeString(string(dom.Text())))
		if len(txtContent) > 0 {
			return x_html.EscapeString(txtContent)
		}
		tokenType = dom.Next()
	}
	return ""
}

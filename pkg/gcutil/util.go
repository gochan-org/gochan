package gcutil

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aquilax/tripcode"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrNotImplemented should be used for unimplemented functionality when necessary
	ErrNotImplemented        = errors.New("Not implemented")
	ErrEmptyDurationString   = errors.New("Empty Duration string")
	ErrInvalidDurationString = errors.New("Invalid Duration string")
	durationRegexp           = regexp.MustCompile(`^((\d+)\s?ye?a?r?s?)?\s?((\d+)\s?mon?t?h?s?)?\s?((\d+)\s?we?e?k?s?)?\s?((\d+)\s?da?y?s?)?\s?((\d+)\s?ho?u?r?s?)?\s?((\d+)\s?mi?n?u?t?e?s?)?\s?((\d+)\s?s?e?c?o?n?d?s?)?$`)
)

// BcryptSum generates and returns a checksum using the bcrypt hashing function
func BcryptSum(str string) string {
	digest, err := bcrypt.GenerateFromPassword([]byte(str), 4)
	if err == nil {
		return string(digest)
	}
	return ""
}

// Md5Sum generates and returns a checksum using the MD5 hashing function
func Md5Sum(str string) string {
	hash := md5.New()
	io.WriteString(hash, str)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// Sha1Sum generates and returns a checksum using the SHA-1 hashing function
func Sha1Sum(str string) string {
	hash := sha1.New()
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
	files, err := ioutil.ReadDir(root)
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

// GetFileExtension returns the given file's extension, or a blank string if it has none
func GetFileExtension(filename string) string {
	if !strings.Contains(filename, ".") {
		return ""
	}

	return filename[strings.LastIndex(filename, ".")+1:]
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

// GetThumbnailPath returns the thumbnail path of the given filename
func GetThumbnailPath(thumbType string, img string) string {
	filetype := strings.ToLower(img[strings.LastIndex(img, ".")+1:])
	if filetype == "gif" || filetype == "webm" || filetype == "mp4" {
		filetype = "jpg"
	}
	index := strings.LastIndex(img, ".")
	if index < 0 || index > len(img) {
		return ""
	}
	thumbSuffix := "t." + filetype
	if thumbType == "catalog" {
		thumbSuffix = "c." + filetype
	}
	return img[0:index] + thumbSuffix
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

// ParseDurationString parses the given string into a duration and returns any errors
// based on TinyBoard's parse_time function
func ParseDurationString(str string) (time.Duration, error) {
	if str == "" {
		return 0, ErrEmptyDurationString
	}

	matches := durationRegexp.FindAllStringSubmatch(str, -1)
	if len(matches) == 0 {
		return 0, ErrInvalidDurationString
	}

	var expire int
	if matches[0][2] != "" {
		years, _ := strconv.Atoi(matches[0][2])
		expire += years * 60 * 60 * 24 * 365
	}
	if matches[0][4] != "" {
		months, _ := strconv.Atoi(matches[0][4])
		expire += months * 60 * 60 * 24 * 30
	}
	if matches[0][6] != "" {
		weeks, _ := strconv.Atoi(matches[0][6])
		expire += weeks * 60 * 60 * 24 * 7
	}
	if matches[0][8] != "" {
		days, _ := strconv.Atoi(matches[0][8])
		expire += days * 60 * 60 * 24
	}
	if matches[0][10] != "" {
		hours, _ := strconv.Atoi(matches[0][10])
		expire += hours * 60 * 60
	}
	if matches[0][12] != "" {
		minutes, _ := strconv.Atoi(matches[0][12])
		expire += minutes * 60
	}
	if matches[0][14] != "" {
		seconds, _ := strconv.Atoi(matches[0][14])
		expire += seconds
	}
	dur, err := time.ParseDuration(strconv.Itoa(expire) + "s")
	return dur, err
}

// ParseName takes a name string from a request object and returns the name and tripcode parts
func ParseName(name string) map[string]string {
	parsed := make(map[string]string)
	if !strings.Contains(name, "#") {
		parsed["name"] = name
		parsed["tripcode"] = ""
	} else if strings.Index(name, "#") == 0 {
		parsed["tripcode"] = tripcode.Tripcode(name[1:])
	} else if strings.Index(name, "#") > 0 {
		postNameArr := strings.SplitN(name, "#", 2)
		parsed["name"] = postNameArr[0]
		parsed["tripcode"] = tripcode.Tripcode(postNameArr[1])
	}
	return parsed
}

// RandomString returns a randomly generated string of the given length
func RandomString(length int) string {
	var str string
	for i := 0; i < length; i++ {
		num := rand.Intn(127)
		if num < 32 {
			num += 32
		}
		str += fmt.Sprintf("%c", num)
	}
	return str
}

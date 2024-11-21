package uploads

import (
	"errors"
	"fmt"
	"image"
	"net/http"
	"path"

	"github.com/devedge/imagehash"
	"github.com/disintegration/imaging"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

var (
	ErrVideoThumbFingerprint = errors.New("video thumbnail fingerprinting not enabled")
)

const (
	defaultFingerprintHashLength = 16
)

type FingerprintSource struct {
	FilePath string
	Img      image.Image
	Request  *http.Request
}

func getHashLength() int {
	hashLength := config.GetSiteConfig().FingerprintHashLength
	if hashLength < 1 {
		return defaultFingerprintHashLength
	}
	return hashLength
}

func GetPostImageFingerprint(postID int) (string, error) {
	filename, board, err := gcsql.GetUploadFilenameAndBoard(postID)
	if err != nil {
		return "", err
	}
	subDir := "src"
	if !IsImage(filename) && !IsVideo(filename) {
		return "", ErrUnsupportedFileExt
	} else if IsVideo(filename) {
		if !config.GetSiteConfig().FingerprintVideoThumbnails {
			return "", ErrVideoThumbFingerprint
		}
		filename, _ = GetThumbnailFilenames(filename)
		subDir = "thumb"
	}
	filePath := path.Join(config.GetSystemCriticalConfig().DocumentRoot,
		board, subDir, filename)

	return GetFileFingerprint(filePath)
}

func GetFileFingerprint(filePath string) (string, error) {
	img, err := imaging.Open(filePath)
	if err != nil {
		return "", err
	}

	ba, err := imagehash.Ahash(img, getHashLength())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", ba), nil
}

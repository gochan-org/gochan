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
	filePath := path.Join(config.GetSystemCriticalConfig().DocumentRoot, board, "src", filename)
	return GetFileFingerprint(filePath)
}

func GetFileFingerprint(filePath string) (string, error) {
	if !IsImage(filePath) && !IsVideo(filePath) {
		return "", ErrUnsupportedFileExt
	} else if IsVideo(filePath) {
		if !config.GetSiteConfig().FingerprintVideoThumbnails {
			return "", ErrVideoThumbFingerprint
		}
		filePath, _ = GetThumbnailFilenames(filePath)
		filename := path.Base(filePath)
		fileBoardPath := path.Dir(path.Dir(filePath))
		filePath = path.Join(fileBoardPath, "thumb", filename)
	}
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

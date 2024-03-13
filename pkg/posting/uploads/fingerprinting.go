package uploads

import (
	"database/sql"
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

func checkImageFingerprintBan(img image.Image, _ string) (*gcsql.FileBan, error) {
	hashLength := getHashLength()
	ba, err := imagehash.Ahash(img, hashLength)
	if err != nil {
		return nil, err
	}
	const query = `SELECT id,board_id,staff_id,staff_note,issued_at,checksum,fingerprinter,
	ban_ip,ban_ip_message
	FROM DBPREFIXfile_ban WHERE fingerprinter = 'ahash' AND checksum = ? LIMIT 1`

	var fileBan gcsql.FileBan
	err = gcsql.QueryRowSQL(query, []any{fmt.Sprintf("%x", ba)}, []any{
		&fileBan.ID, &fileBan.BoardID, &fileBan.StaffID, &fileBan.StaffNote,
		&fileBan.IssuedAt, &fileBan.Checksum, &fileBan.Fingerprinter,
		&fileBan.BanIP, &fileBan.BanIPMessage,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &fileBan, err
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

func checkFileFingerprintBan(filePath string, board string) (*gcsql.FileBan, error) {
	img, err := imaging.Open(filePath)
	if err != nil {
		return nil, err
	}
	return checkImageFingerprintBan(img, board)
}

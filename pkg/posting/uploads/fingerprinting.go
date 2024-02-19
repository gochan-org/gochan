package uploads

import (
	"errors"
	"fmt"
	"image"
	"net/http"

	"github.com/devedge/imagehash"
	"github.com/disintegration/imaging"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

var (
	ErrVideoThumbFingerprint = errors.New("video thumbnail fingerprinting not enabled")
)

const (
	defaultFingerprintHashLength = 8
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

func checkImageFingerprintBan(img image.Image, board string) (*gcsql.FileBan, error) {
	hashLength := getHashLength()
	ba, err := imagehash.Ahash(img, hashLength)
	if err != nil {
		return nil, err
	}
	const query = `SELECT id,board_id,staff_id,staff_note,issued_at,checksum,fingerprinter,ban_ip,ban_message
	FROM DBPREFIXfile_ban WHERE fingerprinter = 'ahash' AND checksum = ? LIMIT 1`

	var fileBan gcsql.FileBan
	if err = gcsql.QueryRowSQL(query, []any{fmt.Sprintf("%x", ba)}, []any{
		&fileBan.ID, &fileBan.BoardID, &fileBan.StaffID, &fileBan.StaffNote,
		&fileBan.IssuedAt, &fileBan.Checksum, &fileBan.Fingerprinter,
		&fileBan.BanIP, &fileBan.BanIPMessage,
	}); err != nil {
		return nil, err
	}

	if fileBan.ID == 0 {
		// no matches
		return nil, nil
	}
	return &fileBan, err
}

func FingerprintFile(filePath string) (string, error) {
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

func canFingerprint(filename string) bool {
	siteCfg := config.GetSiteConfig()
	return IsImage(filename) || (IsVideo(filename) && siteCfg.FingerprintVideoThumbnails)
}

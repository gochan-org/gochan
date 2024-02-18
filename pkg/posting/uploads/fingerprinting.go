package uploads

import (
	"fmt"
	"image"
	"net/http"
	"path"

	"github.com/devedge/imagehash"
	"github.com/disintegration/imaging"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

const (
	defaultFingerprintHashLength = 16
)

type FingerprintSource struct {
	FilePath string
	Img      image.Image
	Request  *http.Request
}

func fingerprintImage(img image.Image, board string) (*gcsql.FileBan, error) {
	hashLength := config.GetSiteConfig().FingerprintHashLength
	if hashLength < 1 {
		hashLength = defaultFingerprintHashLength
	}
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

func fingerprintFile(filePath string, board string) (*gcsql.FileBan, error) {
	img, err := imaging.Open(filePath)
	if err != nil {
		return nil, err
	}
	return fingerprintImage(img, board)
}

func canFingerprint(filename string) bool {
	siteCfg := config.GetSiteConfig()
	ext := path.Ext(filename)
	for _, iExt := range ImageExtensions {
		if iExt == ext {
			return true
		}
	}
	if siteCfg.FingerprintVideoThumbnails {
		for _, vExt := range VideoExtensions {
			if vExt == ext {
				return true
			}
		}
	}
	return false
}

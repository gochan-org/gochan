package fingerprinting

import (
	"fmt"
	"image"
	"path"
	"strings"

	"github.com/devedge/imagehash"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

type ahashHandler struct {
	hashVideoThumb bool // if true and the upload is a video, hash the thumb
	hashLength     int
}

func (ah *ahashHandler) Init(options map[string]any) error {
	var ok bool
	for key, val := range options {
		switch strings.ToLower(key) {
		case "hashvideothumb":
			fallthrough
		case "hashvideothumbnail":
			ah.hashVideoThumb, ok = val.(bool)
			if !ok {
				return fmt.Errorf("invalid value type for %q, expected boolean, got %T", key, val)
			}
		case "hashlength":
			ah.hashLength, ok = val.(int)
			if !ok {
				return fmt.Errorf("invalid value type for %q, expected voolean, got %T", key, val)
			}
		}
	}
	if ah.hashLength < 1 {
		ah.hashLength = defaultHashLength
	}
	return nil
}

func (ah *ahashHandler) getImage(source *FingerprintSource) (image.Image, error) {
	if source.Img == nil {
		if source.FilePath == "" {
			file, _, err := source.Request.FormFile("imagefile")
			if err != nil {
				return nil, err
			}
			source.Img, _, err = image.Decode(file)
			return source.Img, err
		}
	}
	return nil, nil
}

func (ah *ahashHandler) CheckFile(source *FingerprintSource, board string) (*gcsql.FileBan, error) {
	img, err := ah.getImage(source)
	if err != nil {
		return nil, err
	}
	ba, err := imagehash.Ahash(img, ah.hashLength)
	if err != nil {
		return nil, err
	}
	const query = `SELECT id,board_id,staff_id,staff_note,issued_at,checksum,fingerprinter,ban_ip,ban_message
	FROM DBPREFIXfile_ban WHERE fingerprinter = 'ahash' AND checksum = ? LIMIT 1`

	var fileBan gcsql.FileBan
	err = gcsql.QueryRowSQL(query, []any{fmt.Sprintf("%x", ba)}, []any{
		&fileBan.ID, &fileBan.BoardID, &fileBan.StaffID, &fileBan.StaffNote,
		&fileBan.IssuedAt, &fileBan.Checksum, &fileBan.Fingerprinter,
		&fileBan.BanIP, &fileBan.BanIPMessage,
	})
	return &fileBan, err
}

func (ah *ahashHandler) IsCompatible(upload *gcsql.Upload) bool {
	switch strings.ToLower(path.Ext(upload.OriginalFilename)) {
	case ".jpg":
		fallthrough
	case ".jpeg":
		fallthrough
	case ".png":
		fallthrough
	case ".gif":
		fallthrough
	case ".tif":
		fallthrough
	case ".tiff":
		fallthrough
	case ".bmp":
		fallthrough
	case ".webp":
		return true
	case ".mp4":
		fallthrough
	case ".webm":
		return ah.hashVideoThumb
	}
	return false
}

func (ahashHandler) Close() error {
	return nil
}

package uploads

import (
	"errors"
	"net/http"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

var (
	ErrFilenameNotAllowed = errors.New("filename not allowed")
	ErrCheckingFileBan    = errors.New("unable to check file ban info")
	ErrFileNotAllowed     = errors.New("uploaded file not allowed")
)

func CheckFilenameBan(upload *gcsql.Upload, post *gcsql.Post, postBoard *gcsql.Board, writer http.ResponseWriter, request *http.Request) error {
	filenameBan, err := gcsql.CheckFilenameBan(upload.OriginalFilename, postBoard.ID)
	if err != nil {
		gcutil.LogError(err).
			Str("IP", post.IP).
			Str("filename", upload.OriginalFilename).
			Str("boardDir", postBoard.Dir).
			Msg("Error getting name banned status")
		return ErrCheckingFileBan
	}
	if filenameBan == nil {
		return nil
	}
	gcutil.LogWarning().
		Str("originalFilename", upload.OriginalFilename).
		Msg("File rejected for having a banned filename")
	return ErrFilenameNotAllowed
}

func CheckFileChecksumBan(upload *gcsql.Upload, post *gcsql.Post, postBoard *gcsql.Board, writer http.ResponseWriter, request *http.Request) error {
	fileBan, err := gcsql.CheckFileChecksumBan(upload.Checksum, postBoard.ID)
	if err != nil {
		gcutil.LogError(err).
			Str("IP", post.IP).
			Str("boardDir", postBoard.Dir).
			Str("checksum", upload.Checksum).
			Msg("Error getting file checksum ban status")
		return ErrCheckingFileBan
	}
	if fileBan == nil {
		return nil
	}
	gcutil.LogWarning().
		Str("originalFilename", upload.OriginalFilename).
		Str("checksum", upload.Checksum).
		Msg("File rejected for having a banned checksum")
	return ErrFileNotAllowed
}

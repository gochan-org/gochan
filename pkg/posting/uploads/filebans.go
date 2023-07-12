package uploads

import (
	"net/http"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
)

func IsFilenameBanned(upload *gcsql.Upload, post *gcsql.Post, postBoard *gcsql.Board, writer http.ResponseWriter, request *http.Request) bool {
	filenameBan, err := gcsql.CheckFilenameBan(upload.OriginalFilename, postBoard.ID)
	if err != nil {
		gcutil.LogError(err).
			Str("IP", post.IP).
			Str("filename", upload.OriginalFilename).
			Str("boardDir", postBoard.Dir).
			Msg("Error getting name banned status")
		server.ServeErrorPage(writer, "Error getting filename ban info")
		return true
	}
	if filenameBan == nil {
		return false
	}
	server.ServeError(writer, "Filename not allowed", serverutil.IsRequestingJSON(request), map[string]interface{}{})
	gcutil.LogWarning().
		Str("originalFilename", upload.OriginalFilename).
		Msg("File rejected for having a banned filename")
	return true
}

func IsChecksumBanned(upload *gcsql.Upload, post *gcsql.Post, postBoard *gcsql.Board, writer http.ResponseWriter, request *http.Request) bool {
	fileBan, err := gcsql.CheckFileChecksumBan(upload.Checksum, postBoard.ID)
	if err != nil {
		gcutil.LogError(err).
			Str("IP", post.IP).
			Str("boardDir", postBoard.Dir).
			Str("checksum", upload.Checksum).
			Msg("Error getting file checksum ban status")
		server.ServeErrorPage(writer, "Error processing file: "+err.Error())
		return true
	}
	if fileBan == nil {
		return false
	}
	server.ServeError(writer, "File not allowed", serverutil.IsRequestingJSON(request), map[string]interface{}{})
	gcutil.LogWarning().
		Str("originalFilename", upload.OriginalFilename).
		Str("checksum", upload.Checksum).
		Msg("File rejected for having a banned checksum")
	return true
}

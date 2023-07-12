package uploads

import (
	"crypto/md5"
	"errors"
	"fmt"
	"html"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
)

// AttachUploadFromRequest reads an incoming HTTP request and processes any incoming files.
// It returns the upload (if there was one) and whether or not any errors were served (meaning
// that it should stop processing the post
func AttachUploadFromRequest(request *http.Request, writer http.ResponseWriter, post *gcsql.Post, postBoard *gcsql.Board) (*gcsql.Upload, bool) {
	errEv := gcutil.LogError(nil).
		Str("IP", post.IP)
	infoEv := gcutil.LogInfo().
		Str("IP", post.IP)
	defer func() {
		infoEv.Discard()
		errEv.Discard()
	}()
	wantsJSON := serverutil.IsRequestingJSON(request)
	file, handler, err := request.FormFile("imagefile")
	if errors.Is(err, http.ErrMissingFile) {
		// no file was submitted with the form
		return nil, false
	}
	if err != nil {
		errEv.Err(err).Caller().Send()
		server.ServeError(writer, err.Error(), wantsJSON, nil)
		return nil, true
	}
	upload := &gcsql.Upload{
		OriginalFilename: html.EscapeString(handler.Filename),
	}
	gcutil.LogStr("originalFilename", upload.OriginalFilename, errEv, infoEv)

	boardConfig := config.GetBoardConfig(postBoard.Dir)
	if !boardConfig.AcceptedExtension(upload.OriginalFilename) {
		errEv.Caller().Msg("Upload filetype not supported")
		server.ServeError(writer, "Upload filetype not supported", wantsJSON, map[string]interface{}{
			"filename": upload.OriginalFilename,
		})
		return nil, true
	}

	if IsFilenameBanned(upload, post, postBoard, writer, request) {
		// If checkFilenameBan returns true, an error occured or the file was
		// rejected for having a banned filename, and the incident was logged either way
		return nil, true
	}

	data, err := io.ReadAll(file)
	if err != nil {
		errEv.Err(err).Caller().Send()
		server.ServeError(writer, "Error while trying to read file: "+err.Error(), wantsJSON, map[string]interface{}{
			"filename": upload.OriginalFilename,
		})
		return nil, true
	}
	defer file.Close()

	// Calculate image checksum
	upload.Checksum = fmt.Sprintf("%x", md5.Sum(data)) // skipcq: GSC-G401
	if IsChecksumBanned(upload, post, postBoard, writer, request) {
		// If checkChecksumBan returns true, an error occured or the file was
		// rejected for having a banned checksum, and the incident was logged either way
		return nil, true
	}

	ext := strings.ToLower(filepath.Ext(upload.OriginalFilename))
	upload.Filename = getNewFilename() + ext

	documentRoot := config.GetSystemCriticalConfig().DocumentRoot
	filePath := path.Join(documentRoot, postBoard.Dir, "src", upload.Filename)
	thumbPath := path.Join(documentRoot, postBoard.Dir, "thumb", upload.ThumbnailPath("thumb"))
	catalogThumbPath := path.Join(documentRoot, postBoard.Dir, "thumb", upload.ThumbnailPath("catalog"))

	errEv.
		Str("originalFilename", upload.OriginalFilename).
		Str("filePath", filePath)
	if post.ThreadID == 0 {
		errEv.Str("catalogThumbPath", catalogThumbPath)
	}

	if err = os.WriteFile(filePath, data, config.GC_FILE_MODE); err != nil {
		errEv.Err(err).Caller().Send()
		server.ServeError(writer, fmt.Sprintf("Couldn't write file %q", upload.OriginalFilename), wantsJSON, map[string]interface{}{
			"filename":         upload.Filename,
			"originalFilename": upload.OriginalFilename,
		})
		return nil, true
	}

	gcutil.LogStr("stripImageMetadata", boardConfig.StripImageMetadata, errEv, infoEv)
	if err = stripImageMetadata(filePath, boardConfig); err != nil {
		errEv.Err(err).Caller().Msg("Unable to strip metadata")
		server.ServeError(writer, "Unable to strip metadata from image", wantsJSON, map[string]interface{}{
			"filename":         upload.Filename,
			"originalFilename": upload.OriginalFilename,
		})
		return nil, true
	}

	// event triggered after the file is successfully written but be
	_, err, recovered := events.TriggerEvent("upload-saved", filePath)
	if recovered {
		writer.WriteHeader(http.StatusInternalServerError)
		server.ServeError(writer, "Unable to save upload (recovered from a panic in event handler)", wantsJSON,
			map[string]interface{}{"event": "upload-saved"})
		return nil, true
	}
	if err != nil {
		server.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
			"filename":         upload.Filename,
			"originalFilename": upload.OriginalFilename,
		})
		return nil, true
	}

	infoEv.Str("referer", request.Referer()).Str("filename", handler.Filename).Send()
	access := gcutil.LogAccess(request).
		Str("filename", handler.Filename).
		Str("referer", request.Referer())

	upload.IsSpoilered = request.FormValue("spoiler") == "on"
	switch ext {
	// images
	case ".gif":
		fallthrough
	case ".jpg":
		fallthrough
	case ".jpeg":
		fallthrough
	case ".png":
		fallthrough
	case ".webp":
		if err = processImage(upload, post, postBoard.Dir, filePath, thumbPath, catalogThumbPath, infoEv, errEv); err != nil {
			server.ServeError(writer, "Error processing image: "+err.Error(), wantsJSON, map[string]interface{}{
				"filename": upload.OriginalFilename,
			})
		}
		access.Str("handler", "image").Send()
	// videos
	case ".mp4":
		fallthrough
	case ".webm":
		if err = processVideo(upload, post, postBoard.Dir, filePath, thumbPath, catalogThumbPath, infoEv, errEv); err != nil {
			server.ServeError(writer, "Error processing video: "+err.Error(), wantsJSON, map[string]interface{}{
				"filename": upload.OriginalFilename,
			})
		}
		access.Str("handler", "video").Send()
	default:
		// other (.pdf, .zip, etc)
		if err = processOther(upload, post, postBoard.Dir, filePath, thumbPath, catalogThumbPath, infoEv, errEv); err != nil {
			server.ServeError(writer, "Error processing file: "+err.Error(), wantsJSON, map[string]interface{}{
				"filename": upload.OriginalFilename,
			})
		}
		access.Str("handler", "other").Send()
	}

	return upload, false
}

func getNewFilename() string {
	now := time.Now().Unix()
	// rand.Seed(now)
	return strconv.Itoa(int(now)) + strconv.Itoa(rand.Intn(98)+1)
}

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
	"github.com/rs/zerolog"
)

var (
	uploadHandlers  map[string]UploadHandler
	ImageExtensions = []string{
		".gif", ".jpg", ".jpeg", ".png", ".webp",
	}
	VideoExtensions = []string{
		".mp4", ".webm",
	}
)

type UploadHandler func(upload *gcsql.Upload, post *gcsql.Post, board string, filePath string, thumbPath string, catalogThumbPath string, infoEv *zerolog.Event, accessEv *zerolog.Event, errEv *zerolog.Event) error

func RegisterUploadHandler(ext string, handler UploadHandler) {
	gcutil.LogInfo().Str("ext", ext).Msg("Registering upload extension handler")
	uploadHandlers[ext] = handler
}

func IsImage(file string) bool {
	ext := path.Ext(file)
	for _, iExt := range ImageExtensions {
		if ext == iExt {
			return true
		}
	}
	return false
}

func IsVideo(file string) bool {
	ext := path.Ext(file)
	for _, vExt := range VideoExtensions {
		if ext == vExt {
			return true
		}
	}
	return false
}

func init() {
	uploadHandlers = make(map[string]UploadHandler)
	for _, ext := range ImageExtensions {
		uploadHandlers[ext] = processImage
	}
	for _, ext := range VideoExtensions {
		uploadHandlers[ext] = processVideo
	}
}

// AttachUploadFromRequest reads an incoming HTTP request and processes any incoming files.
// It returns the upload (if there was one) and whether or not any errors were served (meaning
// that it should stop processing the post
func AttachUploadFromRequest(request *http.Request, writer http.ResponseWriter, post *gcsql.Post, postBoard *gcsql.Board) (*gcsql.Upload, error) {
	errEv := gcutil.LogError(nil).
		Str("IP", post.IP)
	infoEv := gcutil.LogInfo().
		Str("IP", post.IP)
	defer func() {
		infoEv.Discard()
		errEv.Discard()
	}()
	file, handler, err := request.FormFile("imagefile")
	if errors.Is(err, http.ErrMissingFile) {
		// no file was submitted with the form
		return nil, nil
	}
	if err != nil {
		errEv.Err(err).Caller().Send()
		return nil, err
	}
	upload := &gcsql.Upload{
		OriginalFilename: html.EscapeString(handler.Filename),
		FileSize:         int(handler.Size),
	}
	gcutil.LogStr("originalFilename", upload.OriginalFilename, errEv, infoEv)

	boardConfig := config.GetBoardConfig(postBoard.Dir)
	if !boardConfig.AcceptedExtension(upload.OriginalFilename) {
		errEv.Caller().Msg("Upload filetype not supported")
		return nil, ErrUnsupportedFileExt
	}

	if err = CheckFilenameBan(upload, post, postBoard); err != nil {
		// If checkFilenameBan returns true, an error occured or the file was
		// rejected for having a banned filename, and the incident was logged either way
		return nil, err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return nil, errors.New("Error while trying to read file: " + err.Error())
	}
	defer file.Close()

	// Calculate image checksum
	upload.Checksum = fmt.Sprintf("%x", md5.Sum(data)) // skipcq: GSC-G401
	if err = CheckFileChecksumBan(upload, post, postBoard); err != nil {
		// If CheckFileChecksumBan returns a non-nil error, an error occured or the file was
		// rejected for having a banned checksum, and the incident was logged either way
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(upload.OriginalFilename))
	upload.Filename = getNewFilename() + ext

	documentRoot := config.GetSystemCriticalConfig().DocumentRoot
	filePath := path.Join(documentRoot, postBoard.Dir, "src", upload.Filename)
	thumbPath, catalogThumbPath := GetThumbnailFilenames(
		path.Join(documentRoot, postBoard.Dir, "thumb", upload.Filename))

	errEv.
		Str("originalFilename", upload.OriginalFilename).
		Str("filePath", filePath)
	if post.ThreadID == 0 {
		errEv.Str("catalogThumbPath", catalogThumbPath)
	}

	if err = os.WriteFile(filePath, data, config.GC_FILE_MODE); err != nil {
		errEv.Err(err).Caller().Send()
		return nil, fmt.Errorf("couldn't write file %q", upload.OriginalFilename)
	}

	// event triggered after the file is successfully written but be
	_, err, recovered := events.TriggerEvent("upload-saved", filePath)
	if recovered {
		writer.WriteHeader(http.StatusInternalServerError)
		return nil, errors.New("unable to save upload (recovered from a panic in event handler)")
	}
	if err != nil {
		return nil, err
	}

	infoEv.Str("referer", request.Referer()).Str("filename", handler.Filename).Send()
	accessEv := gcutil.LogAccess(request).
		Str("filename", handler.Filename).
		Str("referer", request.Referer())

	upload.IsSpoilered = request.FormValue("spoiler") == "on"

	uploadHandler, ok := uploadHandlers[ext]
	if !ok {
		// ext isn't registered by default (jpg, jpeg, png, gif, webp, mp4, webm) or by a plugin,
		// it's either unsupported or a static thumb as set in configuration
		uploadHandler = processOther
	}

	if err = uploadHandler(upload, post, postBoard.Dir, filePath, thumbPath, catalogThumbPath, infoEv, accessEv, errEv); err != nil {
		return nil, errors.New("error processing upload: " + err.Error())
	}
	accessEv.Send()
	return upload, nil
}

func getNewFilename() string {
	now := time.Now().Unix()
	return strconv.Itoa(int(now)) + strconv.Itoa(rand.Intn(98)+1) // skipcq: GSC-G404
}

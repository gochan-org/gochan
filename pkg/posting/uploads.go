package posting

import (
	"crypto/md5"
	"errors"
	"fmt"
	"html"
	"image"
	"image/gif"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	_ "golang.org/x/image/webp"
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

	if checkFilenameBan(upload, post, postBoard, writer, request) {
		// If checkFilenameBan returns true, an error occured or the file was
		// rejected for having a banned filename, and the incident was logged either way
		return nil, true
	}
	data, err := io.ReadAll(file)
	if err != nil {
		errEv.Err(err).Caller().Send()
		server.ServeErrorPage(writer, "Error while trying to read file: "+err.Error())
		return nil, true
	}
	defer file.Close()

	// Calculate image checksum
	upload.Checksum = fmt.Sprintf("%x", md5.Sum(data)) // skipcq: GSC-G401
	if checkChecksumBan(upload, post, postBoard, writer, request) {
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
	gcutil.LogStr("stripImageMetadata", boardConfig.StripImageMetadata)
	if err = stripImageMetadata(filePath, boardConfig); err != nil {
		errEv.Err(err).Caller().Msg("Unable to strip metadata")
		server.ServeError(writer, "Unable to strip metadata from image", wantsJSON, map[string]interface{}{
			"filename":         upload.Filename,
			"originalFilename": upload.OriginalFilename,
		})
		return nil, true
	}
	_, recovered := events.TriggerEvent("upload-saved", filePath)
	if recovered {
		gcutil.LogWarning().Caller().
			Str("filePath", filePath).Str("triggeredEvent", "upload-saved").
			Msg("Recovered from a panic in event handler")
	}

	if ext == ".webm" || ext == ".mp4" {
		infoEv.Str("post", "withVideo").
			Str("filename", handler.Filename).
			Str("referer", request.Referer()).Send()
		if post.ThreadID == 0 {
			if err := createVideoThumbnail(filePath, thumbPath, boardConfig.ThumbWidth); err != nil {
				errEv.Err(err).Caller().
					Int("thumbWidth", boardConfig.ThumbWidth).
					Msg("Error creating video thumbnail")
				server.ServeErrorPage(writer, "Error creating video thumbnail: "+err.Error())
				return nil, true
			}
		} else {
			if err := createVideoThumbnail(filePath, thumbPath, boardConfig.ThumbWidthReply); err != nil {
				errEv.Err(err).Caller().
					Str("thumbPath", thumbPath).
					Int("thumbWidth", boardConfig.ThumbWidthReply).
					Msg("Error creating video thumbnail for reply")
				server.ServeErrorPage(writer, "Error creating video thumbnail: "+err.Error())
				return nil, true
			}
		}

		if err := createVideoThumbnail(filePath, catalogThumbPath, boardConfig.ThumbWidthCatalog); err != nil {
			errEv.Err(err).Caller().
				Str("thumbPath", thumbPath).
				Int("thumbWidth", boardConfig.ThumbWidthCatalog).
				Msg("Error creating video thumbnail for catalog")
			server.ServeErrorPage(writer, "Error creating video thumbnail: "+err.Error())
			return nil, true
		}

		outputBytes, err := exec.Command("ffprobe", "-v", "quiet", "-show_format", "-show_streams", filePath).CombinedOutput()
		if err != nil {
			gcutil.LogError(err).Msg("Error getting video info")
			server.ServeErrorPage(writer, "Error getting video info: "+err.Error())
			return nil, true
		}
		if outputBytes != nil {
			outputStringArr := strings.Split(string(outputBytes), "\n")
			for _, line := range outputStringArr {
				lineArr := strings.Split(line, "=")
				if len(lineArr) < 2 {
					continue
				}
				value, _ := strconv.Atoi(lineArr[1])
				switch lineArr[0] {
				case "width":
					upload.Width = value
				case "height":
					upload.Height = value
				case "size":
					upload.FileSize = value
				}
			}
			thumbType := "reply"
			if post.ThreadID == 0 {
				thumbType = "op"
			}
			upload.ThumbnailWidth, upload.ThumbnailHeight = getThumbnailSize(
				upload.Width, upload.Height, postBoard.Dir, thumbType)
		}
	} else if cfgThumb, ok := boardConfig.AllowOtherExtensions[ext]; ok {
		stat, err := os.Stat(filePath)
		if err != nil {
			errEv.Err(err).Caller().
				Str("filePath", filePath).Send()
			server.ServeErrorPage(writer, "Couldn't get upload filesize: "+err.Error())
			return nil, true
		}
		upload.FileSize = int(stat.Size())
		if post.ThreadID == 0 {
			// OP
			upload.ThumbnailWidth = boardConfig.ThumbWidth
			upload.ThumbnailHeight = boardConfig.ThumbHeight
		} else {
			// reply
			upload.ThumbnailWidth = boardConfig.ThumbWidthReply
			upload.ThumbnailHeight = boardConfig.ThumbHeightReply
		}
		staticThumbPath := path.Join("static/", cfgThumb)
		originalThumbPath := path.Join(documentRoot, staticThumbPath)
		if _, err = os.Stat(originalThumbPath); err != nil {
			errEv.Err(err).Str("originalThumbPath", originalThumbPath).Send()
			server.ServeError(writer, "missing static thumbnail "+staticThumbPath, wantsJSON, nil)
			return nil, true
		}

		if err = os.Symlink(originalThumbPath, thumbPath); err != nil {
			os.Remove(filePath)
			errEv.Err(err).Caller().
				Str("filePath", filePath).Send()
			server.ServeError(writer, "Failed creating symbolic link to thumbnail path", wantsJSON, nil)
			return nil, true
		}
		if post.ThreadID == 0 {
			if err = os.Symlink(originalThumbPath, catalogThumbPath); err != nil {
				os.Remove(filePath)
				errEv.Err(err).Caller().
					Str("filePath", filePath).Send()
				server.ServeError(writer, "Failed creating symbolic link to thumbnail path", wantsJSON, nil)
				return nil, true
			}
		}
	} else {
		// Attempt to load uploaded file with imaging library
		img, err := imaging.Open(filePath)
		if err != nil {
			os.Remove(filePath)
			errEv.Err(err).Caller().
				Str("filePath", filePath).Send()
			server.ServeErrorPage(writer, "Upload filetype not supported")
			return nil, true
		}
		// Get image filesize
		stat, err := os.Stat(filePath)
		if err != nil {
			errEv.Err(err).Caller().
				Str("filePath", filePath).Send()
			server.ServeErrorPage(writer, "Couldn't get image filesize: "+err.Error())
			return nil, true
		}
		upload.FileSize = int(stat.Size())

		// Get image width and height, as well as thumbnail width and height
		upload.Width = img.Bounds().Max.X
		upload.Height = img.Bounds().Max.Y
		thumbType := "reply"
		if post.ThreadID == 0 {
			thumbType = "op"
		}
		upload.ThumbnailWidth, upload.ThumbnailHeight = getThumbnailSize(
			upload.Width, upload.Height, postBoard.Dir, thumbType)

		gcutil.LogAccess(request).
			Bool("withFile", true).
			Str("filename", handler.Filename).
			Str("referer", request.Referer()).Send()

		if request.FormValue("spoiler") == "on" {
			// If spoiler is enabled, symlink thumbnail to spoiler image
			if _, err := os.Stat(path.Join(documentRoot, "spoiler.png")); err != nil {
				server.ServeErrorPage(writer, "missing spoiler.png")
				return nil, true
			}
			if err = syscall.Symlink(path.Join(documentRoot, "spoiler.png"), thumbPath); err != nil {
				gcutil.LogError(err).
					Str("thumbPath", thumbPath).
					Msg("Error creating symbolic link to thumbnail path")
				server.ServeErrorPage(writer, err.Error())
				return nil, true
			}
		}

		shouldThumb := shouldCreateThumbnail(filePath,
			upload.Width, upload.Height, upload.ThumbnailWidth, upload.ThumbnailHeight)
		if shouldThumb {
			var thumbnail image.Image
			var catalogThumbnail image.Image
			if post.ThreadID == 0 {
				// If this is a new thread, generate thumbnail and catalog thumbnail
				thumbnail = createImageThumbnail(img, postBoard.Dir, "op")
				catalogThumbnail = createImageThumbnail(img, postBoard.Dir, "catalog")
				if err = imaging.Save(catalogThumbnail, catalogThumbPath); err != nil {
					errEv.Err(err).Caller().
						Str("thumbPath", catalogThumbPath).
						Msg("Couldn't generate catalog thumbnail")
					server.ServeErrorPage(writer, "Couldn't generate catalog thumbnail: "+err.Error())
					return nil, true
				}
			} else {
				thumbnail = createImageThumbnail(img, postBoard.Dir, "reply")
			}
			if err = imaging.Save(thumbnail, thumbPath); err != nil {
				errEv.Err(err).Caller().
					Str("thumbPath", thumbPath).
					Msg("Couldn't generate catalog thumbnail")
				server.ServeErrorPage(writer, "Couldn't save thumbnail: "+err.Error())
				return nil, true
			}
		} else {
			// If image fits in thumbnail size, symlink thumbnail to original
			upload.ThumbnailWidth = img.Bounds().Max.X
			upload.ThumbnailHeight = img.Bounds().Max.Y
			if err := syscall.Symlink(filePath, thumbPath); err != nil {
				errEv.Err(err).Caller().
					Str("thumbPath", thumbPath).
					Msg("Couldn't generate catalog thumbnail")
				server.ServeErrorPage(writer, "Couldn't create thumbnail: "+err.Error())
				return nil, true
			}
			if post.ThreadID == 0 {
				// Generate catalog thumbnail
				catalogThumbnail := createImageThumbnail(img, postBoard.Dir, "catalog")
				if err = imaging.Save(catalogThumbnail, catalogThumbPath); err != nil {
					errEv.Err(err).Caller().
						Str("thumbPath", catalogThumbPath).
						Msg("Couldn't generate catalog thumbnail")
					server.ServeErrorPage(writer, "Couldn't generate catalog thumbnail: "+err.Error())
					return nil, true
				}
			}
		}
	}

	return upload, false
}

func stripImageMetadata(filePath string, boardConfig *config.BoardConfig) (err error) {
	var stripFlag string
	switch boardConfig.StripImageMetadata {
	case "exif":
		stripFlag = "-EXIF="
	case "all":
		stripFlag = "-all="
	case "none":
		fallthrough
	case "":
		return nil
	}
	err = exec.Command(boardConfig.ExiftoolPath, "-overwrite_original_in_place", stripFlag, filePath).Run()
	return
}

// func getVideoInfo(path string) (map[string]int, error) {
// 	vidInfo := make(map[string]int)

// 	outputBytes, err := exec.Command("ffprobe", "-v quiet", "-show_format", "-show_streams", path).CombinedOutput()
// 	if err == nil && outputBytes != nil {
// 		outputStringArr := strings.Split(string(outputBytes), "\n")
// 		for _, line := range outputStringArr {
// 			lineArr := strings.Split(line, "=")
// 			if len(lineArr) < 2 {
// 				continue
// 			}

// 			if lineArr[0] == "width" || lineArr[0] == "height" || lineArr[0] == "size" {
// 				value, _ := strconv.Atoi(lineArr[1])
// 				vidInfo[lineArr[0]] = value
// 			}
// 		}
// 	}
// 	return vidInfo, err
// }

func getNewFilename() string {
	now := time.Now().Unix()
	rand.Seed(now)
	return strconv.Itoa(int(now)) + strconv.Itoa(rand.Intn(98)+1)
}

func numImageFrames(imgPath string) (int, error) {
	if path.Ext(imgPath) != ".gif" {
		return 1, nil
	}
	fi, err := os.Open(imgPath)
	if err != nil {
		return 0, err
	}
	defer fi.Close()
	g, err := gif.DecodeAll(fi)
	if err != nil {
		return 0, err
	}
	return len(g.Image), nil
}

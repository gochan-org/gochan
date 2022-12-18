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
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/serverutil"
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
	file, handler, err := request.FormFile("imagefile")
	if err == http.ErrMissingFile {
		// no file was submitted with the form
		return nil, false
	}
	wantsJSON := serverutil.IsRequestingJSON(request)
	if err != nil {
		errEv.Err(err).Caller().Send()
		serverutil.ServeError(writer, err.Error(), wantsJSON, nil)
		return nil, true
	}
	upload := &gcsql.Upload{
		OriginalFilename: html.EscapeString(handler.Filename),
	}
	if checkFilenameBan(upload, post, postBoard, writer, request) {
		// If checkFilenameBan returns true, an error occured or the file was
		// rejected for having a banned filename, and the incident was logged either way
		return nil, true
	}
	data, err := io.ReadAll(file)
	if err != nil {
		errEv.Err(err).Caller().Send()
		serverutil.ServeErrorPage(writer, "Error while trying to read file: "+err.Error())
		return nil, true
	}
	defer file.Close()

	// Calculate image checksum
	upload.Checksum = fmt.Sprintf("%x", md5.Sum(data))
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

	if err = os.WriteFile(filePath, data, 0644); err != nil {
		errEv.Err(err).Caller().
			Str("filename", upload.Filename).
			Str("originalFilename", upload.OriginalFilename).
			Send()
		serverutil.ServeError(writer, fmt.Sprintf("Couldn't write file %q", upload.OriginalFilename), wantsJSON, map[string]interface{}{
			"filename":         upload.Filename,
			"originalFilename": upload.OriginalFilename,
		})
		return nil, true
	}
	errEv.
		Str("filename", handler.Filename).
		Str("filePath", filePath).
		Str("thumbPath", thumbPath)

	boardConfig := config.GetBoardConfig(postBoard.Dir)
	if ext == "webm" || ext == "mp4" {
		infoEv.Str("post", "withVideo").
			Str("filename", handler.Filename).
			Str("referer", request.Referer()).Send()
		if post.IsTopPost {
			if err := createVideoThumbnail(filePath, thumbPath, boardConfig.ThumbWidth); err != nil {
				errEv.Err(err).Caller().
					Str("filePath", filePath).
					Str("thumbPath", thumbPath).
					Int("thumbWidth", boardConfig.ThumbWidth).
					Msg("Error creating video thumbnail")
				serverutil.ServeErrorPage(writer, "Error creating video thumbnail: "+err.Error())
				return nil, true
			}
		} else {
			if err := createVideoThumbnail(filePath, thumbPath, boardConfig.ThumbWidthReply); err != nil {
				errEv.Err(err).Caller().
					Str("filePath", filePath).
					Str("thumbPath", thumbPath).
					Int("thumbWidth", boardConfig.ThumbWidthReply).
					Msg("Error creating video thumbnail for reply")
				serverutil.ServeErrorPage(writer, "Error creating video thumbnail: "+err.Error())
				return nil, true
			}
		}

		if err := createVideoThumbnail(filePath, catalogThumbPath, boardConfig.ThumbWidthCatalog); err != nil {
			errEv.Err(err).Caller().
				Str("filePath", filePath).
				Str("thumbPath", thumbPath).
				Int("thumbWidth", boardConfig.ThumbWidthCatalog).
				Msg("Error creating video thumbnail for catalog")
			serverutil.ServeErrorPage(writer, "Error creating video thumbnail: "+err.Error())
			return nil, true
		}

		outputBytes, err := exec.Command("ffprobe", "-v", "quiet", "-show_format", "-show_streams", filePath).CombinedOutput()
		if err != nil {
			gcutil.LogError(err).Msg("Error getting video info")
			serverutil.ServeErrorPage(writer, "Error getting video info: "+err.Error())
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
			if post.IsTopPost {
				thumbType = "op"
			}
			upload.ThumbnailWidth, upload.ThumbnailHeight = getThumbnailSize(
				upload.Width, upload.Height, postBoard.Dir, thumbType)
		}
	} else {
		// Attempt to load uploaded file with imaging library
		img, err := imaging.Open(filePath)
		if err != nil {
			os.Remove(filePath)
			errEv.Err(err).Caller().
				Str("filePath", filePath).Send()
			serverutil.ServeErrorPage(writer, "Upload filetype not supported")
			return nil, true
		}
		// Get image filesize
		stat, err := os.Stat(filePath)
		if err != nil {
			errEv.Err(err).Caller().
				Str("filePath", filePath).Send()
			serverutil.ServeErrorPage(writer, "Couldn't get image filesize: "+err.Error())
			return nil, true
		}
		upload.FileSize = int(stat.Size())

		// Get image width and height, as well as thumbnail width and height
		upload.Width = img.Bounds().Max.X
		upload.Height = img.Bounds().Max.Y
		thumbType := "reply"
		if post.IsTopPost {
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
				serverutil.ServeErrorPage(writer, "missing spoiler.png")
				return nil, true
			}
			if err = syscall.Symlink(path.Join(documentRoot, "spoiler.png"), thumbPath); err != nil {
				gcutil.LogError(err).
					Str("thumbPath", thumbPath).
					Msg("Error creating symbolic link to thumbnail path")
				serverutil.ServeErrorPage(writer, err.Error())
				return nil, true
			}
		}

		shouldThumb := shouldCreateThumbnail(filePath,
			upload.Width, upload.Height, upload.ThumbnailWidth, upload.ThumbnailHeight)
		if shouldThumb {
			var thumbnail image.Image
			var catalogThumbnail image.Image
			if post.IsTopPost {
				// If this is a new thread, generate thumbnail and catalog thumbnail
				thumbnail = createImageThumbnail(img, postBoard.Dir, "op")
				catalogThumbnail = createImageThumbnail(img, postBoard.Dir, "catalog")
				if err = imaging.Save(catalogThumbnail, catalogThumbPath); err != nil {
					errEv.Err(err).Caller().
						Str("thumbPath", catalogThumbPath).
						Msg("Couldn't generate catalog thumbnail")
					serverutil.ServeErrorPage(writer, "Couldn't generate catalog thumbnail: "+err.Error())
					return nil, true
				}
			} else {
				thumbnail = createImageThumbnail(img, postBoard.Dir, "reply")
			}
			if err = imaging.Save(thumbnail, thumbPath); err != nil {
				errEv.Err(err).Caller().
					Str("thumbPath", thumbPath).
					Msg("Couldn't generate catalog thumbnail")
				serverutil.ServeErrorPage(writer, "Couldn't save thumbnail: "+err.Error())
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
				serverutil.ServeErrorPage(writer, "Couldn't create thumbnail: "+err.Error())
				return nil, true
			}
			if post.IsTopPost {
				// Generate catalog thumbnail
				catalogThumbnail := createImageThumbnail(img, postBoard.Dir, "catalog")
				if err = imaging.Save(catalogThumbnail, catalogThumbPath); err != nil {
					errEv.Err(err).Caller().
						Str("thumbPath", catalogThumbPath).
						Msg("Couldn't generate catalog thumbnail")
					serverutil.ServeErrorPage(writer, "Couldn't generate catalog thumbnail: "+err.Error())
					return nil, true
				}
			}
		}
	}

	return upload, false
}

func getBoardThumbnailSize(boardDir string, thumbType string) (int, int) {
	boardCfg := config.GetBoardConfig(boardDir)
	switch thumbType {
	case "op":
		return boardCfg.ThumbWidth, boardCfg.ThumbHeight
	case "reply":
		return boardCfg.ThumbWidthReply, boardCfg.ThumbHeightReply
	case "catalog":
		return boardCfg.ThumbWidth, boardCfg.ThumbHeight
	}
	// todo: use reflect package to print location to error log, because this shouldn't happen
	return -1, -1
}

func createImageThumbnail(imageObj image.Image, boardDir string, thumbType string) image.Image {
	thumbWidth, thumbHeight := getBoardThumbnailSize(boardDir, thumbType)

	oldRect := imageObj.Bounds()
	if thumbWidth >= oldRect.Max.X && thumbHeight >= oldRect.Max.Y {
		return imageObj
	}

	thumbW, thumbH := getThumbnailSize(oldRect.Max.X, oldRect.Max.Y, boardDir, thumbType)
	imageObj = imaging.Resize(imageObj, thumbW, thumbH, imaging.CatmullRom) // resize to 600x400 px using CatmullRom cubic filter
	return imageObj
}

func shouldCreateThumbnail(imgPath string, imgWidth int, imgHeight int, thumbWidth int, thumbHeight int) bool {
	ext := strings.ToLower(path.Ext(imgPath))
	if ext == ".gif" {
		numFrames, err := numImageFrames(imgPath)
		if err != nil {
			gcutil.LogError(err).
				Str("imgPath", imgPath).Send()
			return true
		}
		if numFrames > 1 {
			return true
		}
	}

	return imgWidth > thumbWidth || imgHeight > thumbHeight
}

func createVideoThumbnail(video, thumb string, size int) error {
	sizeStr := strconv.Itoa(size)
	outputBytes, err := exec.Command("ffmpeg", "-y", "-itsoffset", "-1", "-i", video, "-vframes", "1", "-filter:v", "scale='min("+sizeStr+"\\, "+sizeStr+"):-1'", thumb).CombinedOutput()
	if err != nil {
		outputStringArr := strings.Split(string(outputBytes), "\n")
		if len(outputStringArr) > 1 {
			outputString := outputStringArr[len(outputStringArr)-2]
			err = errors.New(outputString)
		}
	}
	return err
}

func getVideoInfo(path string) (map[string]int, error) {
	vidInfo := make(map[string]int)

	outputBytes, err := exec.Command("ffprobe", "-v quiet", "-show_format", "-show_streams", path).CombinedOutput()
	if err == nil && outputBytes != nil {
		outputStringArr := strings.Split(string(outputBytes), "\n")
		for _, line := range outputStringArr {
			lineArr := strings.Split(line, "=")
			if len(lineArr) < 2 {
				continue
			}

			if lineArr[0] == "width" || lineArr[0] == "height" || lineArr[0] == "size" {
				value, _ := strconv.Atoi(lineArr[1])
				vidInfo[lineArr[0]] = value
			}
		}
	}
	return vidInfo, err
}

func getNewFilename() string {
	now := time.Now().Unix()
	rand.Seed(now)
	return strconv.Itoa(int(now)) + strconv.Itoa(rand.Intn(98)+1)
}

// find out what out thumbnail's width and height should be, partially ripped from Kusaba X
func getThumbnailSize(uploadWidth, uploadHeight int, boardDir string, thumbType string) (newWidth, newHeight int) {
	thumbWidth, thumbHeight := getBoardThumbnailSize(boardDir, thumbType)
	if uploadWidth < thumbWidth && uploadHeight < thumbHeight {
		newWidth = uploadWidth
		newHeight = uploadHeight
	} else if uploadWidth == uploadHeight {
		newWidth = thumbWidth
		newHeight = thumbHeight
	} else {
		var percent float32
		if uploadWidth > uploadHeight {
			percent = float32(thumbWidth) / float32(uploadWidth)
		} else {
			percent = float32(thumbHeight) / float32(uploadHeight)
		}
		newWidth = int(float32(uploadWidth) * percent)
		newHeight = int(float32(uploadHeight) * percent)
	}
	return
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

package posting

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"fmt"
	"html"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/serverutil"
)

const (
	yearInSeconds = 31536000
)

// MakePost is called when a user accesses /post. Parse form data, then insert and build
func MakePost(writer http.ResponseWriter, request *http.Request) {
	var maxMessageLength int
	var post gcsql.Post
	var formName string
	var nameCookie string
	var formEmail string

	systemCritical := config.GetSystemCriticalConfig()
	boardConfig := config.GetBoardConfig("")

	if request.Method == "GET" {
		http.Redirect(writer, request, systemCritical.WebRoot, http.StatusFound)
		return
	}
	post.IP = gcutil.GetRealIP(request)
	post.ParentID, _ = strconv.Atoi(request.FormValue("threadid"))
	post.BoardID, _ = strconv.Atoi(request.FormValue("boardid"))
	var postBoard gcsql.Board
	postBoard, err := gcsql.GetBoardFromID(post.BoardID)
	if err != nil {
		gcutil.LogError(err).
			Int("boardid", post.BoardID).
			Str("IP", post.IP).
			Msg("Error getting board info")
		serverutil.ServeErrorPage(writer, "Error getting board info: "+err.Error())
		return
	}

	var emailCommand string
	formName = request.FormValue("postname")
	parsedName := gcutil.ParseName(formName)
	post.Name = parsedName["name"]
	post.Tripcode = parsedName["tripcode"]

	formEmail = request.FormValue("postemail")

	http.SetCookie(writer, &http.Cookie{
		Name:   "email",
		Value:  formEmail,
		MaxAge: yearInSeconds,
	})

	if !strings.Contains(formEmail, "noko") && !strings.Contains(formEmail, "sage") {
		post.Email = formEmail
	} else if strings.Index(formEmail, "#") > 1 {
		formEmailArr := strings.SplitN(formEmail, "#", 2)
		post.Email = formEmailArr[0]
		emailCommand = formEmailArr[1]
	} else if formEmail == "noko" || formEmail == "sage" {
		emailCommand = formEmail
		post.Email = ""
	}

	post.Subject = request.FormValue("postsubject")
	post.MessageText = strings.Trim(request.FormValue("postmsg"), "\r\n")

	if maxMessageLength, err = gcsql.GetMaxMessageLength(post.BoardID); err != nil {
		gcutil.LogError(err).
			Int("boardid", post.BoardID).
			Str("IP", post.IP).
			Msg("Error getting board info")
		serverutil.ServeErrorPage(writer, "Error getting board info: "+err.Error())
		return
	}

	if len(post.MessageText) > maxMessageLength {
		serverutil.ServeErrorPage(writer, "Post body is too long")
		return
	}

	if post.MessageText, err = ApplyWordFilters(post.MessageText, postBoard.Dir); err != nil {
		gcutil.LogError(err).
			Str("IP", post.IP).
			Str("boardDir", postBoard.Dir).
			Msg("Error applying wordfilters")
		serverutil.ServeErrorPage(writer, "Error formatting post: "+err.Error())
		return
	}

	post.MessageHTML = FormatMessage(post.MessageText, postBoard.Dir)
	password := request.FormValue("postpassword")
	if password == "" {
		password = gcutil.RandomString(8)
	}
	post.Password = gcutil.Md5Sum(password)

	// Reverse escapes
	nameCookie = strings.Replace(formName, "&amp;", "&", -1)
	nameCookie = strings.Replace(nameCookie, "\\&#39;", "'", -1)
	nameCookie = strings.Replace(url.QueryEscape(nameCookie), "+", "%20", -1)

	// add name and email cookies that will expire in a year (31536000 seconds)
	http.SetCookie(writer, &http.Cookie{
		Name:   "name",
		Value:  nameCookie,
		MaxAge: yearInSeconds,
	})
	http.SetCookie(writer, &http.Cookie{
		Name:   "password",
		Value:  password,
		MaxAge: yearInSeconds,
	})

	post.Timestamp = time.Now()
	// post.PosterAuthority = getStaffRank(request)
	post.Bumped = time.Now()
	post.Stickied = request.FormValue("modstickied") == "on"
	post.Locked = request.FormValue("modlocked") == "on"

	//post has no referrer, or has a referrer from a different domain, probably a spambot
	if !serverutil.ValidReferer(request) {
		gcutil.LogWarning().
			Str("spam", "badReferer").
			Str("IP", post.IP).
			Msg("Rejected post from possible spambot")
		return
	}

	akismetResult := serverutil.CheckPostForSpam(
		post.IP, request.Header.Get("User-Agent"), request.Referer(),
		post.Name, post.Email, post.MessageText,
	)
	logEvent := gcutil.LogInfo().
		Str("User-Agent", request.Header.Get("User-Agent")).
		Str("IP", post.IP)
	switch akismetResult {
	case "discard":
		logEvent.Str("akismet", "discard").Send()
		serverutil.ServeErrorPage(writer, "Your post looks like spam.")
		return
	case "spam":
		logEvent.Str("akismet", "spam").Send()
		serverutil.ServeErrorPage(writer, "Your post looks like spam.")
		return
	default:
		logEvent.Discard()
	}

	postDelay, _ := gcsql.SinceLastPost(post.IP)
	if postDelay > -1 {
		if post.ParentID == 0 && postDelay < boardConfig.NewThreadDelay {
			serverutil.ServeErrorPage(writer, "Please wait before making a new thread.")
			return
		} else if post.ParentID > 0 && postDelay < boardConfig.ReplyDelay {
			serverutil.ServeErrorPage(writer, "Please wait before making a reply.")
			return
		}
	}

	banStatus, err := getBannedStatus(request)
	if err != nil && err != sql.ErrNoRows {
		gcutil.LogError(err).
			Str("IP", post.IP).
			Fields(gcutil.ParseName(formName)).
			Msg("Error getting banned status")
		serverutil.ServeErrorPage(writer, "Error getting banned status: "+err.Error())
		return
	}

	boards, _ := gcsql.GetAllBoards()

	if banStatus != nil && banStatus.IsBanned(postBoard.Dir) {
		var banpageBuffer bytes.Buffer

		if err = serverutil.MinifyTemplate(gctemplates.Banpage, map[string]interface{}{
			"systemCritical": config.GetSystemCriticalConfig(),
			"siteConfig":     config.GetSiteConfig(),
			"boardConfig":    config.GetBoardConfig(""),
			"ban":            banStatus,
			"banBoards":      boards[post.BoardID-1].Dir,
		}, writer, "text/html"); err != nil {
			gcutil.LogError(err).
				Str("building", "minifier").Send()
			serverutil.ServeErrorPage(writer, "Error minifying page: "+err.Error())
			return
		}
		writer.Write(banpageBuffer.Bytes())
		return
	}

	post.Sanitize()

	if boardConfig.UseCaptcha {
		captchaID := request.FormValue("captchaid")
		captchaAnswer := request.FormValue("captchaanswer")
		if captchaID == "" && captchaAnswer == "" {
			// browser isn't using JS, save post data to tempPosts and show captcha
			request.Form.Add("temppostindex", strconv.Itoa(len(gcsql.TempPosts)))
			request.Form.Add("emailcmd", emailCommand)
			gcsql.TempPosts = append(gcsql.TempPosts, post)

			ServeCaptcha(writer, request)
			return
		}
	}

	file, handler, err := request.FormFile("imagefile")
	var filePath, thumbPath, catalogThumbPath string
	if err != nil || handler.Size == 0 {
		// no file was uploaded
		post.Filename = ""
		if strings.TrimSpace(post.MessageText) == "" {
			serverutil.ServeErrorPage(writer, "Post must contain a message if no image is uploaded.")
			return
		}
		gcutil.LogAccess(request).
			Str("post", "referred").
			Str("referredFrom", request.Referer()).
			Send()
	} else {
		data, err := io.ReadAll(file)
		if err != nil {
			gcutil.LogError(err).
				Str("upload", "read").Send()
			serverutil.ServeErrorPage(writer, "Error while trying to read file: "+err.Error())
			return
		}
		defer file.Close()
		post.FilenameOriginal = html.EscapeString(handler.Filename)
		ext := gcutil.GetFileExtension(post.FilenameOriginal)
		thumbExt := strings.ToLower(ext)
		if thumbExt == "gif" || thumbExt == "webm" || thumbExt == "mp4" {
			thumbExt = "jpg"
		}

		post.Filename = getNewFilename() + "." + ext
		boardExists := gcsql.DoesBoardExistByID(
			gcutil.HackyStringToInt(request.FormValue("boardid")))
		if !boardExists {
			serverutil.ServeErrorPage(writer, "No boards have been created yet")
			return
		}
		var _board = gcsql.Board{}
		err = _board.PopulateData(gcutil.HackyStringToInt(request.FormValue("boardid")))
		if err != nil {
			gcutil.LogError(err).
				Str("IP", post.IP).
				Str("posting", "updateBoard").Send()
			serverutil.ServeErrorPage(writer, "Server error: "+err.Error())
			return
		}
		boardDir := _board.Dir
		filePath = path.Join(systemCritical.DocumentRoot, boardDir, "src", post.Filename)
		thumbPath = path.Join(systemCritical.DocumentRoot, boardDir, "thumb", strings.Replace(post.Filename, "."+ext, "t."+thumbExt, -1))
		catalogThumbPath = path.Join(systemCritical.DocumentRoot, boardDir, "thumb", strings.Replace(post.Filename, "."+ext, "c."+thumbExt, -1))

		if err = os.WriteFile(filePath, data, 0644); err != nil {
			gcutil.LogError(err).
				Str("posting", "upload").
				Str("IP", post.IP).
				Str("filename", post.Filename).Send()
			serverutil.ServeErrorPage(writer, fmt.Sprintf("Couldn't write file %q", post.FilenameOriginal))
			return
		}

		// Calculate image checksum
		post.FileChecksum = fmt.Sprintf("%x", md5.Sum(data))

		if ext == "webm" || ext == "mp4" {
			gcutil.LogInfo().
				Str("post", "withVideo").
				Str("IP", post.IP).
				Str("filename", handler.Filename).
				Str("referer", request.Referer()).Send()
			if post.ParentID == 0 {
				if err := createVideoThumbnail(filePath, thumbPath, boardConfig.ThumbWidth); err != nil {
					gcutil.LogError(err).
						Str("filePath", filePath).
						Str("thumbPath", thumbPath).
						Int("thumbWidth", boardConfig.ThumbWidth).
						Msg("Error creating video thumbnail")
					serverutil.ServeErrorPage(writer, "Error creating video thumbnail: "+err.Error())
					return
				}
			} else {
				if err := createVideoThumbnail(filePath, thumbPath, boardConfig.ThumbWidthReply); err != nil {
					gcutil.LogError(err).
						Str("filePath", filePath).
						Str("thumbPath", thumbPath).
						Int("thumbWidth", boardConfig.ThumbWidthReply).
						Msg("Error creating video thumbnail for reply")
					serverutil.ServeErrorPage(writer, "Error creating video thumbnail: "+err.Error())
					return
				}
			}

			if err := createVideoThumbnail(filePath, catalogThumbPath, boardConfig.ThumbWidthCatalog); err != nil {
				gcutil.LogError(err).
					Str("filePath", filePath).
					Str("thumbPath", thumbPath).
					Int("thumbWidth", boardConfig.ThumbWidthCatalog).
					Msg("Error creating video thumbnail for catalog")
				serverutil.ServeErrorPage(writer, "Error creating video thumbnail: "+err.Error())
				return
			}

			outputBytes, err := exec.Command("ffprobe", "-v", "quiet", "-show_format", "-show_streams", filePath).CombinedOutput()
			if err != nil {
				gcutil.LogError(err).Msg("Error getting video info")

				serverutil.ServeErrorPage(writer, "Error getting video info: "+err.Error())
				return
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
						post.ImageW = value
					case "height":
						post.ImageH = value
					case "size":
						post.Filesize = value
					}
				}
				thumbType := "reply"
				if post.ParentID == 0 {
					thumbType = "op"
				}
				post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, boardDir, thumbType)
			}
		} else {
			// Attempt to load uploaded file with imaging library
			img, err := imaging.Open(filePath)
			if err != nil {
				os.Remove(filePath)
				gcutil.LogError(err).
					Str("filePath", filePath).Send()
				serverutil.ServeErrorPage(writer, "Upload filetype not supported")
				return
			}
			// Get image filesize
			stat, err := os.Stat(filePath)
			if err != nil {
				gcutil.LogError(err).
					Str("filePath", filePath).Send()
				serverutil.ServeErrorPage(writer, "Couldn't get image filesize: "+err.Error())
				return
			}
			post.Filesize = int(stat.Size())

			// Get image width and height, as well as thumbnail width and height
			post.ImageW = img.Bounds().Max.X
			post.ImageH = img.Bounds().Max.Y
			thumbType := "reply"
			if post.ParentID == 0 {
				thumbType = "op"
			}
			post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, boardDir, thumbType)

			gcutil.LogAccess(request).
				Bool("withFile", true).
				Str("filename", handler.Filename).
				Str("referer", request.Referer()).Send()

			if request.FormValue("spoiler") == "on" {
				// If spoiler is enabled, symlink thumbnail to spoiler image
				if _, err := os.Stat(path.Join(systemCritical.DocumentRoot, "spoiler.png")); err != nil {
					serverutil.ServeErrorPage(writer, "missing spoiler.png")
					return
				}
				if err = syscall.Symlink(path.Join(systemCritical.DocumentRoot, "spoiler.png"), thumbPath); err != nil {
					gcutil.LogError(err).
						Str("thumbPath", thumbPath).
						Msg("Error creating symbolic link to thumbnail path")
					serverutil.ServeErrorPage(writer, err.Error())
					return
				}
			}

			shouldThumb := shouldCreateThumbnail(filePath, post.ImageW, post.ImageH, post.ThumbW, post.ThumbH)
			if shouldThumb {
				var thumbnail image.Image
				var catalogThumbnail image.Image
				if post.ParentID == 0 {
					// If this is a new thread, generate thumbnail and catalog thumbnail
					thumbnail = createImageThumbnail(img, boardDir, "op")
					catalogThumbnail = createImageThumbnail(img, boardDir, "catalog")
					if err = imaging.Save(catalogThumbnail, catalogThumbPath); err != nil {
						gcutil.LogError(err).
							Str("thumbPath", catalogThumbPath).
							Str("IP", post.IP).
							Msg("Couldn't generate catalog thumbnail")
						serverutil.ServeErrorPage(writer, "Couldn't generate catalog thumbnail: "+err.Error())
						return
					}
				} else {
					thumbnail = createImageThumbnail(img, boardDir, "reply")
				}
				if err = imaging.Save(thumbnail, thumbPath); err != nil {
					gcutil.LogError(err).
						Str("thumbPath", thumbPath).
						Str("IP", post.IP).
						Msg("Couldn't generate catalog thumbnail")
					serverutil.ServeErrorPage(writer, "Couldn't save thumbnail: "+err.Error())
					return
				}
			} else {
				// If image fits in thumbnail size, symlink thumbnail to original
				post.ThumbW = img.Bounds().Max.X
				post.ThumbH = img.Bounds().Max.Y
				if err := syscall.Symlink(filePath, thumbPath); err != nil {
					gcutil.LogError(err).
						Str("thumbPath", thumbPath).
						Str("IP", post.IP).
						Msg("Couldn't generate catalog thumbnail")
					serverutil.ServeErrorPage(writer, "Couldn't create thumbnail: "+err.Error())
					return
				}
				if post.ParentID == 0 {
					// Generate catalog thumbnail
					catalogThumbnail := createImageThumbnail(img, boardDir, "catalog")
					if err = imaging.Save(catalogThumbnail, catalogThumbPath); err != nil {
						gcutil.LogError(err).
							Str("thumbPath", catalogThumbPath).
							Str("IP", post.IP).
							Msg("Couldn't generate catalog thumbnail")
						serverutil.ServeErrorPage(writer, "Couldn't generate catalog thumbnail: "+err.Error())
						return
					}
				}
			}
		}
	}

	if err = gcsql.InsertPost(&post, emailCommand != "sage"); err != nil {
		gcutil.LogError(err).
			Str("sql", "postInsertion").Send()
		if post.Filename != "" {
			os.Remove(filePath)
			os.Remove(thumbPath)
			os.Remove(catalogThumbPath)
		}
		return
	}

	// rebuild the board page
	building.BuildBoards(false, post.BoardID)
	building.BuildFrontPage()

	if emailCommand == "noko" {
		if post.ParentID < 1 {
			http.Redirect(writer, request, systemCritical.WebRoot+postBoard.Dir+"/res/"+strconv.Itoa(post.ID)+".html", http.StatusFound)
		} else {
			http.Redirect(writer, request, systemCritical.WebRoot+postBoard.Dir+"/res/"+strconv.Itoa(post.ParentID)+".html#"+strconv.Itoa(post.ID), http.StatusFound)
		}
	} else {
		http.Redirect(writer, request, systemCritical.WebRoot+postBoard.Dir+"/", http.StatusFound)
	}
}

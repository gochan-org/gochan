package posting

import (
	"crypto/md5"
	"errors"
	"fmt"
	"html"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/serverutil"
)

const (
	yearInSeconds = 31536000
)

var (
	ErrorPostTooLong = errors.New("post is too long")
)

func rejectPost(reasonShort string, reasonLong string, data map[string]interface{}, writer http.ResponseWriter, request *http.Request) {
	gcutil.LogError(errors.New(reasonLong)).
		Str("rejectedPost", reasonShort).
		Str("IP", gcutil.GetRealIP(request)).
		Fields(data).Send()
	data["rejected"] = reasonLong
	serverutil.ServeError(writer, reasonLong, serverutil.IsRequestingJSON(request), data)
}

// MakePost is called when a user accesses /post. Parse form data, then insert and build
func MakePost(writer http.ResponseWriter, request *http.Request) {
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
	wantsJSON := serverutil.IsRequestingJSON(request)
	post.IP = gcutil.GetRealIP(request)
	var err error
	threadidStr := request.FormValue("threadid")
	if threadidStr != "" {
		// post is a reply
		if post.ThreadID, err = strconv.Atoi(threadidStr); err != nil {
			rejectPost("invalidFormData", "Invalid form data (invalid threadid)", map[string]interface{}{
				"threadidStr": threadidStr,
			}, writer, request)
			return
		}
	}

	boardidStr := request.FormValue("boardid")
	boardID, err := strconv.Atoi(boardidStr)
	if err != nil {
		rejectPost("invalidForm", "Invalid form data (invalid boardid)", map[string]interface{}{
			"boardidStr": boardidStr,
		}, writer, request)
		return
	}
	postBoard, err := gcsql.GetBoardFromID(boardID)
	if err != nil {
		rejectPost("boardInfoError", "Error getting board info: "+err.Error(), map[string]interface{}{
			"boardid": boardID,
		}, writer, request)
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
	post.MessageRaw = strings.TrimSpace(request.FormValue("postmsg"))
	if len(post.MessageRaw) > postBoard.MaxMessageLength {
		rejectPost("messageLength", "Message is too long", map[string]interface{}{
			"messageLength": len(post.MessageRaw),
			"boardid":       boardID,
		}, writer, request)
		return
	}

	if post.MessageRaw, err = ApplyWordFilters(post.MessageRaw, postBoard.Dir); err != nil {
		rejectPost("wordfilterError", "Error formatting post: "+err.Error(), map[string]interface{}{
			"boardDir": postBoard.Dir,
		}, writer, request)
		return
	}

	post.Message = FormatMessage(post.MessageRaw, postBoard.Dir)
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

	post.CreatedOn = time.Now()
	// post.PosterAuthority = getStaffRank(request)
	// bumpedTimestamp := time.Now()
	// isSticky := request.FormValue("modstickied") == "on"
	// isLocked := request.FormValue("modlocked") == "on"

	//post has no referrer, or has a referrer from a different domain, probably a spambot
	if !serverutil.ValidReferer(request) {
		gcutil.LogWarning().
			Str("spam", "badReferer").
			Str("IP", post.IP).
			Msg("Rejected post from possible spambot")
		serverutil.ServeError(writer, "Your post looks like spam", wantsJSON, nil)
		return
	}

	akismetResult := serverutil.CheckPostForSpam(
		post.IP, request.Header.Get("User-Agent"), request.Referer(),
		post.Name, post.Email, post.MessageRaw,
	)
	logEvent := gcutil.LogInfo().
		Str("User-Agent", request.Header.Get("User-Agent")).
		Str("IP", post.IP)
	switch akismetResult {
	case "discard":
		logEvent.Str("akismet", "discard").Send()
		serverutil.ServeError(writer, "Your post looks like spam.", wantsJSON, nil)
		return
	case "spam":
		logEvent.Str("akismet", "spam").Send()
		serverutil.ServeError(writer, "Your post looks like spam.", wantsJSON, nil)
		return
	default:
		logEvent.Discard()
	}

	var delay int
	var tooSoon bool
	if threadidStr == "" {
		// creating a new thread
		delay, err = gcsql.SinceLastThread(post.IP)
		tooSoon = delay < boardConfig.NewThreadDelay
	} else {
		delay, err = gcsql.SinceLastPost(post.IP)
		tooSoon = delay < boardConfig.ReplyDelay
	}
	if err != nil {
		rejectPost("cooldownError", "Error checking post cooldown: "+err.Error(), map[string]interface{}{
			"boardDir": postBoard.Dir,
		}, writer, request)
		return
	}
	if tooSoon {
		rejectPost("cooldownError", "Please wait before making a new post", map[string]interface{}{}, writer, request)
		return
	}

	if checkIpBan(&post, postBoard, writer, request) {
		return
	}
	if checkUsernameBan(formName, &post, postBoard, writer, request) {
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
		if strings.TrimSpace(post.MessageRaw) == "" {
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
		var upload gcsql.Upload
		upload.OriginalFilename = html.EscapeString(handler.Filename)

		ext := strings.ToLower(filepath.Ext(upload.OriginalFilename))
		upload.Filename = getNewFilename() + ext

		boardExists := gcsql.DoesBoardExistByID(
			gcutil.HackyStringToInt(request.FormValue("boardid")))
		if !boardExists {
			serverutil.ServeErrorPage(writer, "No boards have been created yet")
			return
		}
		filePath = path.Join(systemCritical.DocumentRoot, postBoard.Dir, "src", upload.Filename)
		thumbPath = path.Join(systemCritical.DocumentRoot, postBoard.Dir, "thumb", upload.ThumbnailPath("thumb"))
		catalogThumbPath = path.Join(systemCritical.DocumentRoot, postBoard.Dir, "thumb", upload.ThumbnailPath("catalog"))

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
				if post.IsTopPost {
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
	building.BuildBoards(false, postBoard.ID)
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

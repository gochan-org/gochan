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
	maxFormBytes  = 50000000
)

var (
	ErrorPostTooLong = errors.New("post is too long")
)

// MakePost is called when a user accesses /post. Parse form data, then insert and build
func MakePost(writer http.ResponseWriter, request *http.Request) {
	request.ParseMultipartForm(maxFormBytes)
	ip := gcutil.GetRealIP(request)
	errEv := gcutil.LogError(nil).
		Str("IP", ip)
	infoEv := gcutil.LogInfo().
		Str("IP", ip)
	defer func() {
		errEv.Discard()
		infoEv.Discard()
	}()
	var post gcsql.Post
	var formName string
	var nameCookie string
	var formEmail string

	systemCritical := config.GetSystemCriticalConfig()

	if request.Method == "GET" {
		infoEv.Msg("Invalid request (expected POST, not GET)")
		http.Redirect(writer, request, systemCritical.WebRoot, http.StatusFound)
		return
	}

	if request.FormValue("doappeal") != "" {
		handleAppeal(writer, request, errEv)
		return
	}

	wantsJSON := serverutil.IsRequestingJSON(request)
	post.IP = gcutil.GetRealIP(request)
	var err error
	threadidStr := request.FormValue("threadid")
	// to avoid potential hiccups, we'll just treat the "threadid" form field as the OP ID and convert it internally
	// to the real thread ID
	var opID int
	if threadidStr != "" {
		// post is a reply
		if opID, err = strconv.Atoi(threadidStr); err != nil {
			errEv.Err(err).
				Str("opIDstr", threadidStr).
				Caller().Msg("Invalid threadid value")
			serverutil.ServeError(writer, "Invalid form data (invalid threadid)", wantsJSON, map[string]interface{}{
				"threadid": threadidStr,
			})
			return
		}
		if opID > 0 {
			if post.ThreadID, err = gcsql.GetTopPostThreadID(opID); err != nil {
				errEv.Err(err).
					Int("opID", opID).
					Caller().Send()
				serverutil.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
					"opID": opID,
				})
			}
		}
	}

	boardidStr := request.FormValue("boardid")
	boardID, err := strconv.Atoi(boardidStr)
	if err != nil {
		errEv.Str("boardid", boardidStr).Caller().Msg("Invalid boardid value")
		serverutil.ServeError(writer, "Invalid form data (invalid boardid)", wantsJSON, map[string]interface{}{
			"boardid": boardidStr,
		})
		return
	}
	postBoard, err := gcsql.GetBoardFromID(boardID)
	if err != nil {
		errEv.Err(err).Caller().
			Int("boardid", boardID).
			Msg("Unable to get board info")
		serverutil.ServeError(writer, "Unable to get board info", wantsJSON, map[string]interface{}{
			"boardid": boardID,
		})
		return
	}
	boardConfig := config.GetBoardConfig(postBoard.Dir)

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
		errEv.
			Int("messageLength", len(post.MessageRaw)).
			Int("maxMessageLength", postBoard.MaxMessageLength).Send()
		serverutil.ServeError(writer, "Message is too long", wantsJSON, map[string]interface{}{
			"messageLength": len(post.MessageRaw),
			"boardid":       boardID,
		})
		return
	}

	if post.MessageRaw, err = ApplyWordFilters(post.MessageRaw, postBoard.Dir); err != nil {
		errEv.Err(err).Caller().Msg("Error formatting post")
		serverutil.ServeError(writer, "Error formatting post: "+err.Error(), wantsJSON, map[string]interface{}{
			"boardDir": postBoard.Dir,
		})
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
		errEv.Err(err).Caller().Str("boardDir", postBoard.Dir).Msg("Unable to check psot cooldown")
		serverutil.ServeError(writer, "Error checking post cooldown: "+err.Error(), wantsJSON, map[string]interface{}{
			"boardDir": postBoard.Dir,
		})
		return
	}
	if tooSoon {
		errEv.Int("delay", delay).Msg("Rejecting post (user must wait before making another post)")
		serverutil.ServeError(writer, "Please wait before making a new post", wantsJSON, nil)
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

	var upload *gcsql.Upload
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
		upload = &gcsql.Upload{
			OriginalFilename: html.EscapeString(handler.Filename),
		}

		if checkFilenameBan(upload, &post, postBoard, writer, request) {
			// If checkFilenameBan returns true, an error occured or the file was
			// rejected for having a banned filename, and the incident was logged either way
			return
		}

		data, err := io.ReadAll(file)
		if err != nil {
			gcutil.LogError(err).
				Str("IP", post.IP).
				Str("upload", "read").Send()
			serverutil.ServeErrorPage(writer, "Error while trying to read file: "+err.Error())
			return
		}
		defer file.Close()

		// Calculate image checksum
		upload.Checksum = fmt.Sprintf("%x", md5.Sum(data))
		if checkChecksumBan(upload, &post, postBoard, writer, request) {
			// If checkChecksumBan returns true, an error occured or the file was
			// rejected for having a banned checksum, and the incident was logged either way
			return
		}

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
				Str("filename", upload.Filename).
				Str("originalFilename", upload.OriginalFilename).
				Send()
			serverutil.ServeErrorPage(writer, fmt.Sprintf("Couldn't write file %q", upload.OriginalFilename))
			return
		}

		if ext == "webm" || ext == "mp4" {
			gcutil.LogInfo().
				Str("post", "withVideo").
				Str("IP", post.IP).
				Str("filename", handler.Filename).
				Str("referer", request.Referer()).Send()
			if post.IsTopPost {
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
						gcutil.LogError(err).
							Str("thumbPath", catalogThumbPath).
							Str("IP", post.IP).
							Msg("Couldn't generate catalog thumbnail")
						serverutil.ServeErrorPage(writer, "Couldn't generate catalog thumbnail: "+err.Error())
						return
					}
				} else {
					thumbnail = createImageThumbnail(img, postBoard.Dir, "reply")
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
				upload.ThumbnailWidth = img.Bounds().Max.X
				upload.ThumbnailHeight = img.Bounds().Max.Y
				if err := syscall.Symlink(filePath, thumbPath); err != nil {
					gcutil.LogError(err).
						Str("thumbPath", thumbPath).
						Str("IP", post.IP).
						Msg("Couldn't generate catalog thumbnail")
					serverutil.ServeErrorPage(writer, "Couldn't create thumbnail: "+err.Error())
					return
				}
				if post.IsTopPost {
					// Generate catalog thumbnail
					catalogThumbnail := createImageThumbnail(img, postBoard.Dir, "catalog")
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

	if err = post.Insert(emailCommand != "sage", postBoard.ID, false, false, false, false); err != nil {
		gcutil.LogError(err).
			Str("IP", post.IP).
			Str("sql", "postInsertion").
			Msg("Unable to insert post")
		if upload != nil {
			os.Remove(filePath)
			os.Remove(thumbPath)
			os.Remove(catalogThumbPath)
		}
		serverutil.ServeErrorPage(writer, "Unable to insert post: "+err.Error())
		return
	}

	if err = post.AttachFile(upload); err != nil {
		gcutil.LogError(err).
			Str("IP", post.IP).
			Str("sql", "postInsertion").
			Msg("Unable to attach upload to post")
		os.Remove(filePath)
		os.Remove(thumbPath)
		os.Remove(catalogThumbPath)
		serverutil.ServeErrorPage(writer, "Unable to attach upload: "+err.Error())
		return
	}

	// rebuild the board page
	building.BuildBoards(false, postBoard.ID)
	building.BuildFrontPage()

	if emailCommand == "noko" {
		if post.IsTopPost {
			http.Redirect(writer, request, systemCritical.WebRoot+postBoard.Dir+"/res/"+strconv.Itoa(post.ID)+".html", http.StatusFound)
		} else {
			topPost, _ := post.TopPostID()
			http.Redirect(writer, request, systemCritical.WebRoot+postBoard.Dir+"/res/"+strconv.Itoa(topPost)+".html#"+strconv.Itoa(post.ID), http.StatusFound)
		}
	} else {
		http.Redirect(writer, request, systemCritical.WebRoot+postBoard.Dir+"/", http.StatusFound)
	}
}

package posting

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"fmt"
	"html"
	"image"
	"io/ioutil"
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
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/serverutil"
)

const (
	yearInSeconds = 31536000
	errStdLogs    = gclog.LErrorLog | gclog.LStdLog
)

// MakePost is called when a user accesses /post. Parse form data, then insert and build
func MakePost(writer http.ResponseWriter, request *http.Request) {
	var maxMessageLength int
	var post gcsql.Post
	// domain := request.Host
	var formName string
	var nameCookie string
	var formEmail string

	if request.Method == "GET" {
		http.Redirect(writer, request, config.Config.SiteWebfolder, http.StatusFound)
		return
	}
	// fix new cookie domain for when you use a port number
	// domain = chopPortNumRegex.Split(domain, -1)[0]

	post.ParentID, _ = strconv.Atoi(request.FormValue("threadid"))
	post.BoardID, _ = strconv.Atoi(request.FormValue("boardid"))

	var emailCommand string
	formName = request.FormValue("postname")
	parsedName := gcutil.ParseName(formName)
	post.Name = parsedName["name"]
	post.Tripcode = parsedName["tripcode"]

	formEmail = request.FormValue("postemail")

	http.SetCookie(writer, &http.Cookie{Name: "email", Value: formEmail, MaxAge: yearInSeconds})

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
	var err *gcutil.GcError
	if maxMessageLength, err = gcsql.GetMaxMessageLength(post.BoardID); err != nil {
		serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
			"Error getting board info: ", err.Error()))
	}

	if len(post.MessageText) > maxMessageLength {
		serverutil.ServeErrorPage(writer, "Post body is too long")
		return
	}
	post.MessageHTML = FormatMessage(post.MessageText)
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
	http.SetCookie(writer, &http.Cookie{Name: "name", Value: nameCookie, MaxAge: yearInSeconds})
	http.SetCookie(writer, &http.Cookie{Name: "password", Value: password, MaxAge: yearInSeconds})

	post.IP = gcutil.GetRealIP(request)
	post.Timestamp = time.Now()
	// post.PosterAuthority = getStaffRank(request)
	post.Bumped = time.Now()
	post.Stickied = request.FormValue("modstickied") == "on"
	post.Locked = request.FormValue("modlocked") == "on"

	//post has no referrer, or has a referrer from a different domain, probably a spambot
	if !serverutil.ValidReferer(request) {
		gclog.Print(gclog.LAccessLog, "Rejected post from possible spambot @ "+post.IP)
		return
	}

	switch serverutil.CheckPostForSpam(post.IP, request.Header["User-Agent"][0], request.Referer(),
		post.Name, post.Email, post.MessageText) {
	case "discard":
		serverutil.ServeErrorPage(writer, "Your post looks like spam.")
		gclog.Print(gclog.LAccessLog, "Akismet recommended discarding post from: "+post.IP)
		return
	case "spam":
		serverutil.ServeErrorPage(writer, "Your post looks like spam.")
		gclog.Print(gclog.LAccessLog, "Akismet suggested post is spam from "+post.IP)
		return
	default:
	}

	file, handler, gErr := request.FormFile("imagefile")

	if gErr != nil || handler.Size == 0 {
		// no file was uploaded
		post.Filename = ""
		gclog.Printf(gclog.LAccessLog,
			"Receiving post from %s, referred from: %s", post.IP, request.Referer())
	} else {
		data, gErr := ioutil.ReadAll(file)
		if gErr != nil {
			serverutil.ServeErrorPage(writer,
				gclog.Print(gclog.LErrorLog, "Error while trying to read file: ", gErr.Error()))
			return
		}
		defer file.Close()
		post.FilenameOriginal = html.EscapeString(handler.Filename)
		filetype := gcutil.GetFileExtension(post.FilenameOriginal)
		thumbFiletype := strings.ToLower(filetype)
		if thumbFiletype == "gif" || thumbFiletype == "webm" {
			thumbFiletype = "jpg"
		}

		post.Filename = getNewFilename() + "." + gcutil.GetFileExtension(post.FilenameOriginal)
		boardExists, err := gcsql.DoesBoardExistByID(
			gcutil.HackyStringToInt(request.FormValue("boardid")))
		if err != nil {
			serverutil.ServeErrorPage(writer, "Server error: "+err.Error())
			return
		}
		if !boardExists {
			serverutil.ServeErrorPage(writer, "No boards have been created yet")
			return
		}
		var _board = gcsql.Board{}
		err = _board.PopulateData(gcutil.HackyStringToInt(request.FormValue("boardid")))
		if err != nil {
			serverutil.ServeErrorPage(writer, "Server error: "+err.Error())
			return
		}
		boardDir := _board.Dir
		filePath := path.Join(config.Config.DocumentRoot, "/"+boardDir+"/src/", post.Filename)
		thumbPath := path.Join(config.Config.DocumentRoot, "/"+boardDir+"/thumb/", strings.Replace(post.Filename, "."+filetype, "t."+thumbFiletype, -1))
		catalogThumbPath := path.Join(config.Config.DocumentRoot, "/"+boardDir+"/thumb/", strings.Replace(post.Filename, "."+filetype, "c."+thumbFiletype, -1))

		if gErr = ioutil.WriteFile(filePath, data, 0777); err != nil {
			gclog.Printf(gclog.LErrorLog, "Couldn't write file %q: %s", post.Filename, gErr.Error())
			serverutil.ServeErrorPage(writer, `Couldn't write file "`+post.FilenameOriginal+`"`)
			return
		}

		// Calculate image checksum
		post.FileChecksum = fmt.Sprintf("%x", md5.Sum(data))

		var allowsVids bool
		if allowsVids, err = gcsql.GetEmbedsAllowed(post.BoardID); err != nil {
			serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
				"Couldn't get board info: ", err.Error()))
			return
		}

		if filetype == "webm" {
			if !allowsVids || !config.Config.AllowVideoUploads {
				serverutil.ServeErrorPage(writer, gclog.Print(gclog.LAccessLog,
					"Video uploading is not currently enabled for this board."))
				os.Remove(filePath)
				return
			}

			gclog.Printf(gclog.LAccessLog, "Receiving post with video: %s from %s, referrer: %s",
				handler.Filename, post.IP, request.Referer())
			if post.ParentID == 0 {
				if err := createVideoThumbnail(filePath, thumbPath, config.Config.ThumbWidth); err != nil {
					serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
						"Error creating video thumbnail: ", err.Error()))
					return
				}
			} else {
				if err := createVideoThumbnail(filePath, thumbPath, config.Config.ThumbWidthReply); err != nil {
					serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
						"Error creating video thumbnail: ", err.Error()))
					return
				}
			}

			if err := createVideoThumbnail(filePath, catalogThumbPath, config.Config.ThumbWidthCatalog); err != nil {
				serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
					"Error creating video thumbnail: ", err.Error()))
				return
			}

			outputBytes, err := exec.Command("ffprobe", "-v", "quiet", "-show_format", "-show_streams", filePath).CombinedOutput()
			if err != nil {
				serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
					"Error getting video info: ", err.Error()))
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
				if post.ParentID == 0 {
					post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, "op")
				} else {
					post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, "reply")
				}
			}
		} else {
			// Attempt to load uploaded file with imaging library
			img, err := imaging.Open(filePath)
			if err != nil {
				os.Remove(filePath)
				gclog.Printf(gclog.LErrorLog, "Couldn't open uploaded file %q: %s", post.Filename, err.Error())
				serverutil.ServeErrorPage(writer, "Upload filetype not supported")
				return
			}
			// Get image filesize
			stat, err := os.Stat(filePath)
			if err != nil {
				serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
					"Couldn't get image filesize: "+err.Error()))
				return
			}
			post.Filesize = int(stat.Size())

			// Get image width and height, as well as thumbnail width and height
			post.ImageW = img.Bounds().Max.X
			post.ImageH = img.Bounds().Max.Y
			if post.ParentID == 0 {
				post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, "op")
			} else {
				post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, "reply")
			}

			gclog.Printf(gclog.LAccessLog, "Receiving post with image: %q from %s, referrer: %s",
				handler.Filename, post.IP, request.Referer())

			if request.FormValue("spoiler") == "on" {
				// If spoiler is enabled, symlink thumbnail to spoiler image
				if _, err := os.Stat(path.Join(config.Config.DocumentRoot, "spoiler.png")); err != nil {
					serverutil.ServeErrorPage(writer, "missing /spoiler.png")
					return
				}
				if err = syscall.Symlink(path.Join(config.Config.DocumentRoot, "spoiler.png"), thumbPath); err != nil {
					serverutil.ServeErrorPage(writer, err.Error())
					return
				}
			} else if config.Config.ThumbWidth >= post.ImageW && config.Config.ThumbHeight >= post.ImageH {
				// If image fits in thumbnail size, symlink thumbnail to original
				post.ThumbW = img.Bounds().Max.X
				post.ThumbH = img.Bounds().Max.Y
				if err := syscall.Symlink(filePath, thumbPath); err != nil {
					serverutil.ServeErrorPage(writer, err.Error())
					return
				}
			} else {
				var thumbnail image.Image
				var catalogThumbnail image.Image
				if post.ParentID == 0 {
					// If this is a new thread, generate thumbnail and catalog thumbnail
					thumbnail = createImageThumbnail(img, "op")
					catalogThumbnail = createImageThumbnail(img, "catalog")
					if err = imaging.Save(catalogThumbnail, catalogThumbPath); err != nil {
						serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
							"Couldn't generate catalog thumbnail: ", err.Error()))
						return
					}
				} else {
					thumbnail = createImageThumbnail(img, "reply")
				}
				if err = imaging.Save(thumbnail, thumbPath); err != nil {
					serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
						"Couldn't save thumbnail: ", err.Error()))
					return
				}
			}
		}
	}

	if strings.TrimSpace(post.MessageText) == "" && post.Filename == "" {
		serverutil.ServeErrorPage(writer, "Post must contain a message if no image is uploaded.")
		return
	}

	postDelay, _ := gcsql.SinceLastPost(post.ID)
	if postDelay > -1 {
		if post.ParentID == 0 && postDelay < config.Config.NewThreadDelay {
			serverutil.ServeErrorPage(writer, "Please wait before making a new thread.")
			return
		} else if post.ParentID > 0 && postDelay < config.Config.ReplyDelay {
			serverutil.ServeErrorPage(writer, "Please wait before making a reply.")
			return
		}
	}

	banStatus, err := getBannedStatus(request)
	if err != nil && !gcutil.CompareErrors(err, sql.ErrNoRows) {
		serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
			"Error getting banned status: ", err.Error()))
		return
	}

	boards, _ := gcsql.GetAllBoards()

	postBoard, _ := gcsql.GetBoardFromID(post.BoardID)
	if banStatus != nil && banStatus.IsBanned(postBoard.Dir) {
		var banpageBuffer bytes.Buffer

		if err = gcutil.MinifyTemplate(gctemplates.Banpage, map[string]interface{}{
			"config": config.Config, "ban": banStatus, "banBoards": boards[post.BoardID-1].Dir,
		}, writer, "text/html"); err != nil {
			serverutil.ServeErrorPage(writer,
				gclog.Print(gclog.LErrorLog, "Error minifying page: ", err.Error()))
			return
		}
		writer.Write(banpageBuffer.Bytes())
		return
	}

	post.Sanitize()

	if config.Config.UseCaptcha {
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

	if err = gcsql.InsertPost(&post, emailCommand != "sage"); err != nil {
		serverutil.ServeErrorPage(writer,
			gclog.Print(gclog.LErrorLog, "Error inserting post: ", err.Error()))
		return
	}

	// rebuild the board page
	building.BuildBoards(false, post.BoardID)
	building.BuildFrontPage()

	if emailCommand == "noko" {
		if post.ParentID < 1 {
			http.Redirect(writer, request, config.Config.SiteWebfolder+postBoard.Dir+"/res/"+strconv.Itoa(post.ID)+".html", http.StatusFound)
		} else {
			http.Redirect(writer, request, config.Config.SiteWebfolder+postBoard.Dir+"/res/"+strconv.Itoa(post.ParentID)+".html#"+strconv.Itoa(post.ID), http.StatusFound)
		}
	} else {
		http.Redirect(writer, request, config.Config.SiteWebfolder+postBoard.Dir+"/", http.StatusFound)
	}
}

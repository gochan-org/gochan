// functions for handling posting, uploading, and bans

package main

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"image"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aquilax/tripcode"
	"github.com/disintegration/imaging"
)

const (
	gt            = "&gt;"
	yearInSeconds = 31536000
)

var (
	allSections       []BoardSection
	allBoards         []Board
	tempPosts         []Post
	tempCleanerTicker *time.Ticker
)

// bumps the given thread on the given board and returns true if there were no errors
func bumpThread(postID, boardID int) error {
	_, err := execSQL("UPDATE DBPREFIXposts SET bumped = ? WHERE id = ? AND boardid = ?",
		time.Now(), postID, boardID,
	)

	return err
}

// Checks check poster's name/tripcode/file checksum (from Post post) for banned status
// returns ban table if the user is banned or errNotBanned if they aren't
func getBannedStatus(request *http.Request) (*BanInfo, error) {
	var banEntry BanInfo

	formName := request.FormValue("postname")
	var tripcode string
	if formName != "" {
		parsedName := parseName(formName)
		tripcode += parsedName["name"]
		if tc, ok := parsedName["tripcode"]; ok {
			tripcode += "!" + tc
		}
	}
	ip := getRealIP(request)

	var filename string
	var checksum string
	file, fileHandler, err := request.FormFile("imagefile")
	defer closeHandle(file)
	if err == nil {
		html.EscapeString(fileHandler.Filename)
		if data, err2 := ioutil.ReadAll(file); err2 == nil {
			checksum = fmt.Sprintf("%x", md5.Sum(data))
		}
	}

	in := []interface{}{ip}
	query := "SELECT id,ip,name,boards,timestamp,expires,permaban,reason,type,appeal_at,can_appeal FROM DBPREFIXbanlist WHERE ip = ? "

	if tripcode != "" {
		in = append(in, tripcode)
		query += "OR name = ? "
	}
	if filename != "" {
		in = append(in, filename)
		query += "OR filename = ? "
	}
	if checksum != "" {
		in = append(in, checksum)
		query += "OR file_checksum = ? "
	}
	query += " ORDER BY id DESC LIMIT 1"

	err = queryRowSQL(query, in, []interface{}{
		&banEntry.ID, &banEntry.IP, &banEntry.Name, &banEntry.Boards, &banEntry.Timestamp,
		&banEntry.Expires, &banEntry.Permaban, &banEntry.Reason, &banEntry.Type,
		&banEntry.AppealAt, &banEntry.CanAppeal},
	)
	return &banEntry, err
}

func isBanned(ban *BanInfo, board string) bool {
	if ban.Boards == "" && (ban.Expires.After(time.Now()) || ban.Permaban) {
		return true
	}
	boardsArr := strings.Split(ban.Boards, ",")
	for _, b := range boardsArr {
		if b == board && (ban.Expires.After(time.Now()) || ban.Permaban) {
			return true
		}
	}

	return false
}

func sinceLastPost(post *Post) int {
	var lastPostTime time.Time
	if err := queryRowSQL("SELECT timestamp FROM DBPREFIXposts WHERE ip = ? ORDER BY timestamp DESC LIMIT 1",
		[]interface{}{post.IP},
		[]interface{}{&lastPostTime},
	); err == sql.ErrNoRows {
		// no posts by that IP.
		return -1
	}
	return int(time.Since(lastPostTime).Seconds())
}

func createImageThumbnail(imageObj image.Image, size string) image.Image {
	var thumbWidth int
	var thumbHeight int

	switch size {
	case "op":
		thumbWidth = config.ThumbWidth
		thumbHeight = config.ThumbHeight
	case "reply":
		thumbWidth = config.ThumbWidth_reply
		thumbHeight = config.ThumbHeight_reply
	case "catalog":
		thumbWidth = config.ThumbWidth_catalog
		thumbHeight = config.ThumbHeight_catalog
	}
	oldRect := imageObj.Bounds()
	if thumbWidth >= oldRect.Max.X && thumbHeight >= oldRect.Max.Y {
		return imageObj
	}

	thumbW, thumbH := getThumbnailSize(oldRect.Max.X, oldRect.Max.Y, size)
	imageObj = imaging.Resize(imageObj, thumbW, thumbH, imaging.CatmullRom) // resize to 600x400 px using CatmullRom cubic filter
	return imageObj
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
func getThumbnailSize(w int, h int, size string) (newWidth int, newHeight int) {
	var thumbWidth int
	var thumbHeight int

	switch {
	case size == "op":
		thumbWidth = config.ThumbWidth
		thumbHeight = config.ThumbHeight
	case size == "reply":
		thumbWidth = config.ThumbWidth_reply
		thumbHeight = config.ThumbHeight_reply
	case size == "catalog":
		thumbWidth = config.ThumbWidth_catalog
		thumbHeight = config.ThumbHeight_catalog
	}
	if w == h {
		newWidth = thumbWidth
		newHeight = thumbHeight
	} else {
		var percent float32
		if w > h {
			percent = float32(thumbWidth) / float32(w)
		} else {
			percent = float32(thumbHeight) / float32(h)
		}
		newWidth = int(float32(w) * percent)
		newHeight = int(float32(h) * percent)
	}
	return
}

func parseName(name string) map[string]string {
	parsed := make(map[string]string)
	if !strings.Contains(name, "#") {
		parsed["name"] = name
		parsed["tripcode"] = ""
	} else if strings.Index(name, "#") == 0 {
		parsed["tripcode"] = tripcode.Tripcode(name[1:])
	} else if strings.Index(name, "#") > 0 {
		postNameArr := strings.SplitN(name, "#", 2)
		parsed["name"] = postNameArr[0]
		parsed["tripcode"] = tripcode.Tripcode(postNameArr[1])
	}
	return parsed
}

// inserts prepared post object into the SQL table so that it can be rendered
func insertPost(post *Post, bump bool) error {
	queryStr := "INSERT INTO DBPREFIXposts " +
		"(boardid,parentid,name,tripcode,email,subject,message,message_raw,password,filename,filename_original,file_checksum,filesize,image_w,image_h,thumb_w,thumb_h,ip,tag,timestamp,autosage,deleted_timestamp,bumped,stickied,locked,reviewed)" +
		"VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

	result, err := execSQL(queryStr,
		post.BoardID, post.ParentID, post.Name, post.Tripcode, post.Email,
		post.Subject, post.MessageHTML, post.MessageText, post.Password,
		post.Filename, post.FilenameOriginal, post.FileChecksum, post.Filesize,
		post.ImageW, post.ImageH, post.ThumbW, post.ThumbH, post.IP, post.Capcode,
		post.Timestamp, post.Autosage, post.DeletedTimestamp, post.Bumped,
		post.Stickied, post.Locked, post.Reviewed)
	if err != nil {
		return err
	}

	switch config.DBtype {
	case "mysql":
		var postID int64
		postID, err = result.LastInsertId()
		post.ID = int(postID)
	case "postgres":
		err = queryRowSQL("SELECT currval(pg_get_serial_sequence('DBPREFIXposts','id'))", nil, []interface{}{&post.ID})
	case "sqlite3":
		err = queryRowSQL("SELECT LAST_INSERT_ROWID()", nil, []interface{}{&post.ID})
	}

	// Bump parent post if requested.
	if err != nil && post.ParentID != 0 && bump {
		err = bumpThread(post.ParentID, post.BoardID)
	}
	return err
}

// called when a user accesses /post. Parse form data, then insert and build
func makePost(writer http.ResponseWriter, request *http.Request) {
	var maxMessageLength int
	var post Post
	// domain := request.Host
	var formName string
	var nameCookie string
	var formEmail string

	if request.Method == "GET" {
		http.Redirect(writer, request, config.SiteWebfolder, http.StatusFound)
		return
	}
	// fix new cookie domain for when you use a port number
	// domain = chopPortNumRegex.Split(domain, -1)[0]

	post.ParentID, _ = strconv.Atoi(request.FormValue("threadid"))
	post.BoardID, _ = strconv.Atoi(request.FormValue("boardid"))

	var emailCommand string
	formName = request.FormValue("postname")
	parsedName := parseName(formName)
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

	if err := queryRowSQL("SELECT max_message_length from DBPREFIXboards WHERE id = ?",
		[]interface{}{post.BoardID},
		[]interface{}{&maxMessageLength},
	); err != nil {
		serveErrorPage(writer, gclog.Print(lErrorLog,
			"Error getting board info: ", err.Error()))
	}

	if len(post.MessageText) > maxMessageLength {
		serveErrorPage(writer, "Post body is too long")
		return
	}
	post.MessageHTML = formatMessage(post.MessageText)
	password := request.FormValue("postpassword")
	if password == "" {
		password = randomString(8)
	}
	post.Password = md5Sum(password)

	// Reverse escapes
	nameCookie = strings.Replace(formName, "&amp;", "&", -1)
	nameCookie = strings.Replace(nameCookie, "\\&#39;", "'", -1)
	nameCookie = strings.Replace(url.QueryEscape(nameCookie), "+", "%20", -1)

	// add name and email cookies that will expire in a year (31536000 seconds)
	http.SetCookie(writer, &http.Cookie{Name: "name", Value: nameCookie, MaxAge: yearInSeconds})
	http.SetCookie(writer, &http.Cookie{Name: "password", Value: password, MaxAge: yearInSeconds})

	post.IP = getRealIP(request)
	post.Timestamp = time.Now()
	// post.PosterAuthority = getStaffRank(request)
	post.Bumped = time.Now()
	post.Stickied = request.FormValue("modstickied") == "on"
	post.Locked = request.FormValue("modlocked") == "on"

	//post has no referrer, or has a referrer from a different domain, probably a spambot
	if !validReferrer(request) {
		gclog.Print(lAccessLog, "Rejected post from possible spambot @ "+post.IP)
		return
	}

	switch checkPostForSpam(post.IP, request.Header["User-Agent"][0], request.Referer(),
		post.Name, post.Email, post.MessageText) {
	case "discard":
		serveErrorPage(writer, "Your post looks like spam.")
		gclog.Print(lAccessLog, "Akismet recommended discarding post from: "+post.IP)
		return
	case "spam":
		serveErrorPage(writer, "Your post looks like spam.")
		gclog.Print(lAccessLog, "Akismet suggested post is spam from "+post.IP)
		return
	default:
	}

	file, handler, err := request.FormFile("imagefile")
	defer closeHandle(file)

	if err != nil || handler.Size == 0 {
		// no file was uploaded
		post.Filename = ""
		gclog.Printf(lAccessLog, "Receiving post from %s, referred from: %s", post.IP, request.Referer())
	} else {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			serveErrorPage(writer, gclog.Print(lErrorLog, "Error while trying to read file: ", err.Error()))
			return
		}

		post.FilenameOriginal = html.EscapeString(handler.Filename)
		filetype := getFileExtension(post.FilenameOriginal)
		thumbFiletype := strings.ToLower(filetype)
		if thumbFiletype == "gif" || thumbFiletype == "webm" {
			thumbFiletype = "jpg"
		}

		post.Filename = getNewFilename() + "." + getFileExtension(post.FilenameOriginal)
		boardArr, _ := getBoardArr(map[string]interface{}{"id": request.FormValue("boardid")}, "")
		if len(boardArr) == 0 {
			serveErrorPage(writer, "No boards have been created yet")
			return
		}
		_boardDir, _ := getBoardArr(map[string]interface{}{"id": request.FormValue("boardid")}, "")
		boardDir := _boardDir[0].Dir
		filePath := path.Join(config.DocumentRoot, "/"+boardDir+"/src/", post.Filename)
		thumbPath := path.Join(config.DocumentRoot, "/"+boardDir+"/thumb/", strings.Replace(post.Filename, "."+filetype, "t."+thumbFiletype, -1))
		catalogThumbPath := path.Join(config.DocumentRoot, "/"+boardDir+"/thumb/", strings.Replace(post.Filename, "."+filetype, "c."+thumbFiletype, -1))

		if err = ioutil.WriteFile(filePath, data, 0777); err != nil {
			gclog.Printf(lErrorLog, "Couldn't write file %q: %s", post.Filename, err.Error())
			serveErrorPage(writer, `Couldn't write file "`+post.FilenameOriginal+`"`)
			return
		}

		// Calculate image checksum
		post.FileChecksum = fmt.Sprintf("%x", md5.Sum(data))

		var allowsVids bool
		if err = queryRowSQL("SELECT embeds_allowed FROM DBPREFIXboards WHERE id = ? LIMIT 1",
			[]interface{}{post.BoardID},
			[]interface{}{&allowsVids},
		); err != nil {
			serveErrorPage(writer, gclog.Print(lErrorLog,
				"Couldn't get board info: ", err.Error()))
			return
		}

		if filetype == "webm" {
			if !allowsVids || !config.AllowVideoUploads {
				serveErrorPage(writer, gclog.Print(lAccessLog,
					"Video uploading is not currently enabled for this board."))
				os.Remove(filePath)
				return
			}

			gclog.Printf(lAccessLog, "Receiving post with video: %s from %s, referrer: %s",
				handler.Filename, post.IP, request.Referer())
			if post.ParentID == 0 {
				if err := createVideoThumbnail(filePath, thumbPath, config.ThumbWidth); err != nil {
					serveErrorPage(writer, gclog.Print(lErrorLog,
						"Error creating video thumbnail: ", err.Error()))
					return
				}
			} else {
				if err := createVideoThumbnail(filePath, thumbPath, config.ThumbWidth_reply); err != nil {
					serveErrorPage(writer, gclog.Print(lErrorLog,
						"Error creating video thumbnail: ", err.Error()))
					return
				}
			}

			if err := createVideoThumbnail(filePath, catalogThumbPath, config.ThumbWidth_catalog); err != nil {
				serveErrorPage(writer, gclog.Print(lErrorLog,
					"Error creating video thumbnail: ", err.Error()))
				return
			}

			outputBytes, err := exec.Command("ffprobe", "-v", "quiet", "-show_format", "-show_streams", filePath).CombinedOutput()
			if err != nil {
				serveErrorPage(writer, gclog.Print(lErrorLog,
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
				gclog.Printf(lErrorLog, "Couldn't open uploaded file %q: %s", post.Filename, err.Error())
				serveErrorPage(writer, "Upload filetype not supported")
				return
			}
			// Get image filesize
			stat, err := os.Stat(filePath)
			if err != nil {
				serveErrorPage(writer, gclog.Print(lErrorLog,
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

			gclog.Printf(lAccessLog, "Receiving post with image: %q from %s, referrer: %s",
				handler.Filename, post.IP, request.Referer())

			if request.FormValue("spoiler") == "on" {
				// If spoiler is enabled, symlink thumbnail to spoiler image
				if _, err := os.Stat(path.Join(config.DocumentRoot, "spoiler.png")); err != nil {
					serveErrorPage(writer, "missing /spoiler.png")
					return
				}
				if err = syscall.Symlink(path.Join(config.DocumentRoot, "spoiler.png"), thumbPath); err != nil {
					serveErrorPage(writer, err.Error())
					return
				}
			} else if config.ThumbWidth >= post.ImageW && config.ThumbHeight >= post.ImageH {
				// If image fits in thumbnail size, symlink thumbnail to original
				post.ThumbW = img.Bounds().Max.X
				post.ThumbH = img.Bounds().Max.Y
				if err := syscall.Symlink(filePath, thumbPath); err != nil {
					serveErrorPage(writer, err.Error())
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
						serveErrorPage(writer, gclog.Print(lErrorLog,
							"Couldn't generate catalog thumbnail: ", err.Error()))
						return
					}
				} else {
					thumbnail = createImageThumbnail(img, "reply")
				}
				if err = imaging.Save(thumbnail, thumbPath); err != nil {
					serveErrorPage(writer, gclog.Print(lErrorLog,
						"Couldn't save thumbnail: ", err.Error()))
					return
				}
			}
		}
	}

	if strings.TrimSpace(post.MessageText) == "" && post.Filename == "" {
		serveErrorPage(writer, "Post must contain a message if no image is uploaded.")
		return
	}

	postDelay := sinceLastPost(&post)
	if postDelay > -1 {
		if post.ParentID == 0 && postDelay < config.NewThreadDelay {
			serveErrorPage(writer, "Please wait before making a new thread.")
			return
		} else if post.ParentID > 0 && postDelay < config.ReplyDelay {
			serveErrorPage(writer, "Please wait before making a reply.")
			return
		}
	}

	banStatus, err := getBannedStatus(request)
	if err != nil && err != sql.ErrNoRows {
		serveErrorPage(writer, gclog.Print(lErrorLog,
			"Error getting banned status: ", err.Error()))
		return
	}

	boards, _ := getBoardArr(nil, "")

	if isBanned(banStatus, boards[post.BoardID-1].Dir) {
		var banpageBuffer bytes.Buffer

		if err = minifyTemplate(banpageTmpl, map[string]interface{}{
			"config": config, "ban": banStatus, "banBoards": boards[post.BoardID-1].Dir,
		}, writer, "text/html"); err != nil {
			serveErrorPage(writer, gclog.Print(lErrorLog, "Error minifying page: ", err.Error()))
			return
		}
		writer.Write(banpageBuffer.Bytes())
		return
	}

	post.Sanitize()

	if config.UseCaptcha {
		captchaID := request.FormValue("captchaid")
		captchaAnswer := request.FormValue("captchaanswer")
		if captchaID == "" && captchaAnswer == "" {
			// browser isn't using JS, save post data to tempPosts and show captcha
			request.Form.Add("temppostindex", strconv.Itoa(len(tempPosts)))
			request.Form.Add("emailcmd", emailCommand)
			tempPosts = append(tempPosts, post)
			serveCaptcha(writer, request)
			return
		}
	}

	if err = insertPost(&post, emailCommand != "sage"); err != nil {
		serveErrorPage(writer, gclog.Print(lErrorLog, "Error inserting post: ", err.Error()))
		return
	}

	// rebuild the board page
	buildBoards(post.BoardID)
	buildFrontPage()

	if emailCommand == "noko" {
		if post.ParentID < 1 {
			http.Redirect(writer, request, config.SiteWebfolder+boards[post.BoardID-1].Dir+"/res/"+strconv.Itoa(post.ID)+".html", http.StatusFound)
		} else {
			http.Redirect(writer, request, config.SiteWebfolder+boards[post.BoardID-1].Dir+"/res/"+strconv.Itoa(post.ParentID)+".html#"+strconv.Itoa(post.ID), http.StatusFound)
		}
	} else {
		http.Redirect(writer, request, config.SiteWebfolder+boards[post.BoardID-1].Dir+"/", http.StatusFound)
	}
}

func tempCleaner() {
	for {
		select {
		case <-tempCleanerTicker.C:
			for p, post := range tempPosts {
				if !time.Now().After(post.Timestamp.Add(time.Minute * 5)) {
					continue
				}
				// temporary post is >= 5 minutes, time to prune it
				tempPosts[p] = tempPosts[len(tempPosts)-1]
				tempPosts = tempPosts[:len(tempPosts)-1]
				if post.FilenameOriginal == "" {
					continue
				}
				var board Board
				err := board.PopulateData(post.BoardID, "")
				if err != nil {
					continue
				}

				fileSrc := path.Join(config.DocumentRoot, board.Dir, "src", post.FilenameOriginal)
				if err = os.Remove(fileSrc); err != nil {
					gclog.Printf(lErrorLog|lStdLog,
						"Error pruning temporary upload for %q: %s", fileSrc, err.Error())
				}

				thumbSrc := getThumbnailPath("thread", fileSrc)
				if err = os.Remove(thumbSrc); err != nil {
					gclog.Printf(lErrorLog|lStdLog,
						"Error pruning temporary upload for %q: %s", thumbSrc, err.Error())
				}

				if post.ParentID == 0 {
					catalogSrc := getThumbnailPath("catalog", fileSrc)
					if err = os.Remove(catalogSrc); err != nil {
						gclog.Printf(lErrorLog|lStdLog,
							"Error pruning temporary upload for %s: %s", catalogSrc, err.Error())
					}
				}
			}
		}
	}
}

func formatMessage(message string) string {
	message = msgfmtr.Compile(message)
	// prepare each line to be formatted
	postLines := strings.Split(message, "<br>")
	for i, line := range postLines {
		trimmedLine := strings.TrimSpace(line)
		lineWords := strings.Split(trimmedLine, " ")
		isGreentext := false // if true, append </span> to end of line
		for w, word := range lineWords {
			if strings.LastIndex(word, gt+gt) == 0 {
				//word is a backlink
				if _, err := strconv.Atoi(word[8:]); err == nil {
					// the link is in fact, a valid int
					var boardDir string
					var linkParent int

					if err = queryRowSQL("SELECT dir,parentid FROM DBPREFIXposts,DBPREFIXboards WHERE DBPREFIXposts.id = ?",
						[]interface{}{word[8:]},
						[]interface{}{&boardDir, &linkParent},
					); err != nil {
						gclog.Print(lErrorLog, "Error getting board information for backlink: ", err.Error())
					}

					// get post board dir
					if boardDir == "" {
						lineWords[w] = "<a href=\"javascript:;\"><strike>" + word + "</strike></a>"
					} else if linkParent == 0 {
						lineWords[w] = "<a href=\"" + config.SiteWebfolder + boardDir + "/res/" + word[8:] + ".html\" class=\"postref\">" + word + "</a>"
					} else {
						lineWords[w] = "<a href=\"" + config.SiteWebfolder + boardDir + "/res/" + strconv.Itoa(linkParent) + ".html#" + word[8:] + "\" class=\"postref\">" + word + "</a>"
					}
				}
			} else if strings.Index(word, gt) == 0 && w == 0 {
				// word is at the beginning of a line, and is greentext
				isGreentext = true
				lineWords[w] = "<span class=\"greentext\">" + word
			}
		}
		line = strings.Join(lineWords, " ")
		if isGreentext {
			line += "</span>"
		}
		postLines[i] = line
	}
	return strings.Join(postLines, "<br />")
}

func bannedForever(ban *BanInfo) bool {
	return ban.Permaban && !ban.CanAppeal && ban.Type == 3 && ban.Boards == ""
}

func banHandler(writer http.ResponseWriter, request *http.Request) {
	appealMsg := request.FormValue("appealmsg")
	banStatus, err := getBannedStatus(request)

	if appealMsg != "" {
		if bannedForever(banStatus) {
			fmt.Fprint(writer, "No.")
			return
		}
		escapedMsg := html.EscapeString(appealMsg)
		if _, err = execSQL("INSERT INTO DBPREFIXappeals (ban,message) VALUES(?,?)",
			banStatus.ID, escapedMsg,
		); err != nil {
			serveErrorPage(writer, err.Error())
		}
		fmt.Fprint(writer,
			"Appeal sent. It will (hopefully) be read by a staff member. check "+config.SiteWebfolder+"banned occasionally for a response",
		)
		return
	}

	if err != nil && err != sql.ErrNoRows {
		serveErrorPage(writer, gclog.Print(lErrorLog,
			"Error getting banned status:", err.Error()))
		return
	}

	if err = minifyTemplate(banpageTmpl, map[string]interface{}{
		"config": config, "ban": banStatus, "banBoards": banStatus.Boards, "post": Post{},
	}, writer, "text/html"); err != nil {
		serveErrorPage(writer, gclog.Print(lErrorLog,
			"Error minifying page template: ", err.Error()))
		return
	}
}
